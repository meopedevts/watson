# CLAUDE.md

## O que é este projeto

Watson é um daemon em Go que automatiza code reviews de PRs no GitHub usando o Claude Code como engine de análise. A cada tick, ele busca PRs onde o usuário configurado é reviewer, gera um review via `claude --print` e posta como comentário no PR.

**O daemon nunca faz push, merge ou aprovação. A única escrita no GitHub é o comentário de review.**

## Comandos essenciais

```bash
task build    # Compila o binário em ./watson
task check    # go vet + go test ./... (rodar antes de qualquer commit)
task test     # go test ./...
task tidy     # go mod tidy
```

Sem o Taskfile: `go build -o watson ./cmd/` e `go test ./...`

## Arquitetura

```
cmd/main.go           Entry point: flags, config, ticker, signal handling
config/config.go      Leitura de env vars com defaults e validação
internal/github/      Interface Executor, listagem de PRs, postagem de comentários
internal/git/         Clone, diff, merge (detecção de conflitos), sanitização
internal/reviewer/    Orquestração do pipeline, construção do prompt, chamada ao Claude
```

O projeto **não tem dependências externas** — stdlib only.

### Interface Executor

Toda chamada a `git`, `gh` e `claude` passa pela interface `github.Executor`. Em testes, um `MockExecutor` substitui o `ShellExecutor` sem executar nada real. Ao adicionar chamadas a CLIs externas, sempre use o `Executor` injetado — nunca `os/exec` diretamente.

### Prompt de review

O template de saída esperado pelo Claude está em `internal/reviewer/prompt.go`. Qualquer mudança no formato do review deve ser feita ali.

## Convenções

- **Commits:** conventional commits (`feat:`, `fix:`, `docs:`, `chore:`, `refactor:`)
- **Branches:** `feat/`, `fix:`, `docs/`, `chore/`
- **Testes:** sem dependências externas — todo I/O é mockado via `Executor`
- **Sem `os/exec` direto:** use sempre a interface `Executor`
- **Sem paths Unix hardcoded:** use `filepath.Join(os.TempDir(), ...)` para diretórios temporários

## O que não fazer

- Não commitar o binário `./watson` (está no `.gitignore`)
- Não commitar `.env` (está no `.gitignore`)
- Não adicionar dependências externas sem discussão — o projeto é stdlib only por design
- Não usar `os/exec` diretamente fora do `ShellExecutor`
