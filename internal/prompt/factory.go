package prompt

import (
	"fmt"
	"strings"
)

type PromptFactory struct{}

// NewPromptFactory cria uma nova instância da fábrica de prompts
func NewPromptFactory() *PromptFactory {
	return &PromptFactory{}
}

type ExecutionContext struct {
	Skills                 string
	Spec                   string
	Summary                string
	Roadmap                string
	PrdExcerpt             string
	SprintID               int
	SprintGoal             string
	TaskTitle              string
	TaskDescription        string
	AcceptanceCriteria     []string
	ReferenceFiles         []string
	Architecture           string
	FileTree               string
	Dependencies           string
	ApiContracts           string
	UserQuestion           string
	KnowledgeBase          string
}

// GenerateExecutionPrompt cria o prompt definitivo para o Harness de Execução
func (f *PromptFactory) GenerateExecutionPrompt(ctx ExecutionContext) string {
	criteria := ""
	for _, c := range ctx.AcceptanceCriteria {
		criteria += fmt.Sprintf("- %s\n", c)
	}

	refFiles := "None provided."
	if len(ctx.ReferenceFiles) > 0 {
		refFiles = strings.Join(ctx.ReferenceFiles, ", ")
	}

	return fmt.Sprintf(`**AUTHORITATIVE PERSONA:**
"Act as a Senior Software Engineer specializing in %s. You are operating under the Spec Wizard Rigid Specification Protocol. Your language must be technical, direct, and free of uncertainty."

**GOLDEN RULES (SYSTEM):**
%s

**ARCHITECTURAL CONTRACT:**
%s

**CURRENT TASK CONTEXT (SPRINT %d):**
"Sprint Goal: %s"
"Specific Task: %s"
"Description: %s"

**ADDITIONAL USER QUESTION/CONTEXT:**
"%s"

**ACCEPTANCE CRITERIA (MANDATORY):**
%s
**EXECUTION METADATA:**
- **File Tree:** 
%s
- **Dependency Check:** 
%s
- **API Contracts:** 
%s

**REFERENCE FILES (IF ANY):**
"%s"

%s

**FINAL COMMAND:**
"Implement the functionality strictly following the %s pattern defined in the SPEC. Respond ONLY the code or file creation instructions. If the code violates any rule in skills.md, it will be rejected by the Validation Harness."`,
		ctx.Architecture,
		ctx.Skills,
		ctx.Spec,
		ctx.SprintID,
		ctx.SprintGoal,
		ctx.TaskTitle,
		ctx.TaskDescription,
		ctx.UserQuestion,
		criteria,
		ctx.FileTree,
		ctx.Dependencies,
		ctx.ApiContracts,
		refFiles,
		ctx.KnowledgeBase,
		ctx.Architecture)
}

// GenerateAnchoringPrompt creates the command for the AI to generate PRD and SPEC
func (f *PromptFactory) GenerateAnchoringPrompt(projectType string, facts map[string]interface{}, rawFile string) string {
	return fmt.Sprintf(`
You are a Senior Architect specializing in %s.
Analyze these technical facts extracted from the project:
%v

And the original configuration file content:
%s

Based on this, generate a detailed PRD (business goals) description 
and a technical specification (SPEC) for the project foundation.
Return the result in structured JSON format so I can apply it to my templates.
RESPONSE EVERYTHING IN ENGLISH.`,
		projectType, facts, rawFile)
}

// GenerateResearchPrompt creates the prompt to transform technical facts into business requirements
func (f *PromptFactory) GenerateResearchPrompt(metadata map[string]interface{}) string {
	return fmt.Sprintf(`
		You are a senior Product Owner. Analyze these technical metadata extracted from a project:
		- Technologies: %v
		- Suggested Name: %v
		
		Based on this, generate a Product Requirement Document (PRD) in Markdown.
		The PRD must contain:
		1. Project Objective
		2. Target Audience
		3. Main Business Features
		Respond only with the Markdown content.
		RESPONSE EVERYTHING IN ENGLISH.`,
		metadata["detected_libs"], metadata["project_name"])
}

type RoadmapContext struct {
	Language               string
	Patterns               []string
	Architecture           string
	DataStrategy           string
	StateManagement        string
	Customization          string
	AdditionalInstructions string
	PRD                    string
	Spec                   string
	Skills                 string
	ExistingRoadmap        string
	UserLanguage           string
}

// CreateRoadmapPrompt generates a rich prompt for sprint slicing based on the chosen architecture
func (f *PromptFactory) CreateRoadmapPrompt(ctx RoadmapContext) string {
	patternStr := ""
	if len(ctx.Patterns) > 0 {
		patternStr = fmt.Sprintf("using the patterns: %s", strings.Join(ctx.Patterns, ", "))
	}

	architectureContext := fmt.Sprintf(`
SELECTED TECHNICAL STRATEGY:
- Architecture: %s
- Data Strategy: %s
- State Management: %s
- Customizations/Subtleties: %s
- Additional Instructions: %s
`, ctx.Architecture, ctx.DataStrategy, ctx.StateManagement, ctx.Customization, ctx.AdditionalInstructions)

	return fmt.Sprintf(`
You are a senior Tech Lead expert in %s %s.
Your task is to slice the development of a new project into 4 logical Sprints.

%s

MANDATORY STRUCTURE RULES:
1. Respond ONLY the raw JSON, no markdown code blocks, no explanations or introductions.
2. Each Sprint must have a clear technical goal.
3. Each Task MUST be a complete object (never a string).
4. Sprint IDs must be 1, 2, 3, 4.
5. Task IDs must be global incremental (1, 2, 3... until project end).

CONTENT RULES:
- Titles: Must be short and action-focused (e.g., "Configure Provider", "Create User Model").
- Descriptions: Must detail the technical "how", taking the technical strategy above into account.
- Acceptance Criteria: Must be lists of technical validations that can be verified via code or tests.
- LANGUAGE: ALL CONTENT (Titles, Descriptions, Criteria) MUST BE IN ENGLISH.

EXPECTED JSON SCHEMA:
{
  "language": "%s",
  "pattern": "%s",
  "sprints": [
    {
      "id": 1,
      "goal": "Technical goal of the sprint",
      "tasks": [
        {
          "id": 1,
          "title": "Descriptive title",
          "description": "Explanation of what needs to be done",
          "acceptance_criteria": ["validation 1", "validation 2"],
          "status": "pending"
        }
      ]
    }
  ]
}

Generate the roadmap now:`, ctx.Language, patternStr, architectureContext, ctx.Language, strings.Join(ctx.Patterns, "|"))
}

// CreateModuleMappingPrompt gera o prompt para a Fase 1 (Mapeamento de Módulos)
func (f *PromptFactory) CreateModuleMappingPrompt(ctx RoadmapContext, fileTree string) string {
	return fmt.Sprintf(`### ROLE: SENIOR SYSTEMS AUDITOR
Your mission is to perform a DEEP CRITICAL AUDIT of the %s project named "GCI".
Do not be optimistic. If you don't see specific implementation details for a requirement, it is NOT fully implemented.

### SOURCES:
1. PRD (Requirements): %s
2. SPEC (Architecture): %s
3. FILE TREE (Physical Evidence): %s

### INVESTIGATION STRATEGY:
1. **Visual Grouping**: Look at Sidebar, Drawer, or BottomNavigationBar components to identify high-level user modules.
2. **Logic Grouping**: Inspect Routing files (GoRouter, AutoRoute), Services, and Repositories. If multiple files share a prefix or domain folder, group them.
3. **Non-Visual Grouping**: Look at State Management (Bloc/Provider/Riverpod providers) and Dependency Injection (GetIt) to see which services are initialized together.
4. **Data Grouping**: Check Supabase/Database schema or API Client definitions for entity clusters (e.g., all endpoints related to 'condominium' or 'financial').

### AUDIT LOGIC (STRICT EVIDENCE):
1. **TRUST BUT VERIFY**: Use tools to read specific files (routers, services) to confirm groupings.
2. **PROGRESS CALCULATION**: Estimate progress for each sub-feature based on the existence of logic (e.g., "UI exists but Service is empty" = 30%%).
3. **EVIDENCE CHECK**: Only mark as "fully_implemented" if UI, Service, and Repository layers are found.
4. **NO HALLUCINATIONS**: If you cannot find evidence using tools, the feature is "pending".
5. **Status Definitions**:
   - **"fully_implemented"**: Evidence of controllers, models, and services covering all PRD requirements.
   - **"partially_implemented"**: Folder exists but lacks key files or implementation.
   - **"empty_pending"**: No corresponding folder/logic found.
- **Connections**: Map how modules interact (imports/dependencies).

### RESPONSE FORMAT (RAW JSON ONLY):
{
  "project_name": "GCI",
  "detected_modules": [
    {
      "name": "Module Name",
      "status": "fully_implemented | partially_implemented | pending",
      "progress_percentage": 0,
      "functionalities": ["List of sub-features found, e.g. 'Cadastro de Clientes'"],
      "grouped_view": "Module[sub1, sub2, sub3]",
      "missing_features": ["Feature required by PRD but NOT found in code"],
      "connections": ["Module X", "Service Y"],
      "evidence": "Path to main files"
    }
  ],
  "tech_stack": ["package1", "package2"],
  "architectural_health": "Critical comment on MVC/Patterns adherence and Technical Debt detected."
}

Generate the audit now:`, ctx.Language, ctx.PRD, ctx.Spec, fileTree)
}

// CreateUpdateRoadmapPrompt gera um prompt para atualizar o roadmap baseado no mapeamento anterior (Fase 2)
func (f *PromptFactory) CreateUpdateRoadmapPrompt(ctx RoadmapContext, fileTree string, moduleMapping string) string {
	return fmt.Sprintf(`### ROLE: SENIOR TECHNICAL AUDITOR & SYSTEMS ARCHITECT
You are performing a deep technical audit of an ongoing %s project named "GCI". 
Your goal is to REGENERATE a professional Roadmap that accurately reflects the current state and charts the path to completion.

### SOURCES OF TRUTH:
1. PRD (Requirements): %s
2. SPEC (Technical Specs): %s
3. SKILLS (Golden Rules): %s
4. FILE TREE (Physical Evidence): %s
5. EXISTING ROADMAP (Current Status): %s
6. PROJECT DOSSIER (Mapping Phase 1): %s

### AUDIT LOGIC (STRICT ENFORCEMENT):
- **ANTI-REDUNDANCY**: DO NOT suggest tasks like "Set up MVC", "Configure Provider", or "Create folders" if the DOSSIER/FILE TREE shows they are already there.
- **GAP-CENTRIC**: Compare the PRD Requirements with the "functionalities" found in the DOSSIER. If a PRD feature is listed in "missing_features", it MUST be a priority task in the Roadmap.
- **EVOLUTIONARY**: Focus on finishing "partially_implemented" modules and implementing "pending" ones.
- **TECHNICAL DEBT**: Include tasks for missing tests or architectural inconsistencies noted in the DOSSIER.

### OUTPUT RULES:
- Format: RAW JSON ONLY. No markdown blocks.
- PROGRESS RULE: Clearly state if a task is "Completing", "Refining" or "Starting from scratch" based on the DOSSIER.
- TITLES: Use the grouping format in titles when applicable, e.g., "Module[SubA, SubB]: Implementation Title".
- LANGUAGE: ALL CONTENT (Titles, Descriptions, Criteria) MUST BE IN ENGLISH.

### MANDATORY JSON STRUCTURE:
{
  "language": "%s",
  "pattern": "%s",
  "sprints": [
    {
      "id": 1,
      "goal": "Description of the next logical milestone",
      "tasks": [
        {
          "id": 1,
          "title": "Action-focused title",
          "priority": "HIGH | MEDIUM | LOW",
          "description": "Technical instructions on how to implement/fix",
          "acceptance_criteria": ["criteria 1", "criteria 2"],
          "status": "pending"
        }
      ]
    }
  ]
}

Generate the audited roadmap now:`, 
		ctx.Language, ctx.PRD, ctx.Spec, ctx.Skills, fileTree, ctx.ExistingRoadmap, moduleMapping, ctx.Language, strings.Join(ctx.Patterns, "|"))
}

// GenerateInterpretationPrompt creates the prompt for the AI to interpret an existing project
func (f *PromptFactory) GenerateInterpretationPrompt(lang string, facts map[string]interface{}, userLang string) string {
	inferred := facts["inferred_properties"]

	return fmt.Sprintf(`
### SENIOR ARCHITECT INSTRUCTION ###
You received a TECHNICAL DOSSIER of an existing project. 
Your mission is to complete the expert analysis to configure Spec Wizard.

FACTS INFERRED BY THE EXPERT (TREAT AS TRUTH):
%v

REAL PROJECT DATA:
- Language: %s
- Folder Structure: %v
- Metadata/Dependencies: %v
- Available Patterns in Plugin: %v

### MISSION GUIDELINES ###
1. The Expert has already tried to detect Architecture, State Management, and Data via heuristics.
2. YOUR MAIN FOCUS: 
   - Identify the business DOMAIN (e.g., Fintech, Healthtech, Real Estate ERP).
   - List Functional and Non-Functional REQUIREMENTS based on existing files.
   - Refine any field the Expert marked as "custom" or that you clearly see is wrong in the real data.
3. Do not invent details that do not exist in the code.
4. RESPONSE EVERYTHING IN ENGLISH.

MANDATORY RESPONSE (RAW JSON ONLY):
{
  "projectName": "Name found",
  "domain": "Business domain",
  "functionalRequirements": ["Requirement 1", "Requirement 2"],
  "nonFunctionalRequirements": ["Requirement 1"],
  "architecture": "Refine if necessary",
  "patterns": ["Selected IDs"],
  "dataStrategy": "Refine if necessary",
  "stateManagement": "Refine if necessary",
  "customization": "Observed subtle details"
}

Generate the expert JSON:`, inferred, lang, facts["structure"], facts["dependencies"], facts["plugin_patterns"])
}

func (f *PromptFactory) formatPatterns(patterns []string) string {
	if len(patterns) == 0 {
		return "None"
	}
	return "- " + strings.Join(patterns, "\n- ")
}
