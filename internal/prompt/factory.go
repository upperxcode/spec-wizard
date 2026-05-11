package prompt

import (
	"fmt"
	"strings"
)

type PromptFactory struct{}

func NewPromptFactory() *PromptFactory {
	return &PromptFactory{}
}

func (f *PromptFactory) GenerateExecutionPrompt(ctx ExecutionContext) string {
	criteria := ""
	for i, c := range ctx.AcceptanceCriteria {
		criteria += fmt.Sprintf("%d. %s\n", i+1, c)
	}

	stackRules := ""
	if len(ctx.StackRules) > 0 {
		stackRules = strings.Join(ctx.StackRules, "\n")
	}

	refFiles := ""
	if len(ctx.RelatedFiles) > 0 {
		refFiles = strings.Join(ctx.RelatedFiles, "\n")
	}

	testFilesStr := ""
	if len(ctx.TestFiles) > 0 {
		testFilesStr = strings.Join(ctx.TestFiles, "\n")
	}

	return fmt.Sprintf(ExecutionPromptTemplate,
		ctx.Language,
		stackRules,
		ctx.Skills,
		ctx.Spec,
		ctx.ProjectName,
		ctx.ProjectName,
		refFiles,
		testFilesStr,
		ctx.SprintID,
		ctx.SprintGoal,
		ctx.TaskTitle,
		ctx.TaskDescription,
		ctx.TaskSpec,
		ctx.UserQuestion,
		criteria,
		ctx.FileTree,
		ctx.Dependencies,
		ctx.ApiContracts,
		refFiles,
		ctx.TestLogs,
		ctx.TestFailPrompt,
		ctx.KnowledgeBase,
		ctx.TestCommand,
		ctx.Architecture,
	)
}

func (f *PromptFactory) GenerateAuditTaskPrompt(ctx ExecutionContext) string {
	criteria := strings.Join(ctx.AcceptanceCriteria, "\n")

	return fmt.Sprintf(AuditTaskPromptTemplate,
		ctx.Language,
		ctx.ProjectName,
		ctx.TaskTitle,
		ctx.TaskDescription,
		criteria,
		ctx.Spec, // Usado como Architectural Context em alguns casos
		ctx.FileTree,
		ctx.KnowledgeBase, // Usado como Root Evidence
	)
}

func (f *PromptFactory) GenerateAcceptanceCriteriaAuditPrompt(ctx ExecutionContext) string {
	criteria := ""
	for i, c := range ctx.AcceptanceCriteria {
		criteria += fmt.Sprintf("%d. %s\n", i+1, c)
	}

	testFilesStr := ""
	if len(ctx.TestFiles) > 0 {
		testFilesStr = strings.Join(ctx.TestFiles, "\n")
	}

	return fmt.Sprintf(AcceptanceCriteriaAuditPromptTemplate,
		ctx.TaskTitle,
		ctx.TaskDescription,
		criteria,
		testFilesStr,
		ctx.FileTree,
		ctx.TestLogs,
	)
}

func (f *PromptFactory) CreateRoadmapPrompt(ctx RoadmapContext) string {
	patternStr := strings.Join(ctx.Patterns, ", ")
	strategy := fmt.Sprintf("Architecture: %s\nData: %s\nState: %s\nCustom: %s\nExtra: %s",
		ctx.Architecture, ctx.DataStrategy, ctx.StateManagement, ctx.Customization, ctx.AdditionalInstructions)

	return fmt.Sprintf(RoadmapPromptTemplate,
		ctx.Language,
		patternStr,
		strategy,
		ctx.UserLanguage,
		patternStr,
	)
}

func (f *PromptFactory) CreateModuleMappingPrompt(ctx RoadmapContext, criticalContext string) string {
	return fmt.Sprintf(ModuleMappingPromptTemplate,
		ctx.Language,
		ctx.ProjectName,
		ctx.PRD,
		ctx.Spec,
		criticalContext,
		ctx.ProjectName,
	)
}

func (f *PromptFactory) CreateUpdateRoadmapPrompt(ctx RoadmapContext, fileTree, dossier string) string {
	patternStr := strings.Join(ctx.Patterns, ", ")
	return fmt.Sprintf(UpdateRoadmapPromptTemplate,
		ctx.Language,
		ctx.ProjectName,
		ctx.PRD,
		ctx.Spec,
		ctx.Skills,
		fileTree,
		ctx.ExistingRoadmap,
		dossier,
		ctx.AdditionalInstructions,
		ctx.UserLanguage,
		patternStr,
	)
}

func (f *PromptFactory) GenerateInterpretationPrompt(lang string, technicalFacts map[string]interface{}, userLang string) string {
	inferred := technicalFacts["inferred_properties"]
	tree := technicalFacts["file_tree"]
	meta := technicalFacts["metadata"]
	patterns := technicalFacts["plugin_patterns"]

	return fmt.Sprintf(InterpretationPromptTemplate,
		inferred,
		lang,
		tree,
		meta,
		patterns,
	)
}

func (f *PromptFactory) GenerateAnchoringPrompt(lang string, facts interface{}, config string) string {
	return fmt.Sprintf(AnchoringPromptTemplate,
		lang,
		facts,
		config,
	)
}

func (f *PromptFactory) GenerateResearchPrompt(resp map[string]interface{}) string {
	techs := resp["technologies"]
	name := resp["project_name"]

	return fmt.Sprintf(ResearchPromptTemplate,
		techs,
		name,
	)
}
func (f *PromptFactory) GenerateTaskSpecBootstrap(ctx ExecutionContext) string {
	criteria := ""
	for _, c := range ctx.AcceptanceCriteria {
		criteria += fmt.Sprintf("- %s\n", c)
	}

	return fmt.Sprintf(TaskSpecBootstrapTemplate,
		ctx.ProjectName,
		ctx.Architecture,
		ctx.Language,
		ctx.Summary, // Usando summary como excerpt do global spec
		ctx.SprintGoal,
		ctx.TaskTitle,
		ctx.TaskDescription,
		criteria,
		ctx.ProjectName, // Import Matrix module prefix
		ctx.ProjectName, // Module name
	)
}
