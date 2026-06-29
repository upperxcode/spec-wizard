# 🧙‍♂️ Guia do Usuário - Spec Wizard v2

O Spec Wizard é um orquestrador de desenvolvimento assistido por IA que transforma requisitos em código seguindo padrões de arquitetura rigorosos.

## 🚀 Instalação

Para instalar o Spec Wizard globalmente no seu sistema Linux:

```bash
# Dentro da pasta do projeto
chmod +x scripts/all
./scripts/all
```

Isso instalará:
- O binário `wz` (alias para `mcp-wizard`) em `~/.local/bin/`.
- O servidor MCP para integração com editores (Cursor, VSCode).
- A interface web (Dashboard) em `~/.spec-wizard/ui`.

## 🛠️ Comandos Principais

### Governança e Roadmap
- `wz roadmap` (ou `wz status`): Exibe o progresso global do projeto e estatísticas de conclusão.
- `wz status <id>`: Exibe a **Fotografia Técnica** de uma tarefa (diagnóstico físico de arquivos, bugs e specs).
- `wz sync`: Sincroniza o arquivo `ROADMAP.md` com o estado interno do sistema.

### Ciclo de Desenvolvimento de Tarefas
1. `wz goal <id>`: Recupera o objetivo original (User Story) da tarefa.
2. `wz refine <id>`: Gera o **TASK SPEC** (contrato técnico que a IA deve seguir).
3. `wz prepare <id>`: Carrega o contexto e valida se a tarefa está pronta para codificação.
4. `wz code <id>`: Inicia o modo de implementação (exclusivo para agentes de IA).
5. `wz audit <id>`: Executa a auditoria formal (testes e análise de arquivos) para validar a entrega.

## 🔄 Fluxo de Trabalho Recomendado

1. **Inicie o dia** com `wz roadmap` para ver o que está pendente.
2. **Escolha uma tarefa** e veja o detalhe com `wz status <id>`.
3. **Refine a tarefa** com `wz refine <id>` para criar o contrato técnico.
4. **Deixe a IA trabalhar** usando `wz code <id>`.
5. **Valide tudo** com `wz audit <id>`. Se houver bugs, o sistema os registrará automaticamente na fotografia da tarefa.

## 🧪 Configuração
Os arquivos de configuração ficam em `~/.config/spec-wizard/` e o estado do projeto local em `.spec-wizard/`.
