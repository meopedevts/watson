package reviewer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/meopedevts/watson/config"
	"github.com/meopedevts/watson/internal/git"
	ghpkg "github.com/meopedevts/watson/internal/github"
)

// reviewRecord stores state for a PR that has been reviewed at least once.
type reviewRecord struct {
	PR         ghpkg.PullRequest
	Review     string    // the review text Watson posted
	ReviewedAt time.Time // when it was posted
}

// reReviewContext carries the context for a mention-triggered re-review.
type reReviewContext struct {
	PreviousReview string
	MentionComment string
}

// Reviewer orchestrates the full PR review pipeline.
// All external dependencies are injected for testability.
type Reviewer struct {
	cfg       *config.Config
	executor  ghpkg.Executor
	processed *sync.Map
	logger    *slog.Logger
}

// NewReviewer constructs a Reviewer with all required dependencies.
func NewReviewer(cfg *config.Config, exec ghpkg.Executor, logger *slog.Logger) *Reviewer {
	return &Reviewer{
		cfg:       cfg,
		executor:  exec,
		processed: &sync.Map{},
		logger:    logger,
	}
}

// ProcessPendingPRs is called on every ticker tick. It lists pending PRs,
// launches a goroutine for each one, then checks reviewed PRs for new @mentions.
// Per-PR errors are logged but do not abort the loop.
func (r *Reviewer) ProcessPendingPRs(ctx context.Context) error {
	prs, err := ghpkg.ListPendingPRs(ctx, r.executor, r.processed)
	if err != nil {
		return fmt.Errorf("list PRs: %w", err)
	}

	if len(prs) == 0 {
		r.logger.Info("no pending PRs found")
	} else {
		r.logger.Info("processing PRs", "count", len(prs))
		var wg sync.WaitGroup
		for _, pr := range prs {
			wg.Add(1)
			go func(pr ghpkg.PullRequest) {
				defer wg.Done()
				if err := r.ReviewPR(ctx, pr); err != nil {
					r.logger.Error("review failed", "pr", pr.Number, "title", pr.Title, "err", err)
				}
			}(pr)
		}
		wg.Wait()
	}

	r.processMentionedPRs(ctx)
	return nil
}

// processMentionedPRs checks all previously-reviewed PRs for new @mentions.
// When a mention is found after the last review, a re-review is triggered with
// the updated diff, the previous review text, and the triggering comment as context.
func (r *Reviewer) processMentionedPRs(ctx context.Context) {
	r.processed.Range(func(key, value any) bool {
		rec, ok := value.(reviewRecord)
		if !ok {
			return true
		}

		comments, err := ghpkg.FetchPRComments(ctx, r.executor, rec.PR.Number, rec.PR.Repository.NameWithOwner)
		if err != nil {
			r.logger.Warn("failed to fetch comments for mention check", "pr", rec.PR.Number, "err", err)
			return true
		}

		mention := ghpkg.FindMentionAfter(comments, r.cfg.GitHubReviewerUsername, rec.ReviewedAt)
		if mention == nil {
			return true
		}

		r.logger.Info("mention detected, triggering re-review", "pr", rec.PR.Number, "author", mention.Author.Login)
		if err := r.reviewPRInternal(ctx, rec.PR, &reReviewContext{
			PreviousReview: rec.Review,
			MentionComment: mention.Body,
		}); err != nil {
			r.logger.Error("re-review failed", "pr", rec.PR.Number, "err", err)
		}

		return true
	})
}

// ReviewPR runs the full review pipeline for a single PR (first-time review).
func (r *Reviewer) ReviewPR(ctx context.Context, pr ghpkg.PullRequest) error {
	return r.reviewPRInternal(ctx, pr, nil)
}

// reviewPRInternal is the shared review implementation used for both first-time
// reviews and mention-triggered re-reviews. When reCtx is non-nil the prompt
// includes the previous review text and the triggering comment.
//
//  1. Clone repository to a temp directory (cleaned up via defer)
//  2. Compute diff against origin/<base>
//  3. Build prompt and call Claude
//  4. Check for merge conflicts; append warning if found
//  5. Post comment (or print if dry-run)
//  6. Update processed record — ONLY on full success
func (r *Reviewer) reviewPRInternal(ctx context.Context, pr ghpkg.PullRequest, reCtx *reReviewContext) error {
	r.logger.Info("starting review", "pr", pr.Number, "title", pr.Title, "repo", pr.Repository.NameWithOwner)

	refs, err := ghpkg.FetchPRRefs(ctx, r.executor, pr.Number, pr.Repository.NameWithOwner)
	if err != nil {
		return fmt.Errorf("PR #%d fetch refs: %w", pr.Number, err)
	}

	cloneDir, err := git.CloneRepo(ctx, r.executor, pr.CloneURL(r.cfg.GitSSHHost), refs.HeadRefName, r.cfg.RepoBaseDir)
	if err != nil {
		return fmt.Errorf("PR #%d clone: %w", pr.Number, err)
	}
	defer func() {
		if err := os.RemoveAll(cloneDir); err != nil {
			r.logger.Warn("failed to remove clone dir", "dir", cloneDir, "err", err)
		}
	}()

	diffResult, err := git.GetDiff(ctx, r.executor, cloneDir, refs.BaseRefName)
	if err != nil {
		return fmt.Errorf("PR #%d diff: %w", pr.Number, err)
	}

	promptCtx := PromptContext{
		PR:    pr,
		Refs:  refs,
		Diff:  diffResult.Diff,
		Stats: diffResult.Stats,
		Note:  diffResult.Note,
	}
	if reCtx != nil {
		promptCtx.IsReReview = true
		promptCtx.PreviousReview = reCtx.PreviousReview
		promptCtx.MentionComment = reCtx.MentionComment
	}

	review, err := r.runClaude(ctx, BuildPrompt(promptCtx))
	if err != nil {
		return fmt.Errorf("PR #%d claude review: %w", pr.Number, err)
	}

	conflictFiles, err := git.CheckConflicts(ctx, r.executor, cloneDir, refs.BaseRefName)
	if err != nil {
		r.logger.Warn("conflict check failed, skipping conflict warning", "pr", pr.Number, "err", err)
	} else if len(conflictFiles) > 0 {
		r.logger.Info("merge conflicts detected", "pr", pr.Number, "files", conflictFiles)
		review += buildConflictWarning(conflictFiles)
	}

	if err := r.postComment(ctx, pr.Number, pr.Repository.NameWithOwner, review); err != nil {
		return fmt.Errorf("PR #%d post comment: %w", pr.Number, err)
	}

	r.processed.Store(pr.Number, reviewRecord{
		PR:         pr,
		Review:     review,
		ReviewedAt: time.Now(),
	})
	r.logger.Info("review completed", "pr", pr.Number)
	return nil
}

// runClaude invokes the claude CLI with the prompt piped via stdin.
// Using stdin avoids OS argument-length limits for large diffs and
// prevents shell-escaping issues with special characters.
func (r *Reviewer) runClaude(ctx context.Context, prompt string) (string, error) {
	out, err := r.executor.RunWithStdin(ctx, prompt, "claude", "--model", r.cfg.ClaudeModel, "--print")
	if err != nil {
		return "", fmt.Errorf("claude: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// buildConflictWarning formats a conflict notice with the list of affected files.
func buildConflictWarning(files []string) string {
	var sb strings.Builder
	sb.WriteString("\n\n---\n**AVISO DE CONFLITO:** Este PR possui conflitos de merge com a branch base.\n\nArquivos com conflito:\n")
	for _, f := range files {
		sb.WriteString("- `")
		sb.WriteString(f)
		sb.WriteString("`\n")
	}
	sb.WriteString("\nResolva os conflitos antes de fazer o merge.")
	return sb.String()
}

// postComment posts the review body as a GitHub PR comment.
// In dry-run mode it writes to stdout instead of calling gh.
func (r *Reviewer) postComment(ctx context.Context, prNumber int, repo, body string) error {
	if r.cfg.DryRun {
		fmt.Printf("\n=== DRY RUN: Review for PR #%d ===\n%s\n\n", prNumber, body)
		return nil
	}

	_, err := r.executor.Run(ctx,
		"gh", "pr", "comment", fmt.Sprintf("%d", prNumber),
		"--repo", repo,
		"--body", body,
	)
	if err != nil {
		return fmt.Errorf("gh pr comment: %w", err)
	}
	return nil
}
