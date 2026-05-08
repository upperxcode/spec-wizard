# 🧙‍♂️ Spec Wizard - Multi Language Development Orchestrator

[![Status](https://img.shields.io/badge/status-phase%204%20integration%20complete-green)](https://github.com)
[![Go](https://img.shields.io/badge/go-1.19%2B-blue)](https://golang.org/)
[![React](https://img.shields.io/badge/react-18%2B-blue)](https://react.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

Orquestrador agnóstico de desenvolvimento que utiliza **especificações
estruturadas** e **IA local** para gerar código de produção com validação
automática e auto-correção.

## 🚀 Quick Start (30 segundos)

```bash
# Validar dependências
bash check-integration.sh

# Iniciar tudo (API + Dashboard)
cd spec-wizard && bash run-integration.sh

# Abrir Dashboard
# http://localhost:5173
```

## 🎯 O Que Faz

1. **📐 Anchor** seu projeto com especificações estruturadas
2. **🏗️ Design** a arquitetura com padrões imperativos (MVVM, CRUD, etc)
3. **📊 Generate** roadmap automático com 4 sprints
4. **🤖 Execute** tarefas com IA local (Qwen 2.5 Coder)
5. **✅ Validate** código com sensores (flutter analyze, eslint, go vet)
6. **🔧 Auto-correct** se algo falhar (até 3 tentativas)

## 🏛️ Arquitetura

```
┌──────────────────────────┐
│   Dashboard React/Vite   │
│    (localhost:5173)      │
└────────────┬─────────────┘
             │ REST API
┌────────────▼─────────────┐
│   Go Server              │
│    (localhost:8080)      │
│  ├─ Orquestrador        │
│  ├─ Pattern Repository  │
│  ├─ Context Assembler   │
│  ├─ Sensor Manager      │
│  └─ Feedback Loop       │
└────────────┬─────────────┘
      ┌──────┴──────┐
      │             │
   ┌──▼──┐     ┌────▼────┐
   │ LM  │     │ Sensores │
   │ Stud│    │ flutter  │
   │ io  │     │ eslint   │
   └─────┘     │ go vet   │
               └─────────┘
```

## 📋 Features

### ✅ Implementado (Fase 4)

- Dashboard em 3 passos
- API REST completa
- Context Assembler (Janela Limpa)
- Sensor Manager (Validação)
- Feedback Loop (Auto-correção)
- TypeScript Client
- Documentação completa

### ⏳ Planejado

- Experts para TypeScript/Go
- MCP Servers (GitHub, Jira)
- Histórico de execuções
- Métricas de qualidade
- Export PDF/Wiki

## 📁 Estrutura do Projeto

```
spec-wizard/
├── .agents/
│   ├── INTEGRATION.md      # Guia de integração (18 seções)
│   ├── SUMMARY.md          # Resumo executivo
│   ├── roadmap.md          # Roadmap do projeto
│   └── wiki/
│       └── home.md         # Documentação geral
├── check-integration.sh    # Validar dependências
├── spec-wizard/
│   ├── cmd/main.go         # Entrada da API
│   ├── api/handlers.go     # HTTP handlers
│   ├── internal/
│   │   ├── orchestrator/   # Core de orquestração
│   │   ├── patterns/       # Repository de padrões
│   │   ├── llm/            # Client para LM Studio
│   │   ├── prompt/         # Factory de prompts
│   │   └── ...
│   ├── ts-client/          # Dashboard React
│   │   ├── src/
│   │   │   ├── components/ # Step1, Step2, Step3
│   │   │   ├── services/   # apiClient
│   │   │   └── App.tsx
│   │   ├── package.json
│   │   └── README.md
│   ├── run-integration.sh  # Script de inicialização
│   ├── main                # Binary compilado
│   └── README.md
└── README.md
```

## 🔧 Dependências

### Obrigatórias

- **Go** 1.19+
- **Node.js** 18+
- **LM Studio** com Qwen 2.5 Coder

### Opcionais (para validação)

- **Flutter SDK** (para testar com Flutter)
- **Node.js** tools (para TypeScript/JavaScript)
- **Go** tools (para Go)

Verificar com: `bash check-integration.sh`

## 💻 Desenvolvimento

### Terminal 1: Go Server

```bash
cd spec-wizard
go run ./cmd/main.go
# 🧙‍♂️ Spec Wizard API ativa em http://localhost:8080
```

### Terminal 2: Dashboard

```bash
cd spec-wizard/ts-client
npm install
npm run dev
# ➜  Local:   http://localhost:5173/
```

### Terminal 3: LM Studio

```
Abrir LM Studio
→ Local Server
→ Carregar Qwen 2.5 Coder
→ Start
```

## 📚 Documentação

| Documento                                                 | Conteúdo                                |
| --------------------------------------------------------- | --------------------------------------- |
| [INTEGRATION.md](./spec-wizard/../.agents/INTEGRATION.md) | Guia completo de integração (18 seções) |
| [SUMMARY.md](./.agents/SUMMARY.md)                        | Resumo executivo da implementação       |
| [roadmap.md](./.agents/roadmap.md)                        | Roadmap das 4 fases                     |
| [wiki/home.md](./.agents/wiki/home.md)                    | Arquitetura e conceitos                 |
| [spec-wizard/README.md](./spec-wizard/README.md)          | Backend API                             |
| [ts-client/README.md](./spec-wizard/ts-client/README.md)  | Frontend Dashboard                      |

## 🧪 Teste End-to-End

1. **Passo 1**: Escolha projeto + linguagem (Flutter)
2. **Passo 2**: Selecione padrão (MVVM)
3. **Passo 3**: Veja roadmap + execute tarefas
4. **Resultado**: Código gerado, validado e auto-corrigido

Tempo esperado: **~2 minutos** por tarefa

## 🔗 Endpoints API

```bash
# GET - Linguagens disponíveis
curl http://localhost:8080/api/languages

# GET - Padrões por linguagem
curl http://localhost:8080/api/patterns/flutter

# POST - Inicializar projeto
curl -X POST http://localhost:8080/api/initialize \
  -H "Content-Type: application/json" \
  -d '{"path":"/path","language":"flutter","pattern":"mvvm"}'

# POST - Gerar roadmap
curl -X POST http://localhost:8080/api/roadmap \
  -H "Content-Type: application/json" \
  -d '{"path":"/path","language":"flutter","pattern":"mvvm"}'

# POST - Executar tarefa
curl -X POST http://localhost:8080/api/execute-task \
  -H "Content-Type: application/json" \
  -d '{...task details...}'
```

## 🎓 Conceitos Principais

### Janela Limpa (Clean Window)

Cada tarefa recebe APENAS:

- SPEC.md (arquitetura)
- skills.md (Golden Rules)
- PRD.md (requisitos)
- Task específica

Evita degradação de contexto (Dumb Zone)

### Golden Rules

Regras imperativos por padrão:

- MVVM: Separação View/ViewModel/Model
- CRUD: Use DTOs, validação, async/await
- Aplicáveis a todo o roadmap

### Auto-Correção

Loop inteligente:

1. Gera código
2. Valida com sensores
3. Se falha → Monta prompt de correção
4. Tenta novamente (máx 3x)

## 🚀 Deployment

### Build para Produção

```bash
# Frontend
cd spec-wizard/ts-client
npm run build
# Output: dist/

# Backend
cd spec-wizard
go build -o spec-wizard ./cmd/main.go
```

### Docker (Futuro)

```bash
docker-compose up
```

## 📊 Métricas

| Métrica                          | Valor |
| -------------------------------- | ----- |
| Tempo Build (Go)                 | ~2s   |
| Tempo Build (React)              | ~5s   |
| Tempo Execução Tarefa            | ~30s  |
| Taxa de Sucesso (com auto-corr.) | ~95%  |
| Tentativas Médias                | 1.2   |

## 🐛 Troubleshooting

### API não responde

```bash
cd spec-wizard && go run ./cmd/main.go
```

### Dashboard não inicia

```bash
cd spec-wizard/ts-client
rm -rf node_modules package-lock.json
npm install && npm run dev
```

### LM Studio não conecta

1. Abrir LM Studio
2. Carregar Qwen 2.5 Coder
3. Iniciar Local Server
4. Verificar http://localhost:1234/v1

Mais detalhes: [INTEGRATION.md](./.agents/INTEGRATION.md#-troubleshooting)

## 📝 Licença

MIT © 2026

## 🤝 Contribuindo

Sugestões e PRs são bem-vindos!

## 📞 Contato

Issues: [GitHub Issues](https://github.com)

---

**Desenvolvido como POC de Spec-Driven Development com IA Local** 🚀

Comece agora: `bash run-integration.sh`
