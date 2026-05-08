import axios from 'axios'

const API_BASE = 'http://localhost:8080/api'

// Tipos
export interface Language {
  name: string
}

export interface Pattern {
  ID: string
  Name: string
  Category: string
  GoldenRules: string[]
}

export interface Task {
  id: number
  title: string
  description: string
  acceptance_criteria: string[]
  status?: string
}

export interface Sprint {
  id: number
  goal: string
  tasks: Task[]
}

export interface ProjectRoadmap {
  language: string
  pattern: string
  sprints: Sprint[]
}

export interface ExecutionRequest {
  project_path: string
  sprint_id: number
  task_id: number
  task: Task
  sprint: Sprint
}

export interface ExecutionResponse {
  status: string
  task_id: string
  ai_response: string
  timestamp: string
}

// API Client
export const apiClient = {
  // Obtém as linguagens disponíveis
  async getLanguages(): Promise<string[]> {
    const response = await axios.get(`${API_BASE}/languages`)
    return response.data || []
  },

  // Obtém os padrões para uma linguagem
  async getPatterns(language: string): Promise<Pattern[]> {
    const response = await axios.get(`${API_BASE}/patterns/${language}`)
    return response.data || []
  },

  // Inicializa um novo projeto
  async initializeProject(
    path: string,
    language: string,
    pattern: string
  ): Promise<{ message: string }> {
    const response = await axios.post(`${API_BASE}/initialize`, {
      path,
      language,
      pattern
    })
    return { message: response.data }
  },

  // Gera o roadmap
  async generateRoadmap(
    path: string,
    language: string,
    pattern: string
  ): Promise<ProjectRoadmap> {
    const response = await axios.post(`${API_BASE}/roadmap`, {
      path,
      language,
      pattern
    })
    return response.data
  },

  // Executa uma tarefa
  async executeTask(request: ExecutionRequest): Promise<ExecutionResponse> {
    const response = await axios.post(`${API_BASE}/execute-task`, request)
    return response.data
  }
}

export default apiClient
