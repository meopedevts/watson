# Desenvolvimento

## Pré-requisitos

- Go 1.21+
- [Task](https://taskfile.dev/) (`task --version`)
- [GitHub CLI](https://cli.github.com/) (`gh --version`)
- [Claude Code CLI](https://claude.ai/code) (`claude --version`)

---

## Build

```bash
task build
# ou diretamente:
go build -o pr-reviewer ./cmd/
```

O binário é gerado na raiz do projeto como `./pr-reviewer`.

---

## Testes

```bash
# Todos os testes
task test

# Com saída detalhada
task test-verbose

# Cobertura — gera coverage.html
task test-coverage
```

Os testes unitários não executam comandos reais. Toda chamada a `git`, `gh` e `claude` é interceptada pelo `MockExecutor`.

### Estrutura dos testes

| Arquivo | O que testa |
|---------|-------------|
| `internal/github/pr_test.go` | Parsing do JSON do `gh search prs`, filtragem por `sync.Map`, `CloneURL` com e sem SSH alias, `FetchPRRefs` |
| `internal/git/merge_test.go` | Detecção de conflito via output `"CONFLICT"`, merge limpo, erro de fetch |
| `internal/reviewer/prompt_test.go` | Conteúdo do prompt gerado, fallbacks para body e diff vazios |

### MockExecutor

Definido em `internal/github/mock_executor_test.go`. Consome respostas pré-configuradas em FIFO e registra todas as chamadas para asserção:

```go
exec := &MockExecutor{
    Responses: []MockResponse{
        {Out: []byte(`[{"number":1,...}]`), Err: nil},
    },
}
prs, err := ListPendingPRs(ctx, exec, &sync.Map{})
```

---

## Tasks disponíveis

```bash
task build          # Compila ./pr-reviewer
task test           # go test ./...
task test-verbose   # go test -v ./...
task test-coverage  # Gera coverage.out e coverage.html
task vet            # go vet ./...
task check          # vet + test (recomendado antes de commitar)
task tidy           # go mod tidy
task run            # Compila e executa em produção
task dry-run        # Compila e executa em dry-run
task dev            # dry-run com POLL_INTERVAL_MINUTES=1
task clean          # Remove binário, coverage.out, coverage.html e /tmp/pr-reviewer
```

`task check` é o comando recomendado antes de qualquer commit — garante que o código compila, passa no vet e todos os testes passam.

---

## Dependências

O projeto não tem dependências externas de Go. Apenas stdlib:

```
context, encoding/json, flag, fmt, log, log/slog
os, os/exec, os/signal, strconv, strings
sync, syscall, time
```

Para adicionar uma dependência: `go get <módulo>` seguido de `task tidy`.

---

## Logs

O daemon emite logs JSON estruturados via `log/slog`:

```json
{"time":"...","level":"INFO","msg":"pr-reviewer started","pollIntervalMinutes":15,"dryRun":false}
{"time":"...","level":"INFO","msg":"processing PRs","count":2}
{"time":"...","level":"INFO","msg":"starting review","pr":42,"title":"feat: ...","repo":"org/repo"}
{"time":"...","level":"INFO","msg":"review completed","pr":42}
{"time":"...","level":"ERROR","msg":"review failed","pr":43,"err":"..."}
{"time":"...","level":"WARN","msg":"conflict check failed, skipping conflict warning","pr":42,"err":"..."}
```

Erros por PR são logados em `ERROR` mas não abortam o loop — outros PRs continuam sendo processados.

---

## Adicionando um novo campo ao review

1. Adicionar o campo à query do `gh search prs` em `internal/github/pr.go` (verificar se o campo está disponível com `gh search prs --help`)
2. Se não estiver disponível no search, adicionar ao `gh pr view` em `FetchPRRefs`
3. Atualizar a struct `PullRequest` ou `PRRefs`
4. Atualizar `BuildPrompt` em `internal/reviewer/prompt.go` para incluir o campo no prompt
5. Atualizar os testes correspondentes
