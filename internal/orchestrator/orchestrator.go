package orchestrator

// O pacote orchestrator contém o cérebro agnóstico que coordena os Experts
// e gerencia o estado do projeto, documentação e roadmaps.
// A lógica foi fatiada em arquivos menores para melhor manutenibilidade:
// - types.go: Structs e tipos compartilhados
// - init.go: Inicialização e gerenciamento de estado
// - roadmap.go: Lógica de auditoria e plano estratégico
// - docs.go: Geração e persistência de documentos MD
// - analyzer.go: Interpretação de código e orquestração de IA
// - filetree.go: Scan de arquivos e estrutura visual
