# 🗺️ Spec Wizard Project Map

This file provides a high-level overview of the project structure and key components to avoid "blind flight" research.

## 🏗️ Architecture Overview

The Spec Wizard follows an **Orchestrator-Agent** pattern, where a central Go engine (the Brain) coordinates multiple Expert agents (Plugins) to execute a structured development roadmap.

## 📂 Key Directories & Files

### 🚀 Core (Go Backend)
- `cmd/mcp-server/`: Entry point for the Model Context Protocol (MCP) server.
- `internal/orchestrator/`: Core logic for task management, roadmap execution, and agent coordination.
  - `docs.go`: Technical specification and documentation handling.
- `internal/prompt/`: LLM prompt templates and persona definitions.
- `internal/utils/`: Shared utilities.
  - `i18n.go`: **Central Translation System (Backend)**.
- `config/`: Global and project-specific configuration handling.

### 🎨 Frontend (React/Vite)
- `spec-wizard-ui/`: The main dashboard UI.
  - `src/translations.js`: **Frontend Translation Map**.

### 🧩 Plugins (Experts)
- `plugins/`: Directory containing specific language/framework experts.
- `internal/adapters/`: Adapters to translate project state for specific plugins.

### 📝 Specifications & State
- `.spec-wizard/`: Hidden directory in target projects containing:
  - `config.json`: Persistent state of the project roadmap.
  - `SPEC.md`: Technical specification blueprint.
  - `PRD.md`: Product Requirement Document.

## 🔄 Workflow Logic
1. **Refine**: Architect agent creates a `task-N.md` contract.
2. **Prepare**: System injects the contract into the IDE context (English only for agents).
3. **Code**: Developer agent implements logic based on the contract.
4. **Audit**: System validates implementation against the contract.

---
*Note: Always update this map when adding new major components or changing architectural patterns.*
