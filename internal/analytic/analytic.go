package analytic

import (
	"context"
	"spec-wizard/internal/logger"
	"time"
)

// Session representa uma única interação completa do usuário (Query -> Resposta Final)
type Session struct {
	ID        string
	StartTime time.Time
	Steps     []Step
}

// Step representa uma etapa individual (uma chamada de LLM ou uma execução de ferramenta)
type Step struct {
	Name     string
	Type     string // "llm", "tool", "system"
	Duration time.Duration
	Metadata map[string]any
}

func NewSession(id string) *Session {
	return &Session{
		ID:        id,
		StartTime: time.Now(),
		Steps:     []Step{},
	}
}

func (s *Session) RecordStep(name, stepType string, start time.Time, metadata map[string]any) {
	duration := time.Since(start)
	step := Step{
		Name:     name,
		Type:     stepType,
		Duration: duration,
		Metadata: metadata,
	}
	s.Steps = append(s.Steps, step)

	// Log imediato do passo para visibilidade analítica
	logger.Info("[ANALYTIC_LOG] Step Completed",
		"session_id", s.ID,
		"step", name,
		"type", stepType,
		"duration_ms", duration.Milliseconds(),
		"meta", metadata,
	)
}

func (s *Session) Finalize(finalResponse string) {
	totalDuration := time.Since(s.StartTime)

	// Resumo analítico final com snippet da resposta
	respSnippet := finalResponse
	if len(respSnippet) > 200 {
		respSnippet = respSnippet[:200] + "..."
	}

	logger.Info("[ANALYTIC_LOG] Session Finalized",
		"session_id", s.ID,
		"total_duration_ms", totalDuration.Milliseconds(),
		"steps_count", len(s.Steps),
		"resp_len", len(finalResponse),
		"response_preview", respSnippet,
	)
}

// Global Helpers
func StartLLMCall(ctx context.Context, provider, model string) time.Time {
	return time.Now()
}

func EndLLMCall(session *Session, name string, start time.Time, pTokens, cTokens int, err error) {
	meta := map[string]any{
		"p_tokens": pTokens,
		"c_tokens": cTokens,
	}
	if err != nil {
		meta["error"] = err.Error()
	}
	session.RecordStep(name, "llm", start, meta)
}

func RecordTool(session *Session, name string, start time.Time, args any, err error) {
	meta := map[string]any{
		"args": args,
	}
	if err != nil {
		meta["error"] = err.Error()
	}
	session.RecordStep(name, "tool", start, meta)
}
