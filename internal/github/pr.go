package github

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Repository holds the repo metadata returned by gh search prs.
type Repository struct {
	NameWithOwner string `json:"nameWithOwner"` // e.g. "owner/repo"
}

// Label represents a GitHub PR label.
type Label struct {
	Name string `json:"name"`
}

// Commit represents a single commit in a PR.
type Commit struct {
	MessageHeadline string `json:"messageHeadline"`
}

// PullRequest represents a single open PR returned by gh search prs.
// HeadRefName is not available from search — use FetchPRRefs to get it.
type PullRequest struct {
	Number     int        `json:"number"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	Repository Repository `json:"repository"`
	Labels     []Label    `json:"labels"`
}

// RepoURL returns the standard HTTPS clone URL.
func (pr PullRequest) RepoURL() string {
	return "https://github.com/" + pr.Repository.NameWithOwner
}

// CloneURL returns the URL to use for git clone.
// If sshHost is set, it uses the SSH alias defined in ~/.ssh/config:
//
//	git@<sshHost>:<owner>/<repo>.git
//
// Otherwise it falls back to the standard HTTPS URL.
// This allows teams with custom SSH config entries to work alongside
// those using default HTTPS auth.
func (pr PullRequest) CloneURL(sshHost string) string {
	if sshHost != "" {
		return fmt.Sprintf("git@%s:%s.git", sshHost, pr.Repository.NameWithOwner)
	}
	return pr.RepoURL()
}

// ListPendingPRs fetches all open PRs where the authenticated user is a
// requested reviewer across all repositories, filtering out PR numbers
// already present in the processed sync.Map.
//
// Uses gh search prs so it works from any directory without a local git context.
func ListPendingPRs(ctx context.Context, exec Executor, processed *sync.Map) ([]PullRequest, error) {
	out, err := exec.Run(ctx,
		"gh", "search", "prs",
		"--review-requested=@me",
		"--state=open",
		"--json", "number,title,body,repository,labels",
	)
	if err != nil {
		return nil, fmt.Errorf("gh search prs: %w", err)
	}

	var all []PullRequest
	if err := json.Unmarshal(out, &all); err != nil {
		return nil, fmt.Errorf("parse gh search prs output: %w", err)
	}

	pending := make([]PullRequest, 0, len(all))
	for _, pr := range all {
		if _, done := processed.Load(pr.Number); !done {
			pending = append(pending, pr)
		}
	}
	return pending, nil
}

// PRRefs holds the head branch, base branch, and commit list for a PR.
type PRRefs struct {
	HeadRefName string   // the PR branch (e.g. "feat/my-feature")
	BaseRefName string   // the target branch (e.g. "main", "master", "develop")
	Commits     []Commit // commits included in the PR, oldest first
}

// FetchPRRefs returns the head/base branch names and commit list for the given PR.
//
//	gh pr view <number> --repo <nameWithOwner> --json headRefName,baseRefName,commits
func FetchPRRefs(ctx context.Context, exec Executor, prNumber int, nameWithOwner string) (PRRefs, error) {
	out, err := exec.Run(ctx,
		"gh", "pr", "view", fmt.Sprintf("%d", prNumber),
		"--repo", nameWithOwner,
		"--json", "headRefName,baseRefName,commits",
	)
	if err != nil {
		return PRRefs{}, fmt.Errorf("gh pr view #%d: %w", prNumber, err)
	}

	var result struct {
		HeadRefName string `json:"headRefName"`
		BaseRefName string `json:"baseRefName"`
		Commits     []struct {
			MessageHeadline string `json:"messageHeadline"`
		} `json:"commits"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return PRRefs{}, fmt.Errorf("parse refs for PR #%d: %w", prNumber, err)
	}

	commits := make([]Commit, len(result.Commits))
	for i, c := range result.Commits {
		commits[i] = Commit{MessageHeadline: c.MessageHeadline}
	}

	return PRRefs{
		HeadRefName: result.HeadRefName,
		BaseRefName: result.BaseRefName,
		Commits:     commits,
	}, nil
}
