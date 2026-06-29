# 🤖 Agent Guide: Spec Wizard v2 Orchestration

Este guia é destinado a agentes de IA (como Cursor, Claude Code, Antigravity) que desejam operar de forma autônoma utilizando o framework Spec Wizard v2.

## Fluxo de Operação Autônomo

O Spec Wizard v2 introduz um ciclo de vida estruturado para tarefas, baseado no padrão **Judge-Athlete**.

### 1. Descoberta de Tarefas (`wz_status`)
Sempre comece consultando o status global do projeto.
- **Ferramenta**: `wz_status`
- **Ação**: Identifique a primeira tarefa com `status: "pending"`.

### 2. Aquisição de Contexto (`wz_instructions`)
Antes de codificar, obtenha as regras e o contexto específico da tarefa.
- **Ferramenta**: `wz_instructions(task_id="X")`
- **Ação**: Leia o payload retornado. Ele contém as **Golden Rules** do projeto e a descrição técnica da tarefa.

### 3. Implementação & TDD Cycle
Execute as alterações de código conforme as instruções.
- **Dica**: Siga rigorosamente os padrões arquiteturais descritos no `config.yaml` (ex: Repository Pattern, Clean Architecture).

### 4. Auditoria de Qualidade (`wz_audit`)
Após concluir a implementação, você **DEVE** submeter seu trabalho para auditoria.
- **Ferramenta**: `wz_audit(task_id="X")`
- **Ação**: O Spec Wizard executará testes e análises estáticas. Se houver falhas, corrija-as e rode o audit novamente.

### 5. Atualização do Roadmap (`wz_sync`)
Quando a tarefa estiver pronta e aprovada (ou se você precisar marcar progresso parcial no Markdown):
- **Ação**: Marque o checkbox correspondente no `ROADMAP.md` (ex: `- [x] **Task 1**`).
- **Ferramenta**: `wz_sync`
- **Resultado**: O estado no banco de dados será sincronizado com o Markdown.

## Melhores Práticas para Agentes

- **Não ignore as Golden Rules**: Elas são injetadas em seus prompts de sistema via arquivos `.cursor/rules` ou similares. Elas têm precedência sobre implementações genéricas.
- **Sincronização Frequente**: Use `wz_sync` sempre que fizer alterações significativas no `ROADMAP.md`.
- **Mensagens de Erro**: Se `wz_audit` falhar, analise os logs retornados. Eles são projetados para serem consumidos por IAs e indicam exatamente onde o contrato foi quebrado.

---
*Documentação v2.0 - Headless First*
