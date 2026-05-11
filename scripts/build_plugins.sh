#!/bin/bash

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}🏗️ Spec Wizard - Plugin Builder${NC}"
echo "-----------------------------------"

# 1. Go Expert
if [ -d "plugins/go-expert" ]; then
    echo -e "${YELLOW}Compilando Go Expert...${NC}"
    cd plugins/go-expert
    go build -o expert main.go
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Go Expert pronto!${NC}"
    else
        echo -e "${RED}❌ Falha ao compilar Go Expert${NC}"
    fi
    cd ../..
fi

# 2. Flutter Expert (Go)
if [ -d "plugins/flutter-expert" ]; then
    echo -e "${YELLOW}Compilando Flutter Expert (Go)...${NC}"
    cd plugins/flutter-expert
    go build -o expert main.go
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Flutter Expert pronto!${NC}"
    else
        echo -e "${RED}❌ Falha ao compilar Flutter Expert${NC}"
    fi
    cd ../..
fi

echo "-----------------------------------"
echo -e "${GREEN}🌟 Todos os plugins processados!${NC}"
