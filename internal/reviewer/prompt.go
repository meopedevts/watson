package reviewer

import (
	"fmt"
	"strings"

	"github.com/meopedevts/watson/internal/github"
)

// PromptContext holds all data needed to build the review prompt.
type PromptContext struct {
	PR    github.PullRequest
	Refs  github.PRRefs
	Diff  string
	Stats string // output of "git diff --stat"
	Note  string // files excluded by sanitization, or truncation warning
}

// BuildPrompt assembles the review prompt sent to the Claude CLI.
func BuildPrompt(ctx PromptContext) string {
	pr := ctx.PR

	body := pr.Body
	if body == "" {
		body = "(sem descrição fornecida)"
	}

	diff := ctx.Diff
	if diff == "" {
		diff = "(diff vazio — sem alterações em relação à branch principal)"
	}

	var sb strings.Builder

	sb.WriteString("Você é um engenheiro de software sênior fazendo um code review. Responda em português de forma direta e técnica.\n\n")
	fmt.Fprintf(&sb, "## Pull Request #%d: %s\n", pr.Number, pr.Title)
	fmt.Fprintf(&sb, "**Repositório:** %s\n", pr.Repository.NameWithOwner)

	if len(pr.Labels) > 0 {
		names := make([]string, len(pr.Labels))
		for i, l := range pr.Labels {
			names[i] = l.Name
		}
		fmt.Fprintf(&sb, "**Labels:** %s\n", strings.Join(names, ", "))
	}

	sb.WriteString("\n### Descrição do autor\n")
	sb.WriteString(body)
	sb.WriteString("\n")

	if len(ctx.Refs.Commits) > 0 {
		sb.WriteString("\n### Commits neste PR\n")
		for _, c := range ctx.Refs.Commits {
			fmt.Fprintf(&sb, "- %s\n", c.MessageHeadline)
		}
	}

	sb.WriteString("\n### Resumo das mudanças\n")
	if ctx.Stats != "" {
		sb.WriteString(ctx.Stats)
		sb.WriteString("\n")
	}
	if ctx.Note != "" {
		sb.WriteString(ctx.Note)
		sb.WriteString("\n")
	}

	sb.WriteString("\n### Diff\n")
	sb.WriteString("```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n\n")

	sb.WriteString("Seu review deve cobrir:\n")
	sb.WriteString("1. **Resumo das mudanças** — o que foi alterado e por quê\n")
	sb.WriteString("2. **Problemas encontrados** — bugs, falhas de segurança, má prática\n")
	sb.WriteString("3. **Sugestões de melhoria** — refatoração, performance, legibilidade\n")
	sb.WriteString("4. **Pontos positivos** — o que está bem implementado\n")
	sb.WriteString("5. **Veredicto** — Aprovado / Mudanças necessárias\n\n")
	sb.WriteString("Seja objetivo. Aponte arquivo e trecho de código quando relevante.")

	return sb.String()
}
