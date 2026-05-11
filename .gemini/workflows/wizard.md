---
description: Executa tarefas do Roadmap do Spec Wizard de forma autônoma (TDD).
---

# Workflow: /wizard
Este workflow integra o Antigravity com o ecossistema Spec Wizard, permitindo a execução autônoma de tarefas do Roadmap.

## 📋 Como usar:
Digite `/wizard <task_id>` no chat. Exemplo: `/wizard 2`

## 🚀 Passos de Execução:

### 1. Extração de Contexto
// turbo
O Antigravity deve localizar os arquivos de definição do projeto.
- Leia o arquivo `roadmap.json` (ou `roadmap.md`) para encontrar a tarefa especificada.
- Leia `PRD.md` e `SPEC.md` para entender as regras de negócio e arquitetura.
- Leia o arquivo `go.mod` para garantir o uso correto dos pacotes.

### 2. Configuração da Persona
Assuma a persona:
"Você é um Senior Software Engineer operando sob o Protocolo Spec Wizard. Seu objetivo é implementar a tarefa solicitada com 100% de cobertura de testes e aderência estrita à arquitetura definida no SPEC."

### 3. Ciclo de Execução TDD (Autônomo)
Para a tarefa identificada, siga este ciclo:
1. **Planejar**: Identifique quais arquivos precisam ser criados ou modificados.
2. **Implementar**: Use as ferramentas `write_file` ou `edit_file` para realizar as mudanças.
3. **Testar**: Identifique o comando de teste apropriado (ex: `go test ./...`).
4. **Validar**: Execute o teste usando `run_command`.
5. **Corrigir**: Se os testes falharem, analise os logs e corrija até passar.

---
**IMPORTANTE**: Responda apenas via chamadas de ferramentas durante a execução.
