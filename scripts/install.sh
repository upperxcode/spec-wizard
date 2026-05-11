#!/bin/bash

# Cores para output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🧙‍♂️ Iniciando Instalação do Spec Wizard...${NC}"

# 1. Definição de Caminhos
BIN_DEST="$HOME/.local/bin"
RESOURCES_DEST="$HOME/.spec-wizard"
CONFIG_DEST="$HOME/.config/spec-wizard"

echo -e "📂 Criando estrutura de diretórios..."
mkdir -p "$BIN_DEST"
mkdir -p "$RESOURCES_DEST/plugins"
mkdir -p "$RESOURCES_DEST/logs"
mkdir -p "$RESOURCES_DEST/themes"
mkdir -p "$RESOURCES_DEST/ui"
mkdir -p "$CONFIG_DEST"

# Tenta matar instância rodando para evitar "text file busy" ou porta presa
if [ -f "$BIN_DEST/spec-wizard" ]; then
    echo -e "🛑 ${YELLOW}Verificando instâncias anteriores...${NC}"
    "$BIN_DEST/spec-wizard" --kill > /dev/null 2>&1 || true
    # Mata qualquer processo de plugin órfão para liberar os binários
    pkill -f ".spec-wizard/plugins/.*/expert" || true
fi

# 2. Compilação do App Principal
echo -e "${YELLOW}⚙️ Compilando Spec Wizard...${NC}"
go build -o spec-wizard cmd/main.go
if [ $? -ne 0 ]; then
    echo "❌ Falha na compilação do binário principal."
    exit 1
fi

# 3. Compilação dos Plugins
echo -e "${YELLOW}⚙️ Compilando Plugins...${NC}"
chmod +x scripts/build_plugins.sh
./scripts/build_plugins.sh
if [ $? -ne 0 ]; then
    echo "❌ Falha na compilação dos plugins."
    exit 1
fi

# 4. Build da Interface Web
echo -e "${YELLOW}⚙️ Compilando Interface Web...${NC}"
cd spec-wizard-ui || exit
npm install --silent
npm run build --silent
if [ $? -ne 0 ]; then
    echo "❌ Falha no build da interface web."
    cd ..
    exit 1
fi
cd ..

# 4. Instalação dos Arquivos
echo -e "🚚 Movendo binários e recursos..."

# Binário principal
mv spec-wizard "$BIN_DEST/"

# Experts Config
cp config/experts.yaml "$RESOURCES_DEST/"

# Plugins (binários compilados)
# Mantém a estrutura de subpastas para compatibilidade com o experts.yaml
mkdir -p "$RESOURCES_DEST/plugins/go-expert"
mkdir -p "$RESOURCES_DEST/plugins/flutter-expert"

cp plugins/go-expert/expert "$RESOURCES_DEST/plugins/go-expert/expert"
cp plugins/flutter-expert/expert "$RESOURCES_DEST/plugins/flutter-expert/expert"

# Temas
if [ -d "themes" ]; then
    cp -r themes/* "$RESOURCES_DEST/themes/"
fi

# Interface Web (dist)
echo -e "🌐 Instalando GUI Web..."
rm -rf "$RESOURCES_DEST/ui/*"
cp -r spec-wizard-ui/dist/* "$RESOURCES_DEST/ui/"

# 5. Permissões
chmod +x "$BIN_DEST/spec-wizard"
chmod +x "$RESOURCES_DEST/plugins/go-expert" 2>/dev/null
chmod +x "$RESOURCES_DEST/plugins/flutter-expert" 2>/dev/null

echo -e "${GREEN}✅ Instalação Concluída!${NC}"
echo -e "🚀 Agora você pode rodar '${BLUE}spec-wizard${NC}' de qualquer lugar."
echo -e "💡 Certifique-se de que '${YELLOW}$BIN_DEST${NC}' está no seu PATH."

# Sugestão de PATH
if [[ ":$PATH:" != *":$BIN_DEST:"* ]]; then
    echo -e "\n${YELLOW}Aviso:${NC} $BIN_DEST não parece estar no seu PATH."
    echo "Adicione a seguinte linha ao seu ~/.bashrc ou ~/.zshrc:"
    echo -e "  ${BLUE}export PATH=\"\$PATH:\$HOME/.local/bin\"${NC}"
fi
