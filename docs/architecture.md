# Arquitetura

## Estrutura de pacotes

```
github.com/meopedevts/watson/
├── cmd/
│   └── main.go                   Entry point: flags, config, ticker, signal handling
├── config/
│   └── config.go                 Leitura de env vars com defaults e validação
├── internal/
│   ├── github/
│   │   ├── client.go             Interface Executor + ShellExecutor
│   │   ├── pr.go                 PullRequest, ListPendingPRs, FetchPRRefs
│   │   ├── mock_executor_test.go MockExecutor para testes
│   │   └── pr_test.go
│   ├── git/
│   │   ├── clone.go              git clone --depth=1
│   │   ├── diff.go               git fetch + git diff
│   │   ├── merge.go              git merge --no-commit (detecção de conflitos)
│   │   └── merge_test.go
│   └── reviewer/
│       ├── prompt.go             BuildPrompt: monta o prompt para o Claude
│       ├── review.go             Reviewer: orquestra todo o pipeline
│       └── prompt_test.go
├── docs/
├── Taskfile.yml
└── go.mod                        Sem dependências externas — stdlib only
```

### Grafo de dependências

```
config
  └── (sem deps internas)

internal/github/client       ← Executor interface
  └── (sem deps internas)

internal/github/pr
  └── internal/github/client

internal/git/{clone,diff,merge}
  └── internal/github/client

internal/reviewer/prompt
  └── internal/github/pr

internal/reviewer/review
  └── config
  └── internal/github/client
  └── internal/github/pr
  └── internal/git/*
  └── internal/reviewer/prompt

cmd/main
  └── config
  └── internal/github/client
  └── internal/reviewer/review
```

Sem ciclos. `internal/github/client` é o único ponto de dependência compartilhado entre `internal/git` e `internal/reviewer`.

---

## Decisões de design

### Daemon com `time.Ticker`

Processo único e autossuficiente, sem dependência de cron externo. Executa imediatamente no startup e a cada tick. Shutdown gracioso via `signal.NotifyContext` ao receber `SIGINT` ou `SIGTERM`.

### Interface `Executor`

Toda chamada a `git`, `gh` e `claude` passa por uma interface injetável:

```go
type Executor interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
    RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error)
}
```

Isso desacopla a lógica de negócio dos comandos shell — em testes, um `MockExecutor` substitui o `ShellExecutor` sem executar nada real.

### Deduplicação at-least-once

Um `sync.Map` rastreia os números de PRs processados na execução atual. O PR só é marcado **após o comentário ser postado com sucesso** — se qualquer etapa anterior falhar, o PR é reprocessado no próximo tick.

O estado não persiste entre reinicializações, mantendo o daemon stateless por design.

### Detecção de conflitos local

O merge para detectar conflitos é executado exclusivamente no diretório temporário — sem efeito no repositório remoto. O diretório é destruído pelo `defer os.RemoveAll` ao final da goroutine, independente do resultado.

---

## Prompt enviado ao Claude

```
Você é um engenheiro de software sênior fazendo um code review.
Responda em português de forma direta e técnica.

## Pull Request #<N>: <título>

### Descrição do autor
<body do PR>

### Diff
<diff completo>

Seu review deve cobrir:
1. Resumo das mudanças
2. Problemas encontrados (bugs, segurança, má prática)
3. Sugestões de melhoria
4. Pontos positivos
5. Veredicto: Aprovado / Mudanças necessárias
```

Se conflitos forem detectados, um aviso é acrescentado ao review antes de ser postado.
