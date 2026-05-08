# 🧙‍♂️ Spec Wizard Dashboard

Cliente React/Vite para o Spec Wizard - Orquestrador de Desenvolvimento orientado a Especificações.

## 🚀 Instalação e Execução

### Pré-requisitos
- Node.js 18+ 
- npm ou yarn
- Go Server rodando em `http://localhost:8080`

### Setup

```bash
cd ts-client

# Instalar dependências
npm install

# Rodar em desenvolvimento
npm run dev

# Build para produção
npm run build

# Preview do build
npm run preview
```

O Dashboard estará disponível em `http://localhost:5173`

## 📋 Funcionalidades

### 🏗️ 3 Passos de Orquestração

#### **Passo 1: Projeto & Linguagem**
- Seleção do caminho do projeto
- Escolha da linguagem de programação
- Validação via API

#### **Passo 2: Padrão Arquitetural**
- Lista de padrões disponíveis por linguagem
- Visualização das "Golden Rules"
- Seleção interativa

#### **Passo 3: Roadmap & Execução**
- Visualização das Sprints e Tarefas
- Botão para executar cada tarefa
- Logs de execução em tempo real
- Integração com LLM local

## 🔗 Integração com API Go

### Endpoints Utilizados

```
GET  /api/languages              → Lista de linguagens
GET  /api/patterns/{lang}        → Padrões por linguagem
POST /api/initialize             → Inicializar projeto
POST /api/roadmap                → Gerar roadmap
POST /api/execute-task           → Executar tarefa
```

### Proxy Configuration

O Vite está configurado com proxy automático para `/api` → `http://localhost:8080`:

```typescript
// vite.config.ts
proxy: {
  '/api': {
    target: 'http://localhost:8080',
    changeOrigin: true
  }
}
```

## 📁 Estrutura do Projeto

```
ts-client/
├── src/
│   ├── components/
│   │   ├── Step1.tsx          # Seleção de projeto
│   │   ├── Step2.tsx          # Seleção de padrão
│   │   └── Step3.tsx          # Execução de tarefas
│   ├── services/
│   │   └── apiClient.ts       # Cliente HTTP
│   ├── App.tsx                # Componente principal
│   ├── App.css                # Estilos
│   └── main.tsx               # Entrada
├── index.html
├── vite.config.ts
├── tsconfig.json
└── package.json
```

## 🎨 Componentes

### `apiClient.ts`
Cliente HTTP com tipos TypeScript para todas as rotas:
- `getLanguages()` - Obtém linguagens
- `getPatterns(language)` - Obtém padrões
- `initializeProject(path, language, pattern)` - Inicializa
- `generateRoadmap(path, language, pattern)` - Gera plano
- `executeTask(request)` - Executa tarefa

### `Step1.tsx`
- Input para caminho do projeto
- Dropdown de linguagens
- Validação básica

### `Step2.tsx`
- Grid de padrões arquiteturais
- Visualização de golden rules
- Seleção interativa com feedback visual

### `Step3.tsx`
- Visualização do roadmap
- Accordion de tarefas por sprint
- Execução de tarefas com feedback
- Logs de execução expansíveis

## 🔧 Desenvolvimento

### Adicionar Nova Rota de API

```typescript
// src/services/apiClient.ts
export const apiClient = {
  async myNewFunction(): Promise<MyType> {
    const response = await axios.post(`${API_BASE}/my-endpoint`, data)
    return response.data
  }
}
```

### Adicionar Novo Componente

```typescript
// src/components/MyComponent.tsx
import React from 'react'

const MyComponent: React.FC<MyProps> = ({ prop1 }) => {
  return <div className="my-component">...</div>
}

export default MyComponent
```

## 🎯 Fluxo de Uso

1. **Usuário inicia**: Dashboard em `localhost:5173`
2. **Passo 1**: Seleciona caminho do projeto + linguagem
3. **Passo 2**: Escolhe padrão arquitetural (MVVM, CRUD, etc.)
4. **API**: Go Server cria `.spec-wizard/` com PRD, SPEC, skills.md
5. **Passo 3**: Dashboard mostra roadmap com sprints e tarefas
6. **Execução**: Usuário clica "▶️ Executar" em cada tarefa
7. **LLM**: LM Studio gera código em "Janela Limpa"
8. **Sensores**: Validam código (flutter analyze, etc.)
9. **Auto-correção**: Se falhar, loop automático até 3 tentativas
10. **Resultado**: Log de execução salvo em `.spec-wizard/task-logs/`

## 🚨 Troubleshooting

### API não responde
- Verificar se Go Server está rodando: `http://localhost:8080/api/languages`
- Verificar logs do servidor Go

### CORS errors
- Vite proxy está configurado em `vite.config.ts`
- Em produção, adicionar CORS headers no Go

### LLM Studio não encontrado
- Garantir que LM Studio está rodando em `http://localhost:1234/v1`
- Verificar logs do servidor Go para detalhes

## 📊 Melhorias Futuras

- [ ] Modo offline
- [ ] Histórico de execuções
- [ ] Undo/Redo de tarefas
- [ ] Integração com Git
- [ ] Dashboard de métricas
- [ ] Export de documentação

## 📝 Licença

MIT
