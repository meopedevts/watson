package reviewer

import (
	"strings"
	"testing"

	"github.com/meopedevts/watson/internal/github"
)

func TestBuildPrompt_ContainsTitleAndDiff(t *testing.T) {
	pr := github.PullRequest{
		Number: 42,
		Title:  "feat: add authentication",
		Body:   "This PR adds JWT auth.",
	}
	diff := "diff --git a/auth.go b/auth.go\n+func Login() {}"

	prompt := BuildPrompt(PromptContext{PR: pr, Diff: diff})

	for _, want := range []string{"feat: add authentication", "This PR adds JWT auth.", "diff --git a/auth.go"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPrompt_EmptyBodyFallback(t *testing.T) {
	pr := github.PullRequest{Number: 1, Title: "fix: typo"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "some diff"})

	if !strings.Contains(prompt, "sem descrição fornecida") {
		t.Error("expected empty-body fallback text")
	}
}

func TestBuildPrompt_EmptyDiffFallback(t *testing.T) {
	pr := github.PullRequest{Number: 1, Title: "chore: bump version", Body: "bump"}
	prompt := BuildPrompt(PromptContext{PR: pr})

	if !strings.Contains(prompt, "diff vazio") {
		t.Error("expected empty-diff fallback text")
	}
}

func TestBuildPrompt_ContainsPRNumber(t *testing.T) {
	pr := github.PullRequest{Number: 99, Title: "refactor: cleanup"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	if !strings.Contains(prompt, "#99") {
		t.Error("expected PR number in prompt")
	}
}

func TestBuildPrompt_ContainsRepository(t *testing.T) {
	pr := github.PullRequest{
		Number:     10,
		Title:      "fix: bug",
		Repository: github.Repository{NameWithOwner: "acme/backend"},
	}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	if !strings.Contains(prompt, "acme/backend") {
		t.Error("expected repository name in prompt")
	}
}

func TestBuildPrompt_ContainsLabels(t *testing.T) {
	pr := github.PullRequest{
		Number: 5,
		Title:  "feat: payments",
		Labels: []github.Label{{Name: "feature"}, {Name: "breaking-change"}},
	}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	if !strings.Contains(prompt, "feature") || !strings.Contains(prompt, "breaking-change") {
		t.Error("expected labels in prompt")
	}
}

func TestBuildPrompt_NoLabelsSection_WhenEmpty(t *testing.T) {
	pr := github.PullRequest{Number: 3, Title: "chore: cleanup"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	if strings.Contains(prompt, "**Labels:**") {
		t.Error("labels section should be absent when there are no labels")
	}
}

func TestBuildPrompt_ContainsCommits(t *testing.T) {
	pr := github.PullRequest{Number: 7, Title: "feat: auth"}
	refs := github.PRRefs{
		Commits: []github.Commit{
			{MessageHeadline: "feat: add JWT token generation"},
			{MessageHeadline: "fix: handle expired tokens"},
		},
	}
	prompt := BuildPrompt(PromptContext{PR: pr, Refs: refs, Diff: "diff"})

	if !strings.Contains(prompt, "feat: add JWT token generation") {
		t.Error("expected commit message in prompt")
	}
	if !strings.Contains(prompt, "fix: handle expired tokens") {
		t.Error("expected second commit message in prompt")
	}
}

func TestBuildPrompt_ContainsStats(t *testing.T) {
	pr := github.PullRequest{Number: 8, Title: "refactor: cleanup"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff", Stats: "3 files changed, 20 insertions(+)"})

	if !strings.Contains(prompt, "3 files changed") {
		t.Error("expected diff stats in prompt")
	}
}

func TestBuildPrompt_ContainsReviewTemplate(t *testing.T) {
	pr := github.PullRequest{Number: 1, Title: "fix: bug"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	for _, section := range []string{
		"## Resumo",
		"## Problemas encontrados",
		"## Pontos positivos",
		"## Sugestões",
		"## Veredicto",
		"🔴", "🟡", "✅", "🔄", "🚫",
	} {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing template section %q", section)
		}
	}
}

func TestBuildPrompt_ReReview_ContainsContext(t *testing.T) {
	pr := github.PullRequest{Number: 10, Title: "fix: auth"}
	ctx := PromptContext{
		PR:             pr,
		Diff:           "diff",
		IsReReview:     true,
		PreviousReview: "## Resumo\nAlterou autenticação.",
		MentionComment: "@watson o bug foi corrigido, por favor revise novamente",
	}
	prompt := BuildPrompt(ctx)

	for _, want := range []string{
		"revisão atualizada",
		"@watson o bug foi corrigido",
		"## Resumo\nAlterou autenticação.",
		"Contexto do re-review",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}

func TestBuildPrompt_ReReview_False_NoContext(t *testing.T) {
	pr := github.PullRequest{Number: 10, Title: "fix: auth"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff"})

	if strings.Contains(prompt, "revisão atualizada") {
		t.Error("non-re-review prompt should not contain re-review header")
	}
	if strings.Contains(prompt, "Contexto do re-review") {
		t.Error("non-re-review prompt should not contain re-review section")
	}
}

func TestBuildPrompt_ContainsNote(t *testing.T) {
	pr := github.PullRequest{Number: 9, Title: "chore: deps"}
	prompt := BuildPrompt(PromptContext{PR: pr, Diff: "diff", Note: "Arquivos ignorados: go.sum"})

	if !strings.Contains(prompt, "Arquivos ignorados: go.sum") {
		t.Error("expected sanitization note in prompt")
	}
}
