#!/bin/bash

# Spec Wizard Universal CLI Bridge
# Uso: wizard [comando] [id]

API_URL="http://127.0.0.1:10811/api/project"
PROJECT_PATH=$(pwd)

case "$1" in
    plan)
        echo "📅 Solicitando plano para Task $2..."
        curl -s -X GET "$API_URL/headless/next-step?project_path=$PROJECT_PATH&task_id=$2" | jq .
        ;;
    assist)
        echo "🧠 Gerando plano refinado para Task $2..."
        # Aqui o CLI chama a lógica de assistência
        curl -s -X GET "$API_URL/headless/next-step?project_path=$PROJECT_PATH&task_id=$2&refine=true" | jq .
        ;;
    test|audit)
        echo "🔍 Enviando Task $2 para auditoria formal..."
        curl -s -X POST "$API_URL/audit-task" \
             -H "Content-Type: application/json" \
             -d "{\"project_path\": \"$PROJECT_PATH\", \"sprint_id\": \"1\", \"task_id\": \"$2\"}" | jq .
        ;;
    status|roadmap)
        echo "📊 Status do Roadmap:"
        cat .spec-wizard/sprints.json | jq '.sprints[0].tasks[] | {id, title, status}'
        ;;
    view)
        echo "📋 Detalhes da Task $2:"
        cat .spec-wizard/sprints.json | jq ".sprints[0].tasks[] | select(.id == \"$2\")"
        ;;
    rules)
        echo "📜 Golden Rules Atuais:"
        cat .spec-wizard/skills.md
        ;;
    help)
        echo "🧙‍♂️ Spec Wizard CLI (wz) - Guia de Comandos"
        echo "=========================================="
        echo "Comandos de Planejamento:"
        echo "  wz init [lang] [name] - Inicializa um novo projeto (ex: wz init pt-BR 'Meu Projeto')."
        echo "  wz roadmap            - Exibe o progresso das Sprints e Tasks."
        echo "  wz view [id]        - Detalhes de uma tarefa específica."
        echo "  wz plan [id]        - Obtém as instruções originais da Task."
        echo "  wz assist [id]      - Gera um plano refinado baseado no código atual."
        echo ""
        echo "Comandos de Execução:"
        echo "  wz rules            - Lista as regras de arquitetura obrigatórias."
        echo "  wz test [id]        - Envia a Task para auditoria formal do Juiz."
        echo ""
        echo "Utilidades:"
        echo "  wz install          - Instala o comando 'wz' globalmente no sistema."
        echo "  wz help             - Mostra este guia."
        ;;
    init)
        echo "🧙‍♂️ Inicializando novo projeto Spec Wizard..."
        # Chama o binário MCP passando o comando de init (simulado via CLI)
        # Como o CLI e o MCP compartilham a lógica, podemos apenas criar a pasta aqui
        if [ -d ".spec-wizard" ]; then
            echo "⚠️ A pasta .spec-wizard já existe. Remova-a manualmente para reiniciar."
            exit 1
        fi
        mkdir -p .spec-wizard/tasks .spec-wizard/task-logs
        LANG=${2:-"pt-BR"}
        NAME=${3:-$(basename $(pwd))}
        
        # Gera o JSON básico
        cat <<EOF > .spec-wizard/sprints.json
{
  "project_name": "$NAME",
  "language": "$LANG",
  "pattern": "crud, dry, kiss, yagni, solid, repository",
  "sprints": [
    {
      "id": "1",
      "goal": "Setup inicial e arquitetura base.",
      "tasks": [
        {
          "id": "1",
          "title": "Setup do Projeto e Configurações",
          "description": "Configurar o go.mod, dependências básicas e estrutura de pastas.",
          "status": "pending"
        }
      ]
    }
  ]
}
EOF
        # Gera o skills.md básico
        cat <<EOF > .spec-wizard/skills.md
# 📜 Golden Rules

## Arquitetura
- Use Repository Pattern.
- Injeção de Dependência é obrigatória.
- Domínio deve ser isolado de detalhes de persistência.

## Código
- DRY (Don't Repeat Yourself).
- KISS (Keep It Simple, Stupid).

## UI (Fase Final)
- IMPORTANTE: Após a conclusão das camadas de dados e lógica, o usuário deve ser instruído a realizar os ajustes finos na UI manualmente para garantir a melhor experiência estética.
EOF
        echo "✅ Projeto inicializado com sucesso! Idioma: $LANG"
        echo "Dica: Use 'wz roadmap' para ver sua primeira tarefa."
        ;;
    install)
        echo "🚀 Instalando WZ CLI no diretório do usuário..."
        mkdir -p "$HOME/.local/bin"
        cp "$0" "$HOME/.local/bin/wz"
        chmod +x "$HOME/.local/bin/wz"
        echo "✅ Instalado em ~/.local/bin/wz! Certifique-se que este diretório está no seu PATH."
        ;;
    *)
        echo "Uso: wz {plan|assist|test|view|roadmap|rules|install} [id]"
        ;;
esac
