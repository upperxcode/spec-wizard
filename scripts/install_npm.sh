#!/bin/bash

# Cores para o terminal
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # Sem cor

# Pega o diretório base do projeto (um nível acima de scripts/)
BASE_DIR=$(cd "$(dirname "$0")/.." && pwd)

echo -e "${BLUE}🧙‍♂️ Instalando Spec Wizard MCP via Wrapper NPM...${NC}"

# 1. Verifica dependências
echo -e "\n${YELLOW}Step 1: Verificando dependências...${NC}"
command -v go >/dev/null 2>&1 || { echo "❌ Go não encontrado."; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "❌ NPM não encontrado."; exit 1; }

# 2. Compila o MCP Server
echo -e "\n${YELLOW}Step 2: Compilando o MCP Server...${NC}"
cd "$BASE_DIR"
mkdir -p bin
go build -o bin/mcp-wizard cmd/mcp-server/main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ MCP Server compilado em bin/mcp-wizard${NC}"
else
    echo "❌ Erro na compilação do Go."
    exit 1
fi

# 3. Configura binário local
echo -e "\n${YELLOW}Step 3: Instalando binário em ~/.local/bin...${NC}"
# Para processos antigos para evitar "Text file busy"
pkill mcp-wizard || true
sleep 1
mkdir -p "$HOME/.local/bin"
cp bin/mcp-wizard "$HOME/.local/bin/"
chmod +x "$HOME/.local/bin/mcp-wizard"

# Cria link simbólico 'wz' para facilitar o uso no terminal
ln -sf "$HOME/.local/bin/mcp-wizard" "$HOME/.local/bin/wz"

echo -e "${GREEN}✅ mcp-wizard e link 'wz' atualizados em ~/.local/bin/${NC}"
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo -e "${YELLOW}⚠️ Aviso: ~/.local/bin não está no seu PATH. Adicione-o ao seu .bashrc ou .zshrc.${NC}"
fi

# 4. Instalação Global do Wrapper NPM e Configurações
echo -e "\n${YELLOW}Step 4: Instalando Wrapper NPM e Configurações permanentemente...${NC}"
GLOBAL_DATA_DIR="$HOME/.local/share/spec-wizard"
GLOBAL_WRAPPER_DIR="$GLOBAL_DATA_DIR/npm"
GLOBAL_CONFIG_DIR="$GLOBAL_DATA_DIR/configs"

mkdir -p "$GLOBAL_WRAPPER_DIR"
mkdir -p "$GLOBAL_CONFIG_DIR"

# Copia Wrapper
cp mcp-wrapper-npm/package.json "$GLOBAL_WRAPPER_DIR/"
cp mcp-wrapper-npm/index.js "$GLOBAL_WRAPPER_DIR/"

# Copia Configurações (Traduções)
cp -r configs/* "$GLOBAL_CONFIG_DIR/"

cd "$GLOBAL_WRAPPER_DIR"
npm install --silent
echo -e "${GREEN}✅ Wrapper instalado em $GLOBAL_WRAPPER_DIR${NC}"
echo -e "${GREEN}✅ Configurações instaladas em $GLOBAL_CONFIG_DIR${NC}"

echo -e "\n${GREEN}==========================================${NC}"
echo -e "${GREEN}🎉 INSTALAÇÃO GLOBAL CONCLUÍDA!${NC}"
echo -e "${GREEN}==========================================${NC}"
echo -e "\n${BLUE}Configuração para o GoClaw (Whitelist npx):${NC}"
echo -e "Command: ${YELLOW}npx${NC}"
echo -e "Args:    ${YELLOW}-y $GLOBAL_WRAPPER_DIR${NC}"
echo -e "\n${BLUE}Nota:${NC} Agora você pode mover ou deletar a pasta do código fonte,"
echo -e "o Spec Wizard continuará funcionando no seu editor!"
echo -e "=========================================="
