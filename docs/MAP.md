# 🗺️ Mapa de Arquivos do Projeto

Este arquivo é gerado automaticamente pelo **Spec Wizard**. Ele mapeia cada arquivo físico à sua responsabilidade e tarefa de origem.

## 🏃 Sprint: 
### 🎯 Tarefa 0: 


| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

## 🏃 Sprint: 
### 🎯 Tarefa 1: Core[Logging]: Enforce Structured Logging
Replace all fmt.Println and log.Printf with internal/logger (slog) in mcp-server and main server. Ensure LOG_ANALITIC=true is respected.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 2: Core[Discovery]: Dynamic Path Resolution
Remove absolute fallback paths from server.log initialization. Implement logic to detect project root or use CWD as base for logs and db.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 3: Create Core Data Models
Define the foundational data schemas (User, Product, Order) using migrations. Focus on primary keys and required relationships.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

## 🏃 Sprint: 
### 🎯 Tarefa 1: Develop User Authentication Endpoints
Build the /api/auth routes including sign-up, login, and logout. Implement JWT token generation and validation using bcrypt for password hashing.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 2: Implement Product Listing API
Create GET endpoints for listing all products and filtering/sorting by category. Use database query optimization (indexing) where necessary.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 3: Lifecycle[Init]: Automated Roadmap Generation
Implement autonomous roadmap generation triggered by "wz init". Integrate RoadmapAuditPromptTemplate to analyze existing project state.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 4: Governance[Language]: Bilingual Protocol Implementation
Enforce English for all internal AI directives, code comments, and technical specs. Ensure user-facing responses follow user language preference.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

## 🏃 Sprint: 
### 🎯 Tarefa 1: Implement Authorization Middleware
Develop reusable middleware that checks JWT scope and user roles before allowing access to protected endpoints (e.g., Admin-only routes).

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 2: Develop Search & Filtering Logic (Advanced)
Enhance the product API to support fuzzy searching across multiple fields (name, description) and implementing complex relational joins.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 3: Implement State Persistence for User Preferences
Create a dedicated service/model to handle user-specific settings (e.g., default currency, notification preferences) that need retrieval upon login.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 5: Infra[Port]: Robust Process Termination
Refine killProcessOnPort logic to avoid killing browsers/UIs. Use strict TCP LISTEN check and process ownership validation.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

## 🏃 Sprint: 
### 🎯 Tarefa 1: Build Frontend Core Components
Develop reusable UI components (Header, ProductCard, Form) and integrate routing/state management library (e.g., Redux/Zustand) for the client side.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 2: Write Unit and Integration Tests
Develop comprehensive test coverage (minimum 80%) for all critical business logic paths, focusing on edge cases in payment processing and inventory checks.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

### 🎯 Tarefa 3: Setup Deployment CI/CD Pipeline
Configure GitHub Actions or equivalent pipeline to automatically run tests, build artifacts (frontend), and deploy the backend service upon merging to the main branch.

| Arquivo | Descrição |
| :--- | :--- |
| - | Nenhum arquivo mapeado ainda |

