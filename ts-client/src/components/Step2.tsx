import React, { useState, useEffect } from 'react'
import apiClient, { Pattern } from '../services/apiClient'

interface Step2Props {
  language: string
  onBack: () => void
  onNext: (pattern: string) => void
}

const Step2: React.FC<Step2Props> = ({ language, onBack, onNext }) => {
  const [patterns, setPatterns] = useState<Pattern[]>([])
  const [selectedPattern, setSelectedPattern] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    apiClient.getPatterns(language).then(pats => {
      setPatterns(pats)
      if (pats.length > 0) {
        setSelectedPattern(pats[0].ID)
      }
      setLoading(false)
    })
  }, [language])

  const handleNext = () => {
    if (!selectedPattern) {
      alert('Escolha um padrão')
      return
    }
    onNext(selectedPattern)
  }

  if (loading) return <div className="step">Carregando padrões...</div>

  return (
    <div className="step step-2">
      <h2>🏗️ Passo 2: Escolha o Padrão Arquitetural</h2>
      <p>Linguagem selecionada: <strong>{language}</strong></p>

      <div className="patterns-grid">
        {patterns.map(pattern => (
          <div
            key={pattern.ID}
            className={`pattern-card ${selectedPattern === pattern.ID ? 'selected' : ''}`}
            onClick={() => setSelectedPattern(pattern.ID)}
          >
            <h3>{pattern.Name}</h3>
            <p className="category">{pattern.Category}</p>
            <div className="rules">
              <strong>Regras de Ouro:</strong>
              <ul>
                {pattern.GoldenRules.map((rule, idx) => (
                  <li key={idx}>{rule}</li>
                ))}
              </ul>
            </div>
          </div>
        ))}
      </div>

      <div className="button-group">
        <button onClick={onBack} className="btn btn-secondary">
          ← Voltar
        </button>
        <button onClick={handleNext} className="btn btn-primary">
          Próximo →
        </button>
      </div>
    </div>
  )
}

export default Step2
