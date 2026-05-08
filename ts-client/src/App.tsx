import React, { useState } from 'react'
import Step1 from './components/Step1'
import Step2 from './components/Step2'
import Step3 from './components/Step3'
import './App.css'

type CurrentStep = 1 | 2 | 3

const App: React.FC = () => {
  const [currentStep, setCurrentStep] = useState<CurrentStep>(1)
  const [projectPath, setProjectPath] = useState('')
  const [language, setLanguage] = useState('')
  const [pattern, setPattern] = useState('')

  const handleStep1Next = (path: string, lang: string) => {
    setProjectPath(path)
    setLanguage(lang)
    setCurrentStep(2)
  }

  const handleStep2Next = (pat: string) => {
    setPattern(pat)
    setCurrentStep(3)
  }

  const handleBack = () => {
    setCurrentStep((prev) => (prev === 1 ? 1 : (prev - 1) as CurrentStep))
  }

  return (
    <div className="app">
      <header className="header">
        <h1>🧙‍♂️ Spec Wizard - Dashboard de Orquestração</h1>
        <p>Desenvolvimento orientado a especificações com IA local</p>
      </header>

      <div className="stepper">
        <div className={`step-indicator ${currentStep >= 1 ? 'active' : ''}`}>
          <span className="number">1</span>
          <span className="label">Projeto & Linguagem</span>
        </div>
        <div className="connector"></div>
        <div className={`step-indicator ${currentStep >= 2 ? 'active' : ''}`}>
          <span className="number">2</span>
          <span className="label">Padrão Arquitetural</span>
        </div>
        <div className="connector"></div>
        <div className={`step-indicator ${currentStep >= 3 ? 'active' : ''}`}>
          <span className="number">3</span>
          <span className="label">Roadmap & Execução</span>
        </div>
      </div>

      <main className="main-content">
        {currentStep === 1 && (
          <Step1 onNext={handleStep1Next} />
        )}
        {currentStep === 2 && (
          <Step2 language={language} onBack={handleBack} onNext={handleStep2Next} />
        )}
        {currentStep === 3 && (
          <Step3 projectPath={projectPath} language={language} pattern={pattern} onBack={handleBack} />
        )}
      </main>

      <footer className="footer">
        <p>Spec Wizard © 2026 | Orquestração Agnóstica de Desenvolvimento</p>
      </footer>
    </div>
  )
}

export default App
