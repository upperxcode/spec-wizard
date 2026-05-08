#!/bin/bash

# Spec Wizard - Integração Completa
# Script para rodar Go Server + Dashboard + Sensores

set -e

echo "🧙‍♂️ Spec Wizard - Iniciando Integração Completa"
echo "=================================================="

# Cores
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 1. Compilar Go Server
echo -e "${BLUE}📦 Passo 1: Compilando Go Server...${NC}"
cd "$(dirname "$0")"
go build -o main ./cmd/main.go
echo -e "${GREEN}✅ Go Server compilado${NC}"

# 2. Iniciar Go Server
echo -e "${BLUE}🚀 Passo 2: Iniciando Go Server em :8080...${NC}"
./main &
GO_PID=$!
echo -e "${GREEN}✅ Go Server iniciado (PID: $GO_PID)${NC}"

# 3. Aguardar servidor ficar pronto
sleep 2
echo -e "${BLUE}🔍 Testando conexão com API...${NC}"
if curl -s http://localhost:8080/api/languages > /dev/null; then
  echo -e "${GREEN}✅ API respondendo${NC}"
else
  echo -e "${YELLOW}⚠️  API ainda não respondendo, aguardando...${NC}"
  sleep 3
fi

# 4. Iniciar Dashboard
echo -e "${BLUE}🎨 Passo 3: Instalando dependências do Dashboard...${NC}"
cd ts-client

if [ ! -d "node_modules" ]; then
  npm install
  echo -e "${GREEN}✅ Dependências instaladas${NC}"
else
  echo -e "${GREEN}✅ node_modules já existe${NC}"
fi

echo -e "${BLUE}🎨 Passo 4: Iniciando Dashboard em :5173...${NC}"
npm run dev &
DASHBOARD_PID=$!
echo -e "${GREEN}✅ Dashboard iniciado (PID: $DASHBOARD_PID)${NC}"

# 5. Informações de acesso
echo ""
echo "=================================================="
echo -e "${GREEN}🎉 INTEGRAÇÃO COMPLETA!${NC}"
echo "=================================================="
echo ""
echo -e "${BLUE}Serviços Ativos:${NC}"
echo -e "  🔌 API Go Server:    ${YELLOW}http://localhost:8080${NC}"
echo -e "  🎨 Dashboard React:  ${YELLOW}http://localhost:5173${NC}"
echo -e "  🧠 LM Studio Local:  ${YELLOW}http://localhost:1234/v1${NC} (verifique se está rodando)"
echo ""
echo -e "${BLUE}Próximas Ações:${NC}"
echo -e "  1. Abra ${YELLOW}http://localhost:5173${NC} no navegador"
echo -e "  2. Selecione o caminho do seu projeto Flutter"
echo -e "  3. Escolha o padrão arquitetural (MVVM, CRUD, etc.)"
echo -e "  4. Veja o roadmap e execute tarefas"
echo ""
echo -e "${BLUE}Parar Serviços:${NC}"
echo -e "  ${YELLOW}kill $GO_PID $DASHBOARD_PID${NC}"
echo ""
echo "=================================================="
echo ""

# Manter script rodando
wait
