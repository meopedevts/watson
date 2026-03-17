# Configuração

## Variáveis de ambiente

### `GITHUB_REVIEWER_USERNAME` (obrigatória)

Usuário GitHub cujos review requests serão monitorados. O daemon busca PRs onde este usuário foi solicitado como reviewer.

```bash
GITHUB_REVIEWER_USERNAME=seu-usuario ./pr-reviewer
```

---

### `POLL_INTERVAL_MINUTES`

Intervalo em minutos entre cada poll do GitHub.

- **Padrão:** `15`
- **Restrição:** deve ser um inteiro positivo

```bash
POLL_INTERVAL_MINUTES=5 ./pr-reviewer
```

O daemon também executa imediatamente ao subir, sem esperar o primeiro tick.

---

### `CLAUDE_MODEL`

Identificador do modelo Claude passado para a CLI. Qualquer modelo disponível na sua conta pode ser usado.

- **Padrão:** `claude-sonnet-4-20250514`

```bash
CLAUDE_MODEL=claude-opus-4-5 ./pr-reviewer
```

---

### `REPO_BASE_DIR`

Diretório onde os clones temporários são criados. Cada PR gera um subdiretório com nome único (`pr-clone-*`) que é removido ao final do processamento.

- **Padrão:** `/tmp/pr-reviewer`

```bash
REPO_BASE_DIR=/var/tmp/reviews ./pr-reviewer
```

O diretório é criado automaticamente se não existir.

---

### `GIT_SSH_HOST`

Alias SSH para autenticação customizada, definido em `~/.ssh/config`. Quando configurado, o clone usa SSH em vez de HTTPS.

- **Padrão:** vazio (usa HTTPS)

**Sem a variável** — clone via HTTPS:
```
https://github.com/owner/repo
```

**Com a variável** — clone via SSH com o alias:
```
git@<GIT_SSH_HOST>:owner/repo.git
```

#### Quando usar

Use `GIT_SSH_HOST` quando:

- Você gerencia múltiplas contas GitHub com chaves SSH diferentes
- Sua organização usa GitHub Enterprise com autenticação via certificado
- O acesso HTTPS ao GitHub está bloqueado no seu ambiente

#### Como configurar

1. Crie uma entrada em `~/.ssh/config`:

```
Host <alias>
    HostName github.com        # ou seu host Enterprise
    User git
    AddKeysToAgent yes
    IdentityFile ~/.ssh/<sua-chave>
    IdentitiesOnly yes
```

2. Passe o alias via variável de ambiente:

```bash
GIT_SSH_HOST=<alias> ./pr-reviewer
```

3. Verifique que a chave funciona:

```bash
ssh -T git@<alias>
```

---

## Flag de linha de comando

### `--dry-run`

Imprime os reviews no stdout sem postar comentários no GitHub. Útil para testar o pipeline completo sem efeitos colaterais.

```bash
./pr-reviewer --dry-run
```

Em dry-run, todas as etapas são executadas normalmente (clone, diff, Claude, detecção de conflitos). Apenas o `gh pr comment` é substituído por uma impressão no terminal.

---

## Exemplo completo

```bash
GITHUB_REVIEWER_USERNAME=seu-usuario \
POLL_INTERVAL_MINUTES=10 \
CLAUDE_MODEL=claude-sonnet-4-20250514 \
REPO_BASE_DIR=/tmp/reviews \
GIT_SSH_HOST=meu-alias-ssh \
./pr-reviewer
```

---

## Usando o Taskfile

As variáveis podem ser passadas diretamente para as tasks:

```bash
GITHUB_REVIEWER_USERNAME=seu-usuario task run
GITHUB_REVIEWER_USERNAME=seu-usuario task dry-run
GITHUB_REVIEWER_USERNAME=seu-usuario GIT_SSH_HOST=meu-alias task run
```
