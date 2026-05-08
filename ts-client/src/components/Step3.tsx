import React, { useState, useEffect } from 'react'
import apiClient, { ProjectRoadmap, Sprint, Task } from '../services/apiClient'

interface Step3Props {
  projectPath: string
  language: string
  pattern: string
  onBack: () => void
}

const Step3: React.FC<Step3Props> = ({ projectPath, language, pattern, onBack }) => {
  const [roadmap, setRoadmap] = useState<ProjectRoadmap | null>(null)
  const [loading, setLoading] = useState(true)
  const [executingTask, setExecutingTask] = useState<string | null>(null)
  const [executionLogs, setExecutionLogs] = useState<Record<string, string>>({})

  useEffect(() => {
    apiClient.generateRoadmap(projectPath, language, pattern)
      .then(rm => {
        setRoadmap(rm)
        setLoading(false)
      })
      .catch(err => {
        console.error('Erro ao gerar roadmap:', err)
        setLoading(false)
      })
  }, [projectPath, language, pattern])

  const handleExecuteTask = async (sprint: Sprint, task: Task) => {
    const taskId = `sprint-${sprint.id}-task-${task.id}`
    setExecutingTask(taskId)

    try {
      const response = await apiClient.executeTask({
        project_path: projectPath,
        sprint_id: sprint.id,
        task_id: task.id,
        task,
        sprint
      })

      setExecutionLogs(prev => ({
        ...prev,
        [taskId]: response.ai_response
      }))

      alert(`✅ Tarefa executada com sucesso!\n\nResposta da IA:\n${response.ai_response.substring(0, 200)}...`)
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : 'Erro desconhecido'
      alert(`❌ Erro ao executar tarefa: ${errorMsg}`)
    } finally {
      setExecutingTask(null)
    }
  }

  if (loading) return <div className="step">Gerando roadmap...</div>

  if (!roadmap) return <div className="step">Erro ao carregar roadmap</div>

  return (
    <div className="step step-3">
      <h2>🚀 Passo 3: Roadmap e Execução de Tarefas</h2>
      <p>Linguagem: <strong>{language}</strong> | Padrão: <strong>{pattern}</strong></p>

      <div className="roadmap-container">
        {roadmap.sprints.map(sprint => (
          <div key={sprint.id} className="sprint">
            <h3>🏃 Sprint {sprint.id}: {sprint.goal}</h3>
            
            <div className="tasks">
              {sprint.tasks.map(task => {
                const taskId = `sprint-${sprint.id}-task-${task.id}`
                const hasLog = !!executionLogs[taskId]
                
                return (
                  <div key={task.id} className={`task ${hasLog ? 'completed' : ''}`}>
                    <div className="task-header">
                      <h4>{task.title}</h4>
                      <button
                        onClick={() => handleExecuteTask(sprint, task)}
                        disabled={executingTask === taskId}
                        className="btn btn-execute"
                      >
                        {executingTask === taskId ? '⏳ Executando...' : '▶️ Executar'}
                      </button>
                    </div>
                    
                    <p className="description">{task.description}</p>
                    
                    {task.acceptance_criteria && task.acceptance_criteria.length > 0 && (
                      <div className="criteria">
                        <strong>Critérios de Aceitação:</strong>
                        <ul>
                          {task.acceptance_criteria.map((crit, idx) => (
                            <li key={idx}>{crit}</li>
                          ))}
                        </ul>
                      </div>
                    )}

                    {hasLog && (
                      <details className="execution-log">
                        <summary>📋 Ver Resultado da Execução</summary>
                        <pre>{executionLogs[taskId]}</pre>
                      </details>
                    )}
                  </div>
                )
              })}
            </div>
          </div>
        ))}
      </div>

      <div className="button-group">
        <button onClick={onBack} className="btn btn-secondary">
          ← Voltar
        </button>
      </div>
    </div>
  )
}

export default Step3
