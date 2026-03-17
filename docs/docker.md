# Docker

## Pré-requisitos

- [Docker](https://docs.docker.com/get-docker/) com Compose V2 (`docker compose version`)
- [GitHub CLI](https://cli.github.com/) instalado **localmente** para gerar o token OAuth (Opção 1)

---

## Quickstart

```bash
# 1. Copie e preencha as variáveis de ambiente
cp .env.example .env

# 2. Build da imagem
docker compose build

# 3. Suba o daemon
docker compose up -d

# 4. Acompanhe os logs
docker compose logs -f
```

---

## Autenticação do Claude Code

O watson suporta duas formas de autenticar o Claude Code no container. Escolha a que melhor se encaixa no seu uso.

---

### Opção 1 — OAuth token de longa duração (recomendado)

Usa o seu plano Pro/Max — sem cobrança por token.

**Como gerar o token:**

```bash
claude setup-token
```

O comando exibe um token OAuth de longa duração. Copie-o e adicione ao `.env`:

```env
CLAUDE_CODE_OAUTH_TOKEN=your-oauth-token
```

O Claude Code detecta automaticamente a variável e usa o token para autenticar — sem login interativo, sem sessão para gerenciar.

> **Renovação:** tokens gerados pelo `claude setup-token` têm duração estendida, mas podem expirar. Se o container começar a falhar com erro de autenticação, basta rodar `claude setup-token` novamente e atualizar o `.env`.

---

### Opção 2 — Anthropic API key

Cobrança por token consumido. Não requer plano.

```env
ANTHROPIC_API_KEY=sk-ant-...
```

Gere sua chave em [console.anthropic.com/settings/keys](https://console.anthropic.com/settings/keys).

> Use esta opção para ambientes de CI/CD ou quando não houver um plano Pro/Max disponível.

---

## Autenticação do GitHub CLI

O `gh` CLI autentica via variável de ambiente `GH_TOKEN`. Gere um **Personal Access Token** (clássico ou fine-grained) em [github.com/settings/tokens](https://github.com/settings/tokens).

**Escopos mínimos necessários:**

| Escopo | Motivo |
|--------|--------|
| `repo` | Listar PRs, ler diffs, postar comentários |
| `read:org` | Buscar PRs em repositórios de organizações |

```env
GH_TOKEN=your-github-token
```

---

## Identidade Git (opcional)

O watson executa `git merge --no-commit` para detectar conflitos nos PRs. O git exige uma identidade configurada para isso. Defina no `.env`:

```env
GIT_USER_NAME=Your Name
GIT_USER_EMAIL=you@example.com
```

> Essa identidade nunca é enviada ao GitHub — é usada apenas localmente dentro do container durante a detecção de conflitos.

---

## Clonagem via SSH (opcional)

Por padrão o watson clona repositórios via HTTPS. Se preferir SSH (ex.: múltiplas contas GitHub ou GitHub Enterprise), defina `GIT_SSH_HOST` e monte suas chaves SSH no container.

No `.env`:

```env
GIT_SSH_HOST=my-ssh-alias
```

No `docker-compose.yml`, descomente o volume:

```yaml
volumes:
  - ${HOME}/.ssh:/root/.ssh:ro
```

O alias deve estar definido em `~/.ssh/config` na sua máquina host. Veja [`docs/configuration.md`](configuration.md) para detalhes.

---

## Comandos úteis

```bash
docker compose build          # Reconstrói a imagem
docker compose up -d          # Sobe o daemon em background
docker compose logs -f        # Acompanha os logs em tempo real
docker compose down           # Para e remove o container
docker compose restart        # Reinicia o container
```
