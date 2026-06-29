# 📊 Governança e Dashboard Executivo

O Spec Wizard v2 introduz um sistema de governança baseado em evidências físicas, eliminando a discrepância entre o status reportado e a realidade do código.

## 🔭 Dashboard Executivo (`wz status`)
O comando de status global fornece uma visão macro para gestores e desenvolvedores:
- **Progresso Real**: Calculado com base em tarefas marcadas como `completed`.
- **Visão por Sprint**: Agrupamento lógico de metas.
- **Identificação de Bloqueios**: Destaque para tarefas que falharam na auditoria.

## 📸 Fotografia Técnica (`wz status <id>`)
Diferente de um simples "OK", a fotografia técnica realiza uma inspeção profunda:
- **Verificação de Spec**: Existe um contrato técnico assinado?
- **Detecção de Arquivos**: Os arquivos de código e teste foram realmente criados no disco?
- **Rastreamento de Bugs**: Se um teste falhar, o erro é injetado diretamente na fotografia da tarefa.

## ⚖️ Ciclo de Mão Dupla (Judge-Athlete)
O sistema opera em um loop contínuo de validação:

1. **Athlete (IA)**: Implementa o código baseado no `TASK SPEC`.
2. **Judge (Spec Wizard)**: Roda o `wz audit`.
3. **Sincronização**:
   - **Sucesso**: O sistema marca a tarefa como `completed` automaticamente.
   - **Falha**: O sistema reabre a tarefa, marca como `bug` e anexa o log de erro para que a IA possa corrigir imediatamente.

Este processo garante que o `ROADMAP.md` seja sempre a fonte única da verdade, refletindo o estado físico exato do repositório.
