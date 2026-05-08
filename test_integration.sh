#!/bin/bash

# Script de Teste de Integração - Spec Wizard
# Este script simula as chamadas de API feitas pelo Dashboard

BASE_URL="http://localhost:8080/api"
PROJECT_PATH="$(pwd)/test-project"

# 1. Limpa projeto de teste anterior
rm -rf "$PROJECT_PATH"
mkdir -p "$PROJECT_PATH"

echo "🚀 Iniciando teste de integração..."

# 2. Testa /api/initialize
echo "⚓ Chamando /api/initialize (Flutter + MVVM)..."
curl -s -X POST "$BASE_URL/initialize" \
     -H "Content-Type: application/json" \
     -d "{
           \"path\": \"$PROJECT_PATH\",
           \"language\": \"flutter\",
           \"pattern\": \"mvvm\"
         }"

echo -e "\n🔍 Verificando arquivos de ancoragem..."
if [ -f "$PROJECT_PATH/.spec-wizard/skills.md" ]; then
    echo "✅ skills.md criado."
    grep "MVVM" "$PROJECT_PATH/.spec-wizard/skills.md" > /dev/null && echo "✅ Regras do padrão injetadas corretamente."
else
    echo "❌ skills.md NÃO encontrado."
fi

echo -e "\n📝 Criando SPEC e PRD fakes para teste de execução..."
echo "# Especificação Técnica" > "$PROJECT_PATH/.spec-wizard/SPEC.md"
echo "# Requisitos de Negócio" > "$PROJECT_PATH/.spec-wizard/PRD.md"

# 3. Testa /api/execute-task (Simulação)
# Nota: Para este teste ser real, o LM Studio precisa estar rodando.
# Se não estiver, esperamos um erro 500, o que confirma que o handler tentou chamar o loop.
echo -e "\n🏃 Chamando /api/execute-task..."
curl -s -X POST "$BASE_URL/execute-task" \
     -H "Content-Type: application/json" \
     -d "{
           \"project_path\": \"$PROJECT_PATH\",
           \"sprint_id\": 1,
           \"task_id\": 1,
           \"task\": {
             \"id\": 1,
             \"title\": \"Criação do Model de Usuário\",
             \"description\": \"Crie uma classe User com id, nome e email em Dart.\",
             \"acceptance_criteria\": [\"Deve ter um construtor nomeado\", \"Deve suportar toMap()\"]
           },
           \"sprint\": {
             \"id\": 1,
             \"goal\": \"Setup de Autenticação\"
           }
         }"

echo -e "\n\n🏁 Teste finalizado. Verifique os logs acima."
