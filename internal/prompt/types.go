package prompt

type LibraryInfo struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	Mandatory bool   `json:"mandatory"`
	Disabled  bool   `json:"disabled"`
}

type ExecutionContext struct {
	Language           string        `json:"language"`
	ProjectName        string        `json:"project_name"`
	Skills             string        `json:"skills"`
	Spec               string        `json:"spec"`
	Summary            string        `json:"summary"`
	Roadmap            string        `json:"roadmap"`
	PrdExcerpt         string        `json:"prd_excerpt"`
	TaskSpec           string        `json:"task_spec"`
	SprintID           string        `json:"sprint_id"`
	SprintGoal         string        `json:"sprint_goal"`
	TaskTitle          string        `json:"task_title"`
	TaskDescription    string        `json:"task_description"`
	AcceptanceCriteria []string      `json:"acceptance_criteria"`
	StackRules         []string      `json:"stack_rules"`
	Architecture       string        `json:"architecture"`
	FileTree           string        `json:"file_tree"`
	Dependencies       string        `json:"dependencies"`
	ApiContracts       string        `json:"api_contracts"`
	UserQuestion       string        `json:"user_question"`
	KnowledgeBase      string        `json:"knowledge_base"`
	ReferenceFiles     []string      `json:"reference_files"`
	RelatedFiles       []string      `json:"related_files"`
	TestFiles          []string      `json:"test_files"`
	Libraries          []LibraryInfo `json:"libraries"`
	TestLogs           string        `json:"test_logs"`
	TestFailPrompt     string        `json:"test_fail_prompt"`
	TestCommand        string        `json:"test_command"`
	UserLanguage       string        `json:"user_language"`
}

type RoadmapContext struct {
	Language               string        `json:"language"`
	ProjectName            string        `json:"project_name"`
	Patterns               []string      `json:"patterns"`
	Architecture           string        `json:"architecture"`
	DataStrategy           string        `json:"data_strategy"`
	StateManagement        string        `json:"state_management"`
	Customization          string        `json:"customization"`
	AdditionalInstructions string        `json:"additional_instructions"`
	UserLanguage           string        `json:"user_language"`
	Libraries              []LibraryInfo `json:"libraries"`
	PRD                    string        `json:"prd"`
	Spec                   string        `json:"spec"`
	Skills                 string        `json:"skills"`
	ExistingRoadmap        string        `json:"existing_roadmap"`
}
