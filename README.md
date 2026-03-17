# watson

[Arquitetura](docs/architecture.md) · [Configuração](docs/configuration.md) · [Desenvolvimento](docs/development.md) · [Docker](docs/docker.md)

---

Daemon em Go que automatiza code reviews de pull requests no GitHub usando o Claude Code como engine de análise.

A cada intervalo configurável, o programa detecta PRs onde você é reviewer, analisa o diff com IA e posta o review diretamente como comentário no PR — sem intervenção manual.

**O programa nunca faz push, merge ou aprovação. A única escrita no GitHub é o comentário de review.**

---

## Como funciona

```mermaid
flowchart TD
    A([Início]) --> B[Carrega configuração]
    B --> C[Inicia time.Ticker]
    C --> D[Busca PRs pendentes\ngh search prs]
    D --> E{PRs encontrados?}
    E -- Não --> WAIT([Aguarda próximo tick])
    E -- Sim --> F[Goroutine por PR]

    subgraph GOROUTINE [" "]
        F --> G[Busca head e base branch\ngh pr view]
        G --> H[Clone raso\ngit clone --depth=1]
        H --> I[Diff contra base\ngit diff]
        I --> J[Gera review\nclaude --print]
        J --> K[Verifica conflitos\ngit merge --no-commit]
        K --> L{Dry-run?}
        L -- Sim --> M[Imprime no stdout]
        L -- Não --> N[Posta comentário\ngh pr comment]
        M & N --> O[Marca PR como processado]
        O --> P[Remove clone temporário]
    end

    P --> WAIT
```

---

## Instalação

### Binário local

**Pré-requisitos:**
- Go 1.25+
- [GitHub CLI](https://cli.github.com/) autenticado (`gh auth login`)
- [Claude Code CLI](https://claude.ai/code) instalado e autenticado

```bash
git clone https://github.com/meopedevts/watson
cd watson
go build -o watson ./cmd/
```

### Docker

Não requer Go, GitHub CLI nem Claude Code instalados localmente.

```bash
cp .env.example .env   # preencha as variáveis
docker compose up -d
```

> Veja [`docs/docker.md`](docs/docker.md) para o guia completo de autenticação e configuração.

---

## Uso

```bash
# Produção — posta comentários nos PRs
GH_TOKEN=... ./watson

# Dry-run — imprime reviews no stdout, sem escrever no GitHub
GH_TOKEN=... ./watson --dry-run

# Poll a cada 5 minutos
GH_TOKEN=... POLL_INTERVAL_MINUTES=5 ./watson
```

Pressione `Ctrl+C` para encerrar o daemon graciosamente.

---

## Configuração

| Variável | Obrigatória | Padrão | Descrição |
|----------|:-----------:|--------|-----------|
| `GH_TOKEN` | Sim¹ | — | Classic PAT do GitHub (escopo `repo`) |
| `CLAUDE_CODE_OAUTH_TOKEN` | Sim² | — | OAuth token de longa duração (`claude setup-token`) |
| `ANTHROPIC_API_KEY` | Sim² | — | API key da Anthropic (alternativa ao OAuth token) |
| `POLL_INTERVAL_MINUTES` | Não | `15` | Intervalo de polling em minutos |
| `CLAUDE_MODEL` | Não | `claude-sonnet-4-20250514` | Modelo Claude utilizado |
| `REPO_BASE_DIR` | Não | `os.TempDir()/watson` | Diretório para clones temporários |
| `GIT_SSH_HOST` | Não | — | Alias SSH para autenticação customizada |
| `REVIEW_TTL_HOURS` | Não | `168` (7 dias) | Tempo máximo que um PR revisado permanece no cache. Após esse período, o PR é removido da memória e Watson deixa de verificar menções nele. Controla o consumo de memória e o volume de chamadas à API do GitHub por tick (uma chamada por PR em cache). |
| `RE_REVIEW_COOLDOWN_MINUTES` | Não | `60` | Intervalo mínimo entre dois reviews consecutivos do mesmo PR. Menções recebidas dentro desse período são ignoradas, evitando spam de comentários em caso de múltiplas menções rápidas. Recomenda-se um valor ≥ `POLL_INTERVAL_MINUTES`. |

¹ Obrigatório no Docker. No uso local o `gh auth login` é suficiente.
² Escolha um dos dois. `CLAUDE_CODE_OAUTH_TOKEN` usa seu plano Pro/Max sem cobrança por token.

> Para autenticação via SSH com configuração customizada, veja [`docs/configuration.md`](docs/configuration.md).

---

## Tasks

```bash
task build          # Compila o binário
task run            # Executa em produção
task dry-run        # Executa sem postar no GitHub
task dev            # Dry-run com poll de 1 minuto
task test           # Roda os testes
task check          # vet + testes
task clean          # Remove binário e artefatos
```

---

## Documentação

| Documento | Conteúdo |
|-----------|----------|
| [`docs/docker.md`](docs/docker.md) | Deploy com Docker: build, autenticação OAuth/API key, SSH |
| [`docs/architecture.md`](docs/architecture.md) | Estrutura de pacotes, decisões técnicas, fluxo interno |
| [`docs/configuration.md`](docs/configuration.md) | Referência completa de configuração e autenticação |
| [`docs/development.md`](docs/development.md) | Build, testes, cobertura e convenções |
