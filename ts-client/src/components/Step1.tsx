import React, { useState, useEffect } from 'react'
import apiClient, { Pattern } from '../services/apiClient'

interface Step1Props {
  onNext: (path: string, language: string) => void
}

const Step1: React.FC<Step1Props> = ({ onNext }) => {
  const [projectPath, setProjectPath] = useState('')
  const [languages, setLanguages] = useState<string[]>([])
  const [selectedLanguage, setSelectedLanguage] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    apiClient.getLanguages().then(langs => {
      setLanguages(langs)
      if (langs.length > 0) {
        setSelectedLanguage(langs[0])
      }
      setLoading(false)
    })
  }, [])

  const handleNext = () => {
    if (!projectPath || !selectedLanguage) {
      alert('Preencha todos os campos')
      return
    }
    onNext(projectPath, selectedLanguage)
  }

  if (loading) return <div className="step">Carregando linguagens...</div>

  return (
    <div className="step step-1">
      <h2>📁 Passo 1: Escolha o Projeto e Linguagem</h2>
      
      <div className="form-group">
        <label>Caminho do Projeto:</label>
        <input
          type="text"
          placeholder="/home/user/meu-projeto"
          value={projectPath}
          onChange={(e) => setProjectPath(e.target.value)}
          className="input"
        />
      </div>

      <div className="form-group">
        <label>Linguagem de Programação:</label>
        <select
          value={selectedLanguage}
          onChange={(e) => setSelectedLanguage(e.target.value)}
          className="select"
        >
          {languages.map(lang => (
            <option key={lang} value={lang}>{lang}</option>
          ))}
        </select>
      </div>

      <button onClick={handleNext} className="btn btn-primary">
        Próximo →
      </button>
    </div>
  )
}

export default Step1
