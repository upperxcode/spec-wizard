import React, { useState, useEffect, useRef } from 'react';
import ReactMarkdown from 'react-markdown';
import { DragDropContext, Droppable, Draggable } from '@hello-pangea/dnd';
import { translations } from './translations';
import { 
  Terminal, 
  Wand2, 
  Settings2, 
  Play, 
  CheckCircle2, 
  AlertCircle, 
  ChevronRight, 
  ChevronDown,
  Code2, 
  Rocket, 
  Cpu, 
  BrainCircuit,
  Workflow,
  ListChecks,
  History,
  FileCode,
  ShieldCheck,
  Zap,
  Layout,
  Layers,
  Database,
  Sparkles,
  GitPullRequest,
  FolderOpen,
  Eye,
  RefreshCw,
  Clock,
  X,
  Copy,
  Edit3,
  Trash2,
  Save,
  Plus,
  Check,
  BookOpen
} from 'lucide-react';

import { KnowledgeBase } from './components/KnowledgeBase';

const API_BASE = 'http://localhost:8080/api';

function App() {
  const [activeTab, setActiveTab] = useState('workspace');
  const [language, setLanguage] = useState(localStorage.getItem('wizard_lang') || 'pt');
  
  const t = (key) => {
    return translations[language][key] || key;
  };

  useEffect(() => {
    localStorage.setItem('wizard_lang', language);
  }, [language]);

  const apiRequest = async (endpoint, options = {}) => {
    const url = endpoint.startsWith('http') ? endpoint : `${API_BASE}${endpoint}`;
    const headers = {
      ...options.headers,
      'X-Language': language,
    };
    if (options.body && !(options.body instanceof FormData) && !headers['Content-Type']) {
      headers['Content-Type'] = 'application/json';
    }
    return fetch(url, { ...options, headers });
  };
  const [harnessQuestion, setHarnessQuestion] = useState('');
  const [generatedPrompt, setGeneratedPrompt] = useState('');
  const [isGeneratingPrompt, setIsGeneratingPrompt] = useState(false);

  const [step, setStep] = useState(1);
  const [languages, setLanguages] = useState([]);
  const [patterns, setPatterns] = useState([]);
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState(null);
  const [roadmap, setRoadmap] = useState(null);
  const [collapsedSprints, setCollapsedSprints] = useState([]); // IDs das sprints recolhidas
  const [collapsedTasks, setCollapsedTasks] = useState([]); // IDs das tarefas recolhidas
  const [roadmapLastUpdated, setRoadmapLastUpdated] = useState("");
  const [showPromptModal, setShowPromptModal] = useState(false);
  const [promptContent, setPromptContent] = useState("");
  const [executingTask, setExecutingTask] = useState(null);
  const [taskOutputs, setTaskOutputs] = useState({}); // Mapeia taskId -> code
  const [taskDiffs, setTaskDiffs] = useState({}); // Mapeia taskId -> diff
  const [viewingCode, setViewingCode] = useState(null); // { title: string, code: string }
  const [viewingDiff, setViewingDiff] = useState(null); // { title: string, diff: string }
  const [logs, setLogs] = useState([]);
  const [llmStatus, setLlmStatus] = useState('checking'); // 'checking', 'online', 'offline'
  const [activeLlmInfo, setActiveLlmInfo] = useState({ provider: '', model: '', label: 'Nenhum' });
  const [isAiSettingsOpen, setIsAiSettingsOpen] = useState(false);
  const [llmConfig, setLlmConfig] = useState(null);
  const [interpreting, setInterpreting] = useState(false);

  const [inferringPurpose, setInferringPurpose] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editingTask, setEditingTask] = useState(null); // { sprintId, taskId }
  const [editedTaskData, setEditedTaskData] = useState(null);
  const [editingSprint, setEditingSprint] = useState(null); // sprintId
  const [editedSprintGoal, setEditedSprintGoal] = useState("");
  const [domainViewMode, setDomainViewMode] = useState('edit'); // 'edit' ou 'preview'
  const [pathStatus, setPathStatus] = useState({ initialized: false, hasCode: false, metadata: null });
  const [workspaceProjects, setWorkspaceProjects] = useState([]);
  const [auditingTaskIds, setAuditingTaskIds] = useState(new Set());
  const [viewingTaskPrompt, setViewingTaskPrompt] = useState(null); // { title: string, prompt: string }
  const abortControllerRef = useRef(null);

  const [formData, setFormData] = useState({
    name: '',
    language: '',
    architecture: '',
    philosophies: [],
    designPatterns: [],
    dataPatterns: [],
    additionalInstructions: '',
    path: '/home/data/aux/dev/projetos/go/wizard-spec/demo-app',
    projectName: '',
    domain: '',
    functionalRequirements: '',
    nonFunctionalRequirements: '',
    dataStrategy: '',
    stateManagement: '',
    apiContract: '',
    customization: ''
  });

  const calculateHealthScore = () => {
    let score = 100;
    const { philosophies, designPatterns, dataPatterns, architecture } = formData;
    
    // Filosofias não custam pontos (são guias de qualidade)
    
    // Penalidade por padrões de dados (Representação)
    // DTO, Entity, VO, POJO, Boilerplate Entity
    const representationPatterns = dataPatterns.filter(p => 
      ['dto', 'entity', 'vo', 'pojo', 'boilerplate_entity'].includes(p)
    );
    
    if (representationPatterns.length === 2) score -= 15;
    if (representationPatterns.length === 3) score -= 40;
    if (representationPatterns.length > 3) score -= 80;

    // Penalidade por excesso de Design Patterns
    if (designPatterns.length > 3) {
      score -= (designPatterns.length - 3) * 15;
    } else {
      score -= designPatterns.length * 5;
    }

    // Penalidade por contradição KISS
    if (philosophies.includes('kiss') && (designPatterns.length + dataPatterns.length) > 5) {
      score -= 25;
    }

    // Penalidade por excesso total (Apenas o que adiciona peso estrutural)
    const total = designPatterns.length + dataPatterns.length;
    
    // Penalidade por CRUD inchado
    if (architecture === 'crud' && total > 4) {
      score -= 30;
    }

    // Bônus por Clareza de Definição
    if (formData.dataStrategy) score += 5;
    if (formData.projectName) score += 2;
    if (formData.domain) score += 3;

    return Math.min(100, Math.max(0, score));
  };

  const isComplex = () => {
    const totalPatterns = formData.philosophies.length + formData.designPatterns.length + formData.dataPatterns.length;
    return totalPatterns > 4 || formData.dataPatterns.length > 3;
  };

  useEffect(() => {
    fetchLanguages();
    checkLlmStatus();
    fetchWorkspaceProjects();
  }, []);

  useEffect(() => {
    if (activeTab === 'workspace') {
      fetchWorkspaceProjects();
    }
  }, [activeTab]);

  const fetchWorkspaceProjects = async () => {
    try {
      const response = await apiRequest(`/projects`);
      const data = await response.json();
      setWorkspaceProjects(data);
    } catch (err) {
      console.error("Erro ao carregar workspace:", err);
    }
  };

  const removeProjectFromWorkspace = async (e, path) => {
    e.stopPropagation();
    if (!window.confirm(t('confirm_remove_project'))) return;

    try {
      await apiRequest(`/workspace/projects?path=${encodeURIComponent(path)}`, { method: 'DELETE' });
      fetchWorkspaceProjects();
      alert(t('project_removed'));
    } catch (err) {
      console.error("Erro ao remover projeto:", err);
      alert(t('error_loading'));
    }
  };

  const deleteProjectAnchor = async (e, path) => {
    e.stopPropagation();
    if (!window.confirm(t('confirm_delete_anchor'))) return;

    try {
      await apiRequest(`/project?path=${encodeURIComponent(path)}`, { method: 'DELETE' });
      fetchWorkspaceProjects();
      alert(t('anchor_deleted'));
    } catch (err) {
      console.error("Erro ao deletar âncora:", err);
      alert(t('error_loading'));
    }
  };

  const checkLlmStatus = async () => {
    try {
      const response = await apiRequest(`/llm/status`);
      const data = await response.json();
      setLlmStatus(data.status);
      setActiveLlmInfo({
        provider: data.provider,
        model: data.model,
        label: data.label
      });
    } catch (err) {
      setLlmStatus('offline');
      setActiveLlmInfo({ provider: '', model: '', label: 'Offline' });
    }
  };

  const onDragEnd = (result) => {
    const { source, destination, type } = result;

    if (!destination) return;

    if (type === 'SPRINT') {
      const newSprints = reorder(
        roadmap.sprints,
        source.index,
        destination.index
      );
      const nextRoadmap = { ...roadmap, sprints: newSprints };
      setRoadmap(nextRoadmap);
      saveProjectSpec(nextRoadmap);
      return;
    }

    const sourceSprintIdStr = source.droppableId.replace('tasks-', '');
    const destSprintIdStr = destination.droppableId.replace('tasks-', '');

    const sourceSprint = roadmap.sprints.find(s => s.id.toString() === sourceSprintIdStr);
    const destSprint = roadmap.sprints.find(s => s.id.toString() === destSprintIdStr);

    if (source.droppableId === destination.droppableId) {
      const newTasks = reorder(
        sourceSprint.tasks,
        source.index,
        destination.index
      );

      const newSprints = roadmap.sprints.map(s => {
        if (s.id.toString() === sourceSprintIdStr) {
          return { ...s, tasks: newTasks };
        }
        return s;
      });

      const nextRoadmap = { ...roadmap, sprints: newSprints };
      setRoadmap(nextRoadmap);
      saveProjectSpec(nextRoadmap);
    } else {
      const moveResult = move(
        sourceSprint.tasks,
        destSprint.tasks,
        source,
        destination
      );

      const newSprints = roadmap.sprints.map(s => {
        if (s.id.toString() === sourceSprintIdStr) {
          return { ...s, tasks: moveResult[source.droppableId] };
        }
        if (s.id.toString() === destSprintIdStr) {
          return { ...s, tasks: moveResult[destination.droppableId] };
        }
        return s;
      });

      const nextRoadmap = { ...roadmap, sprints: newSprints };
      setRoadmap(nextRoadmap);
      saveProjectSpec(nextRoadmap);
    }
  };

  const reorder = (list, startIndex, endIndex) => {
    const result = Array.from(list);
    const [removed] = result.splice(startIndex, 1);
    result.splice(endIndex, 0, removed);
    return result;
  };

  const move = (source, destination, droppableSource, droppableDestination) => {
    const sourceClone = Array.from(source);
    const destClone = Array.from(destination);
    const [removed] = sourceClone.splice(droppableSource.index, 1);

    destClone.splice(droppableDestination.index, 0, removed);

    const result = {};
    result[droppableSource.droppableId] = sourceClone;
    result[droppableDestination.droppableId] = destClone;

    return result;
  };


  const checkProjectStatus = async (path) => {
    if (!path) return;
    try {
      const response = await apiRequest(`/project/status?path=${encodeURIComponent(path)}`);
      const data = await response.json();
      
      setPathStatus({
        initialized: data.is_initialized,
        hasCode: data.has_code,
        metadata: data.metadata || null
      });

      if (data.current_path) {
        setFormData(prev => ({ ...prev, path: data.current_path }));
      }

      if (data.is_initialized) {
        // Se temos config completo (.spec-wizard.json), carregamos tudo
        if (data.config) {
          const cfg = data.config;
          setFormData(prev => ({
            ...prev,
            language: cfg.language || prev.language,
            projectName: cfg.projectName || prev.projectName,
            domain: cfg.domain || prev.domain,
            functionalRequirements: cfg.functionalRequirements || prev.functionalRequirements,
            nonFunctionalRequirements: cfg.nonFunctionalRequirements || prev.nonFunctionalRequirements,
            architecture: cfg.architecture || prev.architecture,
            dataStrategy: cfg.dataStrategy || prev.dataStrategy,
            stateManagement: cfg.stateManagement || prev.stateManagement,
            apiContract: cfg.apiContract || prev.apiContract,
            customization: cfg.customization || prev.customization,
            patterns: cfg.patterns || prev.patterns,
          }));
          
          // Buscar padrões e categorizar se necessário
          fetchPatterns(cfg.language || data.language, cfg.patterns || data.patterns);
        } else {
          // Fallback para config parcial legada
          setFormData(prev => ({
            ...prev,
            language: data.language || prev.language,
            projectName: data.projectName || prev.projectName
          }));
        }
        
        if (data.roadmap) {
          setRoadmap(data.roadmap);
        }
        if (data.config && data.config.roadmapLastUpdated) {
          setRoadmapLastUpdated(data.config.roadmapLastUpdated);
        } else {
          setRoadmapLastUpdated("");
        }
        
        setLogs(prev => [...prev, { 
          id: Date.now(), 
          msg: `Projeto existente detectado em ${path}. Configurações carregadas.`, 
          type: 'success' 
        }]);
      }
    } catch (err) {
      console.error("Erro ao verificar status do projeto:", err);
    }
  };

  const interpretProject = async () => {
    if (!formData.path || !formData.language) {
      const el = document.getElementById('expert-select');
      if (el) el.focus();
      setLogs(prev => [...prev, { id: Date.now(), msg: "⚠️ Selecione o Expert (Linguagem) antes de interpretar.", type: 'error' }]);
      return;
    }

    if (llmStatus !== 'online') {
      setLogs(prev => [...prev, { id: Date.now(), msg: "LM Studio está offline. Não é possível interpretar o projeto agora.", type: 'error' }]);
      return;
    }

    setInterpreting(true);
    setLogs(prev => [...prev, { id: Date.now(), msg: `Interpretando projeto em ${formData.path}...`, type: 'info' }]);

    try {
      const response = await apiRequest(`/project/interpret`, {
        method: 'POST',
        body: JSON.stringify({ path: formData.path, language: formData.language })
      });

      if (!response.ok) throw new Error("Falha na interpretação");
      
      const suggestion = await response.json();
      console.log("Sugestão recebida:", suggestion);
      
      // Converte arrays de requisitos em strings para o formulário
      const functionalReqs = Array.isArray(suggestion.functionalRequirements) 
        ? suggestion.functionalRequirements.join('\n') 
        : suggestion.functionalRequirements;
        
      const nonFunctionalReqs = Array.isArray(suggestion.nonFunctionalRequirements)
        ? suggestion.nonFunctionalRequirements.join('\n')
        : suggestion.nonFunctionalRequirements;

      // Mapeamento robusto para IDs do sistema
      const mapArchitecture = (arch) => {
        if (!arch) return '';
        const lower = arch.toLowerCase();
        if (lower.includes('clean')) return 'clean_architecture';
        if (lower.includes('mvc')) return 'mvc';
        if (lower.includes('bloc')) return 'bloc';
        if (lower.includes('mvp')) return 'mvp';
        if (lower.includes('mvi')) return 'mvi';
        if (lower.includes('viper')) return 'viper';
        if (lower.includes('cqrs')) return 'cqrs';
        if (lower.includes('event')) return 'event_sourcing';
        if (lower.includes('crud')) return 'crud';
        return 'custom';
      };

        const mapDataStrategy = (strat) => {
          if (!strat) return '';
          const lower = strat.toLowerCase();
          if (lower.includes('nosql') || lower.includes('não-relaciona')) return 'nosql';
          if (lower.includes('sql') || lower.includes('relaciona')) return 'sql';
          if (lower.includes('local') || lower.includes('offline')) return 'local';
          return 'custom';
        };

      const mapStateManagement = (sm) => {
        if (!sm) return '';
        const lower = sm.toLowerCase();
        if (lower.includes('getit') || lower.includes('locator') || lower.includes('get_it')) return 'get_it';
        if (lower.includes('bloc')) return 'bloc';
        if (lower.includes('provider') || lower.includes('change')) return 'provider';
        if (lower.includes('riverpod')) return 'riverpod';
        if (lower.includes('getx')) return 'getx';
        if (lower.includes('mobx')) return 'mobx';
        if (lower.includes('signals')) return 'signals';
        return 'custom';
      };

      const philosophies = [];
      const designPatterns = [];
      const dataPatterns = [];
      
      if (Array.isArray(suggestion.patterns)) {
        suggestion.patterns.forEach(pId => {
          const lower = pId.toLowerCase().trim();
          // Busca por ID ou por nome parcial
          const found = patterns.find(p => 
            p.id.toLowerCase() === lower || 
            p.name.toLowerCase().includes(lower) || 
            lower.includes(p.id.toLowerCase())
          );
          
          if (found) {
            if (found.category === 'Philosophy') philosophies.push(found.id);
            else if (found.category === 'DesignPattern') designPatterns.push(found.id);
            else if (found.category === 'Data') dataPatterns.push(found.id);
          }
        });
      }

      setFormData(prev => ({
        ...prev,
        projectName: suggestion.projectName || prev.projectName,
        domain: suggestion.domain || prev.domain,
        functionalRequirements: functionalReqs || prev.functionalRequirements,
        nonFunctionalRequirements: nonFunctionalReqs || prev.nonFunctionalRequirements,
        architecture: mapArchitecture(suggestion.architecture) || prev.architecture,
        dataStrategy: mapDataStrategy(suggestion.dataStrategy) || prev.dataStrategy,
        stateManagement: mapStateManagement(suggestion.stateManagement) || prev.stateManagement,
        apiContract: suggestion.apiContract || prev.apiContract,
        customization: suggestion.customization || prev.customization,
        philosophies: philosophies.length > 0 ? philosophies : prev.philosophies,
        designPatterns: designPatterns.length > 0 ? designPatterns : prev.designPatterns,
        dataPatterns: dataPatterns.length > 0 ? dataPatterns : prev.dataPatterns
      }));

      setPathStatus(prev => ({
        ...prev,
        metadata: suggestion.metadata || null
      }));

      setLogs(prev => [...prev, { 
        id: Date.now(), 
        time: new Date().toLocaleTimeString(),
        msg: "Interpretação concluída! Verifique as sugestões nos campos abaixo.", 
        type: 'success' 
      }]);
      
    } catch (err) {
      console.error("Erro na interpretação:", err);
      setLogs(prev => [...prev, { id: Date.now(), msg: `Erro na interpretação: ${err.message}`, type: 'error' }]);
    } finally {
      setInterpreting(false);
    }
  };

  const inferPurposeByAI = async () => {
    if (!formData.path) {
      setLogs(prev => [...prev, { id: Date.now(), msg: "Defina o caminho do projeto primeiro.", type: 'error' }]);
      return;
    }
    setInferringPurpose(true);
    setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: "Consultando IA para inferir propósito de negócio...", type: 'info' }]);
    
    try {
      const response = await apiRequest(`/project/infer-purpose`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          metadata: pathStatus.metadata
        })
      });

      if (!response.ok) {
        throw new Error(await response.text());
      }

      const data = await response.json();
      if (data.analysis) {
        setFormData(prev => ({
          ...prev,
          domain: data.analysis
        }));
        setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: "Propósito inferido com sucesso via IA!", type: 'success' }]);
      }
    } catch (err) {
      console.error("Erro na inferência:", err);
      setLogs(prev => [...prev, { id: Date.now(), msg: `Erro na inferência: ${err.message}`, type: 'error' }]);
    } finally {
      setInferringPurpose(false);
    }
  };

  const saveProjectSpec = async (updatedRoadmap = null) => {
    if (!formData.path) return;
    
    const roadmapToSave = updatedRoadmap || roadmap;
    
    setSaving(true);
    setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: "Salvando alterações no roadmap...", type: 'info' }]);

    try {
      const response = await apiRequest(`/project/save-spec`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          config: {
            ...formData,
            instructions: formData.additionalInstructions,
            patterns: [
              formData.architecture,
              ...formData.philosophies, 
              ...formData.designPatterns, 
              ...formData.dataPatterns
            ].filter(p => p !== '')
          },
          roadmap: roadmapToSave
        })
      });

      if (!response.ok) throw new Error(await response.text());

      setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: t('spec_saved'), type: 'success' }]);
    } catch (err) {
      console.error("Erro ao salvar:", err);
      setLogs(prev => [...prev, { id: Date.now(), msg: `Erro ao salvar: ${err.message}`, type: 'error' }]);
    } finally {
      setSaving(false);
    }
  };

  const saveRoadmapToBackend = async (newRoadmap) => {
    try {
      await apiRequest(`/project/save-roadmap`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          roadmap: newRoadmap
        })
      });
    } catch (error) {
      console.error("Erro ao salvar roadmap:", error);
      setStatus({ type: 'error', message: 'Erro ao sincronizar com o servidor.' });
    }
  };

  const handleEditTask = (sprintId, task) => {
    setEditingTask({ sprintId, taskId: task.id });
    setEditedTaskData({ 
      ...task, 
      acceptance_criteria: task.acceptance_criteria || [] 
    });
  };

  const saveEditedTask = async () => {
    if (!editingTask || !editedTaskData) return;

    const newRoadmap = { ...roadmap };
    const sprintIndex = newRoadmap.sprints.findIndex(s => s.id === editingTask.sprintId);
    if (sprintIndex === -1) return;

    const taskIndex = newRoadmap.sprints[sprintIndex].tasks.findIndex(t => t.id === editingTask.taskId);
    if (taskIndex === -1) return;

    newRoadmap.sprints[sprintIndex].tasks[taskIndex] = { ...editedTaskData };
    
    setRoadmap(newRoadmap);
    setEditingTask(null);
    setEditedTaskData(null);

    await saveRoadmapToBackend(newRoadmap);
    setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: `${t('task_updated')} (${editedTaskData.title})`, type: 'success' }]);
  };

  const handleEditSprint = (sprint) => {
    setEditingSprint(sprint.id);
    setEditedSprintGoal(sprint.goal);
  };

  const saveSprintGoal = async () => {
    if (editingSprint === null) return;
    const newRoadmap = { ...roadmap };
    const idx = newRoadmap.sprints.findIndex(s => s.id === editingSprint);
    if (idx !== -1) {
      newRoadmap.sprints[idx].goal = editedSprintGoal;
      setRoadmap(newRoadmap);
      await saveRoadmapToBackend(newRoadmap);
      setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: `${t('sprint_goal_updated')} (${editingSprint})`, type: 'success' }]);
    }
    setEditingSprint(null);
  };

  const addNewSprint = () => {
    const nextId = roadmap.sprints.length > 0 
      ? Math.max(...roadmap.sprints.map(s => parseInt(s.id) || 0)) + 1 
      : 1;
    
    const newSprint = {
      id: nextId,
      goal: `Sprint ${nextId} - ${t('sprint_default_goal')}`,
      tasks: []
    };

    const newRoadmap = {
      ...roadmap,
      sprints: [...roadmap.sprints, newSprint]
    };

    setRoadmap(newRoadmap);
    saveRoadmapToBackend(newRoadmap);
    setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: `${t('new_sprint_added')} (${nextId})`, type: 'success' }]);
  };

  const getNextTaskId = (currentRoadmap) => {
    let maxId = 0;
    currentRoadmap.sprints.forEach(s => {
      s.tasks.forEach(t => {
        const id = parseInt(t.id);
        if (id > maxId) maxId = id;
      });
    });
    return maxId + 1;
  };

  const addNewTask = (sprintId) => {
    const newRoadmap = { ...roadmap };
    const sprintIndex = newRoadmap.sprints.findIndex(s => s.id === sprintId);
    if (sprintIndex === -1) return;

    const newTaskId = getNextTaskId(newRoadmap);
    const newTask = {
      id: newTaskId,
      title: t('new_task_default_title'),
      description: t('new_task_default_desc'),
      priority: "MEDIUM",
      status: "pending",
      acceptance_criteria: [t('new_task_default_criterion')]
    };

    newRoadmap.sprints[sprintIndex].tasks.push(newTask);
    setRoadmap(newRoadmap);
    saveRoadmapToBackend(newRoadmap);
    
    // Abre o editor imediatamente para a nova tarefa
    handleEditTask(sprintId, newTask);
  };

  const toggleSprint = (sprintId) => {
    setCollapsedSprints(prev => 
      prev.includes(sprintId) ? prev.filter(id => id !== sprintId) : [...prev, sprintId]
    );
  };

  const toggleTask = (taskId) => {
    setCollapsedTasks(prev => 
      prev.includes(taskId) ? prev.filter(id => id !== taskId) : [...prev, taskId]
    );
  };

  const fetchLanguages = async () => {
    try {
      const response = await apiRequest(`/languages`);
      const data = await response.json();
      setLanguages(data);
    } catch (error) {
      console.error("Erro ao buscar linguagens:", error);
    }
  };

  const fetchPatterns = async (lang = '', initialPatterns = null) => {
    if (!lang) return;
    try {
      const response = await apiRequest(`/patterns/${lang}`);
      const data = await response.json();
      const allPatterns = data || [];
      const uniquePatterns = [];
      const seenIds = new Set();
      allPatterns.forEach(p => {
        if (!seenIds.has(p.id)) {
          seenIds.add(p.id);
          uniquePatterns.push(p);
        }
      });
      setPatterns(uniquePatterns);
      
      // Se temos padrões iniciais (do carregamento de projeto), categorizamos eles
      if (initialPatterns && allPatterns.length > 0) {
        const philo = [];
        const design = [];
        const dataP = [];
        
        initialPatterns.forEach(id => {
          const p = allPatterns.find(item => item.id === id);
          if (p) {
            if (p.category === 'Philosophy') philo.push(id);
            else if (p.category === 'DesignPattern') design.push(id);
            else if (p.category === 'Data') dataP.push(id);
          }
        });
        
        setFormData(prev => ({
          ...prev,
          philosophies: philo,
          designPatterns: design,
          dataPatterns: dataP
        }));
      }
    } catch (error) {
      console.error("Erro ao buscar padrões:", error);
    }
  };

  const handleLanguageChange = (e) => {
    const lang = e.target.value;
    setFormData({ ...formData, language: lang, architecture: '', philosophies: [], designPatterns: [], dataPatterns: [] });
    fetchPatterns(lang);
  };

  const toggleMultiSelect = (category, id) => {
    setFormData(prev => {
      const current = prev[category];
      const isSelected = current.includes(id);
      
      if (isSelected) {
        return { ...prev, [category]: current.filter(item => item !== id) };
      } else {
        const pattern = patterns.find(p => p.id === id);
        const incompatibles = pattern?.incompatibleWith || [];
        
        // Novo estado com o item adicionado
        let newState = { ...prev, [category]: [...current, id] };

        // Remover incompatíveis de todas as categorias
        ['philosophies', 'designPatterns', 'dataPatterns'].forEach(cat => {
          newState[cat] = newState[cat].filter(existingId => !incompatibles.includes(existingId));
        });

        // Se for incompatível com a arquitetura base
        if (incompatibles.includes(prev.architecture)) {
          newState.architecture = '';
        }

        return newState;
      }
    });
  };

  const getConsolidatedAdvice = () => {
    const advice = [];
    const { architecture, philosophies, designPatterns, dataPatterns } = formData;

    if (architecture === 'crud') {
      advice.push("Foco em agilidade: O sistema será centrado em operações diretas de banco.");
    } else if (architecture === 'event_sourcing') {
      advice.push("Foco em auditoria: Cada mudança será um evento imutável. Prepare-se para lidar com projeções de estado.");
    }

    if (dataPatterns.includes('dto') && dataPatterns.includes('entity')) {
      advice.push("Camadas isoladas: O uso de DTOs com Entidades sugere um mapeamento rigoroso entre a API e o Domínio.");
    }

    if (dataPatterns.includes('repository') && philosophies.includes('solid')) {
      advice.push("Alta testabilidade: Repositories com SOLID facilitam o uso de Mocking e Injeção de Dependência.");
    }

    if (dataPatterns.includes('active_record')) {
      advice.push("Modelo 'Fat': Regras de negócio e persistência estarão juntas. Cuidado com classes gigantes.");
    }

    if (dataPatterns.includes('unit_of_work')) {
      advice.push("Transações Atômicas: Útil para garantir consistência em operações que afetam múltiplos modelos.");
    }

    if (philosophies.includes('kiss') && designPatterns.length > 2) {
      advice.push("⚠️ Atenção: Você selecionou KISS mas muitos Design Patterns. Verifique se não está complicando demais.");
    }

    if (isComplex()) {
      advice.push("🔍 Arquitetura Complexa: Recomendamos detalhar como esses modelos devem interagir nas Instruções Adicionais.");
    }

    const health = calculateHealthScore();
    if (health < 40) {
      advice.push("🚨 CRÍTICO: Sua arquitetura está extremamente sobrecarregada. Isso causará lentidão no desenvolvimento e manutenção.");
    } else if (health < 70) {
      advice.push("⚠️ ALERTA: Complexidade moderada detectada. Verifique se todos os padrões são realmente necessários.");
    }

    return advice;
  };

  const handleInitialize = async () => {
    setLoading(true);
    setStatus({ type: 'info', message: 'Iniciando ancoragem do projeto...' });
    
    // Agrupar todos os padrões selecionados
    const allPatterns = [
      formData.architecture,
      ...formData.philosophies,
      ...formData.designPatterns,
      ...formData.dataPatterns
    ].filter(p => p !== '');

    try {
      const response = await apiRequest(`/initialize`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          projectName: formData.projectName,
          language: formData.language,
          patterns: allPatterns,
          instructions: formData.additionalInstructions,
          domain: formData.domain,
          functionalRequirements: formData.functionalRequirements,
          nonFunctionalRequirements: formData.nonFunctionalRequirements,
          dataStrategy: formData.dataStrategy,
          stateManagement: formData.stateManagement,
          apiContract: formData.apiContract,
          customization: formData.customization
        })
      });

      if (response.ok) {
        setStatus({ type: 'success', message: 'Projeto ancorado! Gerando roadmap...' });
        generateRoadmap(allPatterns);
      } else {
        setStatus({ type: 'error', message: 'Falha na inicialização.' });
      }
    } catch (error) {
      setStatus({ type: 'error', message: 'Erro de conexão com o servidor.' });
    } finally {
      setLoading(false);
    }
  };

  const generateRoadmap = async (allPatterns) => {
    try {
      const response = await apiRequest(`/roadmap`, {
        method: 'POST',
        body: JSON.stringify({
          ...formData,
          patterns: allPatterns
        })
      });
      const data = await response.json();
      setRoadmap(data);
      setActiveTab('roadmap');

      // SALVAMENTO AUTOMÁTICO: Gera .spec-wizard/ imediatamente após o roadmap
      const configToSave = {
        ...formData,
        patterns: allPatterns
      };
      
      // Chamada direta para o endpoint de salvamento para garantir sincronia inicial
      apiRequest(`/project/save-spec`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          config: configToSave,
          roadmap: data
        })
      }).then(() => {
        setLogs(prev => [...prev, { id: Date.now(), time: new Date().toLocaleTimeString(), msg: "Ecossistema .spec-wizard inicializado com sucesso!", type: 'success' }]);
      });

    } catch (error) {
      console.error("Erro na geração:", error);
      let errorDetail = error.message;
      try {
        const text = await error.response?.text();
        if (text) errorDetail = text;
      } catch(e) {}
      
      setStatus({ type: 'error', message: `Erro ao gerar roadmap: ${errorDetail}` });
      setLogs(prev => [...prev, { id: Date.now(), msg: `❌ Falha na geração do roadmap: ${errorDetail}`, type: 'error' }]);
    }
  };

  const generateRoadmapFromCode = async () => {
    if (!formData.path) return;
    
    // Cancela requisição anterior se existir
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    
    const controller = new AbortController();
    abortControllerRef.current = controller;
    
    setSaving(true);
    setStatus({ type: 'info', message: 'Analisando diretórios e arquitetura para evolução do roadmap... (Timeout: 180s)' });
    
    const allPatterns = [
      ...(formData.philosophies || []), 
      ...(formData.designPatterns || []), 
      ...(formData.dataPatterns || [])
    ];

    try {
      const response = await apiRequest(`/project/generate-roadmap-from-code`, {
        method: 'POST',
        signal: controller.signal,
        body: JSON.stringify({
          path: formData.path,
          config: {
            ...formData,
            patterns: allPatterns
          }
        })
      });
      
      if (!response.ok) throw new Error('Falha na resposta do servidor');
      
      const data = await response.json();
      setRoadmap(data.roadmap);
      if (data.roadmapLastUpdated) setRoadmapLastUpdated(data.roadmapLastUpdated);
      setStatus({ type: 'success', message: 'Roadmap evoluído com sucesso!' });
      
      setLogs(prev => [...prev, { 
        id: Date.now(), 
        type: 'ai', 
        message: `Roadmap evoluído via IA baseado na estrutura de ${formData.path}.` 
      }]);
      
    } catch (error) {
      if (error.name === 'AbortError') {
        setStatus({ type: 'info', message: 'Geração cancelada pelo usuário.' });
        setLogs(prev => [...prev, { id: Date.now(), type: 'info', message: 'Operação cancelada.' }]);
      } else {
        console.error(error);
        setStatus({ type: 'error', message: 'Falha ao evoluir roadmap via código.' });
      }
    } finally {
      setSaving(false);
      abortControllerRef.current = null;
    }
  };

  const cancelRequest = () => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      setSaving(false);
    }
  };

  const viewRoadmapPrompt = async () => {
    if (!formData.path) return;
    setSaving(true);
    
    const allPatterns = [
      ...(formData.philosophies || []), 
      ...(formData.designPatterns || []), 
      ...(formData.dataPatterns || [])
    ];

    try {
      const response = await apiRequest(`/project/roadmap-prompt`, {
        method: 'POST',
        body: JSON.stringify({
          path: formData.path,
          config: {
            ...formData,
            patterns: allPatterns
          }
        })
      });
      
      if (!response.ok) throw new Error('Falha ao obter prompt');
      
      const text = await response.text();
      setPromptContent(text);
      setShowPromptModal(true);
    } catch (error) {
      console.error(error);
      setStatus({ type: 'error', message: 'Falha ao visualizar prompt.' });
    } finally {
      setSaving(false);
    }
  };

  const handleViewTaskPrompt = async (sprint, task) => {
    try {
      setLoading(true);
      const response = await apiRequest('/project/generate-prompt', {
        method: 'POST',
        body: JSON.stringify({
          project_path: formData.path,
          sprint: sprint,
          task: task
        })
      });
      const data = await response.json();
      if (data.prompt) {
        setViewingTaskPrompt({ title: task.title, prompt: data.prompt });
      }
    } catch (err) {
      console.error("Erro ao carregar prompt:", err);
      alert(t('error_loading'));
    } finally {
      setLoading(false);
    }
  };

  const handleAuditTask = async (sprint, task) => {
    const startMsg = `🔍 Iniciando auditoria técnica: ${task.title}`;
    setLogs(prev => [...prev, { time: new Date().toLocaleTimeString(), msg: startMsg, type: 'info' }]);

    try {
      setAuditingTaskIds(prev => new Set([...prev, task.id]));
      const response = await apiRequest('/project/audit-task', {
        method: 'POST',
        body: JSON.stringify({
          project_path: formData.path,
          sprint_id: sprint.id,
          task_id: task.id
        })
      });
      const data = await response.json();
      
      if (data.audit) {
        const confidence = Math.round(data.audit.confidence * 100);
        const successMsg = `✅ Auditoria concluída: ${task.title} | Confiança: ${confidence}% | Status sugerido: ${data.audit.status?.toUpperCase() || 'N/A'}`;
        setLogs(prev => [...prev, { 
          time: new Date().toLocaleTimeString(), 
          msg: successMsg, 
          type: 'success' 
        }]);
        
        if (data.audit.reasoning) {
          setLogs(prev => [...prev, { 
            time: new Date().toLocaleTimeString(), 
            msg: `🧠 Raciocínio: ${data.audit.reasoning}`, 
            type: 'info' 
          }]);
        }

        // Recarregar o roadmap para refletir as mudanças de status (que agora atualizam o .md também)
        checkProjectStatus(formData.path);
      } else {
        setLogs(prev => [...prev, { 
          time: new Date().toLocaleTimeString(), 
          msg: `❌ Falha na auditoria: ${data.message || 'Sem resposta estruturada'}`, 
          type: 'error' 
        }]);
      }
    } catch (err) {
      console.error("Erro na auditoria:", err);
      setLogs(prev => [...prev, { 
        time: new Date().toLocaleTimeString(), 
        msg: `❌ Erro de conexão durante a auditoria.`, 
        type: 'error' 
      }]);
    } finally {
      setAuditingTaskIds(prev => {
        const next = new Set(prev);
        next.delete(task.id);
        return next;
      });
    }
  };

  const executeTask = async (sprint, task) => {
    setExecutingTask(task.id);
    const logMsg = `Iniciando Sprint ${sprint.id} - Tarefa ${task.id}: ${task.title}`;
    setLogs(prev => [...prev, { time: new Date().toLocaleTimeString(), msg: logMsg, type: 'info' }]);

    try {
      const response = await apiRequest(`/execute-task`, {
        method: 'POST',
        body: JSON.stringify({
          project_path: formData.path,
          sprint_id: sprint.id,
          task_id: task.id,
          task: task,
          sprint: sprint
        })
      });

      const result = await response.json();
      
      if (result.status === 'success') {
        setLogs(prev => [...prev, { 
          time: new Date().toLocaleTimeString(), 
          msg: `Tarefa concluída com sucesso em ${result.attempts} tentativa(s).`, 
          type: 'success' 
        }]);
        
        // Salvar o output e o diff da tarefa
        setTaskOutputs(prev => ({ ...prev, [task.id]: result.ai_response }));
        if (result.diff) {
          setTaskDiffs(prev => ({ ...prev, [task.id]: result.diff }));
        }
        
        // Atualizar status no roadmap local
        const updatedRoadmap = { ...roadmap };
        updatedRoadmap.sprints = updatedRoadmap.sprints.map(s => {
          if (s.id === sprint.id) {
            s.tasks = s.tasks.map(t => t.id === task.id ? { ...t, status: 'completed' } : t);
          }
          return s;
        });
        setRoadmap(updatedRoadmap);
        
        // Sincronizar com o backend para garantir que o roadmap.md na raiz esteja batendo
        checkProjectStatus(formData.path);
      } else {
        setLogs(prev => [...prev, { 
          time: new Date().toLocaleTimeString(), 
          msg: `Falha na execução: ${result.message}`, 
          type: 'error' 
        }]);
      }
    } catch (error) {
      setLogs(prev => [...prev, { 
        time: new Date().toLocaleTimeString(), 
        msg: `Erro de conexão ao executar tarefa.`, 
        type: 'error' 
      }]);
    } finally {
      setExecutingTask(null);
    }
  };

  const generateHarnessTest = async () => {
    if (!formData.path) {
      setStatus({ type: 'error', message: 'Selecione um projeto primeiro no Workspace.' });
      return;
    }
    setIsGeneratingPrompt(true);
    try {
      const response = await apiRequest(`/project/harness-prompt?path=${encodeURIComponent(formData.path)}&question=${encodeURIComponent(harnessQuestion)}`);
      const text = await response.text();
      setGeneratedPrompt(text);
      setLogs(prev => [...prev, { id: Date.now(), msg: `Harness Prompt gerado para: "${harnessQuestion.substring(0, 30)}..."`, type: 'info' }]);
    } catch (err) {
      setStatus({ type: 'error', message: 'Erro ao conectar com o Harness.' });
    } finally {
      setIsGeneratingPrompt(false);
    }
  };

  const renderPatternSection = (title, icon, category, isArchitecture = false) => {
    const filtered = patterns.filter(p => p.category === category);
    if (filtered.length === 0) return null;

    if (isArchitecture) {
      return (
        <div className="mb-6">
          <div className="flex items-center gap-2 mb-3">
            {icon}
            <h3 className="font-semibold text-slate-700">{title}</h3>
          </div>
          <div className="space-y-3">
              <select 
                className="w-full px-4 py-2 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-blue-500"
                value={formData.architecture}
                onChange={(e) => setFormData({ ...formData, architecture: e.target.value })}
              >
                <option value="">{t('select_arch_placeholder')}</option>
                <option value="custom">{t('custom_strategy')}</option>
                {filtered.map(p => {
                  const allSelected = [...formData.philosophies, ...formData.designPatterns, ...formData.dataPatterns];
                  const isConflict = allSelected.some(selId => p.incompatibleWith?.includes(selId));
                  return (
                    <option key={p.id} value={p.id} disabled={isConflict}>
                      {p.scope === 'language' ? '✨ ' : ''}{p.name} {isConflict ? t('incompatible') : ''}
                    </option>
                  );
                })}
              </select>
            {formData.architecture && (
              <p className="text-[11px] text-slate-500 italic px-2">
                {patterns.find(p => p.id === formData.architecture)?.description}
              </p>
            )}
          </div>
        </div>
      );
    }

    // Agrupar por subgrupo se existir
    const groups = {};
    filtered.forEach(p => {
      const g = p.group || t('general');
      if (!groups[g]) groups[g] = [];
      groups[g].push(p);
    });

    return (
      <div className="mb-8">
        <div className="flex items-center gap-2 mb-4 border-b border-slate-100 pb-2">
          {icon}
          <h3 className="font-bold text-slate-800 uppercase tracking-wider text-xs">{title}</h3>
        </div>
        
        {Object.keys(groups).map(groupName => (
          <div key={groupName} className="mb-4">
            {groupName !== 'Geral' && (
              <h4 className="text-[10px] font-bold text-slate-400 mb-2 px-1 uppercase">{groupName}</h4>
            )}
            <div className="grid grid-cols-2 md:grid-cols-3 gap-2">
              {groups[groupName].map(p => {
                const field = category === 'Philosophy' ? 'philosophies' : 
                             category === 'DesignPattern' ? 'designPatterns' : 'dataPatterns';
                const isSelected = formData[field].includes(p.id);
                
                // Verificar se este padrão é incompatível com QUALQUER selecionado
                const allSelected = [...formData.philosophies, ...formData.designPatterns, ...formData.dataPatterns, formData.architecture];
                const isConflict = allSelected.some(selId => p.incompatibleWith?.includes(selId));

                return (
                  <div 
                    key={p.id}
                    onClick={() => !isConflict && toggleMultiSelect(field, p.id)}
                    className={`group relative p-3 rounded-xl border transition-all duration-200 cursor-pointer ${
                      isSelected 
                        ? 'bg-blue-50 border-blue-200 shadow-sm' 
                        : isConflict 
                          ? 'bg-slate-50 border-slate-100 opacity-40 cursor-not-allowed'
                          : 'bg-white border-slate-200/60 text-slate-700 hover:border-blue-300'
                    }`}
                    title={isConflict ? t('conflict_desc') : p.description}
                  >
                    <span className="font-bold flex items-center gap-1 text-[13px]">
                      {p.name}
                      {isConflict && !isSelected && <span className="text-[8px] bg-amber-100 text-amber-600 px-1 rounded">CONFLITO</span>}
                      {p.scope === 'language' && <span className="text-[8px] bg-indigo-100 text-indigo-600 px-1.5 py-0.5 rounded border border-indigo-200 flex items-center gap-0.5"><Zap size={8} /> EXPERT</span>}
                    </span>
                    <span className={`text-[10px] line-clamp-1 ${
                      isSelected ? 'text-blue-600' : 'text-slate-400'
                    }`}>{p.description || 'Sem descrição'}</span>
                  </div>
                );
              })}
            </div>
          </div>
        ))}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-[#F8FAFC] text-slate-900 font-sans selection:bg-blue-100">
      {/* Sidebar - Sleek Glassmorphism */}
      <aside className="fixed left-0 top-0 h-full w-64 bg-white border-r border-slate-200/60 z-20 hidden lg:block">
        <div className="p-6">
          <div className="flex items-center gap-3 mb-10">
            <div className="w-10 h-10 bg-gradient-to-tr from-blue-600 to-indigo-600 rounded-2xl flex items-center justify-center shadow-lg shadow-blue-200">
              <Wand2 className="text-white w-6 h-6" />
            </div>
            <h1 className="text-xl font-bold tracking-tight bg-clip-text text-transparent bg-gradient-to-r from-slate-900 to-slate-600">
              Spec Wizard
            </h1>
          </div>

          <nav className="space-y-1">
            <button 
              onClick={() => setActiveTab('workspace')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'workspace' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              <FolderOpen size={18} /> {t('workspace')}
            </button>
            <button 
              onClick={() => setActiveTab('new')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'new' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              <Settings2 size={18} /> {t('settings')}
            </button>
            <button 
              onClick={() => setActiveTab('roadmap')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'roadmap' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              <Workflow size={18} /> {t('roadmap')}
            </button>
            <button 
              onClick={() => setActiveTab('logs')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'logs' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              <Terminal size={18} /> {t('logs')}
            </button>
            <button 
              onClick={() => setActiveTab('harness')}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'harness' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'}`}
            >
              <BrainCircuit size={18} /> {t('harness_lab')}
            </button>
            <button 
              onClick={() => setActiveTab('knowledge')}
              disabled={!formData.path}
              className={`w-full flex items-center gap-3 px-4 py-3 rounded-xl transition-all ${activeTab === 'knowledge' ? 'bg-blue-50 text-blue-600 font-semibold' : 'text-slate-500 hover:bg-slate-50'} disabled:opacity-30`}
            >
              <BookOpen size={18} /> {t('knowledge_base')}
            </button>
          </nav>
        </div>
        
        {/* Language Selector */}
        <div className="absolute bottom-0 left-0 w-full p-6 border-t border-slate-50">
          <div className="flex items-center justify-between p-1 bg-slate-100/50 rounded-2xl border border-slate-200/50 backdrop-blur-sm">
            {[
              { id: 'pt', flag: '🇧🇷', label: 'PT' },
              { id: 'en', flag: '🇺🇸', label: 'EN' },
              { id: 'es', flag: '🇪🇸', label: 'ES' }
            ].map((lang) => (
              <button 
                key={lang.id}
                onClick={() => setLanguage(lang.id)}
                className={`flex-1 flex flex-col items-center justify-center py-2 rounded-xl transition-all duration-300 ${
                  language === lang.id 
                    ? 'bg-white shadow-md text-indigo-600 scale-105 z-10' 
                    : 'text-slate-400 hover:text-slate-600 hover:bg-white/50'
                }`}
              >
                <span className="text-lg leading-none mb-1">{lang.flag}</span>
                <span className="text-[10px] font-black tracking-tighter">{lang.label}</span>
              </button>
            ))}
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="lg:pl-64 min-h-screen">
        <header className="h-16 border-b border-slate-200/60 bg-white/80 backdrop-blur-md sticky top-0 z-10 flex items-center justify-between px-8">
          <div className="flex items-center gap-2 text-slate-400 text-sm">
            <Layers size={16} />
            <span>{t('workspace')}</span>
            <ChevronRight size={14} />
            <span className="text-slate-900 font-medium">Orchestration Engine</span>
          </div>
          <div className="flex items-center gap-6">
              <button 
                onClick={async () => {
                  try {
                    const response = await apiRequest(`/llm/config`);
                    const data = await response.json();
                    setLlmConfig(data);
                    setIsAiSettingsOpen(true);
                  } catch (err) {
                    setLogs(prev => [...prev, { id: Date.now(), msg: "Erro ao carrerar configurações de IA", type: 'error' }]);
                  }
                }}
                className="flex items-center gap-2 hover:bg-slate-50 px-2 py-1 rounded-lg transition-all"
              >
                <div className={`w-2 h-2 rounded-full ${
                  llmStatus === 'online' ? 'bg-green-500 animate-pulse' : 
                  llmStatus === 'offline' ? 'bg-red-500' : 'bg-slate-300'
                }`} />
                <span className="text-[10px] font-black text-slate-400 uppercase tracking-widest">
                  {activeLlmInfo.label}: {llmStatus}
                </span>
              </button>

              <div className="h-4 w-[1px] bg-slate-200" />
              <button 
                onClick={() => setActiveTab('new')}
                className={`px-4 py-2 rounded-xl text-xs font-bold transition-all ${
                  activeTab === 'new' 
                    ? 'bg-slate-900 text-white shadow-lg shadow-slate-200' 
                    : 'text-slate-500 hover:bg-slate-100'
                }`}
              >
                {t('settings')}
              </button>
              <button 
                onClick={() => setActiveTab('dashboard')}
                disabled={!roadmap}
                className={`px-4 py-2 rounded-xl text-xs font-bold transition-all ${
                  activeTab === 'dashboard' 
                    ? 'bg-slate-900 text-white shadow-lg shadow-slate-200' 
                    : 'text-slate-500 hover:bg-slate-100'
                } disabled:opacity-30`}
              >
                {t('dashboard')}
              </button>
              <button 
                onClick={() => setActiveTab('knowledge')}
                disabled={!formData.path}
                className={`px-4 py-2 rounded-xl text-xs font-bold transition-all ${
                  activeTab === 'knowledge' 
                    ? 'bg-slate-900 text-white shadow-lg shadow-slate-200' 
                    : 'text-slate-500 hover:bg-slate-100'
                } disabled:opacity-30`}
              >
                {t('knowledge_base')}
              </button>
            </div>
        </header>

        <div className="p-4 max-w-5xl mx-auto">
          {activeTab === 'workspace' && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
              <div className="mb-8 flex items-center justify-between">
                <div>
                  <h2 className="text-3xl font-bold text-slate-900 mb-2">Seus Projetos</h2>
                  <p className="text-slate-500 text-sm">Selecione um projeto para carregar as especificações e roadmap.</p>
                </div>
                <button 
                  onClick={() => {
                    setFormData({
                      name: '',
                      language: '',
                      architecture: '',
                      philosophies: [],
                      designPatterns: [],
                      dataPatterns: [],
                      additionalInstructions: '',
                      path: '',
                      projectName: '',
                      domain: '',
                      functionalRequirements: '',
                      nonFunctionalRequirements: '',
                      dataStrategy: '',
                      stateManagement: '',
                      apiContract: '',
                      customization: ''
                    });
                    setRoadmap(null);
                    setStep(1);
                    setActiveTab('new');
                  }}
                  className="flex items-center gap-2 px-6 py-3 bg-blue-600 text-white rounded-2xl font-bold hover:bg-blue-700 transition-all shadow-lg shadow-blue-100"
                >
                  <Rocket size={18} /> Novo Projeto
                </button>
              </div>

              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                {workspaceProjects.length === 0 ? (
                  <div className="col-span-full py-20 text-center bg-white rounded-3xl border border-dashed border-slate-300">
                    <FolderOpen size={48} className="mx-auto text-slate-300 mb-4" />
                    <h3 className="text-xl font-bold text-slate-400">Nenhum projeto cadastrado</h3>
                    <p className="text-slate-400 text-sm mt-2">Adicione um novo projeto nas configurações para começar.</p>
                  </div>
                ) : (
                  workspaceProjects.map(project => (
                    <div 
                      key={project.path}
                      onClick={() => {
                        setFormData(prev => ({ ...prev, path: project.path, projectName: project.name }));
                        checkProjectStatus(project.path);
                        setStep(1);
                        setActiveTab('new');
                      }}
                      className="group bg-white p-6 rounded-3xl border border-slate-200/60 hover:border-blue-300 hover:shadow-xl hover:shadow-blue-500/5 transition-all cursor-pointer relative overflow-hidden"
                    >
                      <div className="absolute top-0 right-0 p-4 flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity z-10">
                        <button 
                          onClick={(e) => { e.stopPropagation(); removeProjectFromWorkspace(e, project.path); }}
                          title={t('remove_from_workspace')}
                          className="w-8 h-8 bg-white/80 hover:bg-amber-50 text-slate-400 hover:text-amber-500 rounded-full flex items-center justify-center shadow-sm border border-slate-100 transition-all"
                        >
                          <X size={14} />
                        </button>
                        <button 
                          onClick={(e) => { e.stopPropagation(); deleteProjectAnchor(e, project.path); }}
                          title={t('delete_project_data')}
                          className="w-8 h-8 bg-white/80 hover:bg-rose-50 text-slate-400 hover:text-rose-600 rounded-full flex items-center justify-center shadow-sm border border-slate-100 transition-all"
                        >
                          <Trash2 size={14} />
                        </button>
                         <div className="w-8 h-8 bg-blue-50 text-blue-600 rounded-full flex items-center justify-center shadow-sm">
                            <ChevronRight size={18} />
                         </div>
                      </div>
                      
                      <div className="w-12 h-12 bg-slate-50 rounded-2xl flex items-center justify-center mb-4 group-hover:bg-blue-50 transition-colors">
                        <Database size={24} className="text-slate-400 group-hover:text-blue-500" />
                      </div>
                      
                      <h3 className="text-lg font-bold text-slate-800 mb-1 group-hover:text-blue-600 transition-colors">{project.name}</h3>
                      <p className="text-[10px] text-slate-400 font-mono truncate bg-slate-50 px-2 py-1 rounded inline-block w-full">{project.path}</p>
                      
                      <div className="mt-4 flex items-center gap-2">
                        <span className="text-[9px] font-black uppercase tracking-widest text-slate-400">Status</span>
                        <div className="h-px flex-1 bg-slate-100" />
                        <span className="text-[9px] font-black text-emerald-500 uppercase tracking-widest">Ativo</span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>
          )}

          {activeTab === 'new' && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-500">
              {/* Header com Navegação de Passo */}
              <div className="mb-6 flex items-center justify-between">
                <div>
                  <h2 className="text-2xl font-bold text-slate-900 leading-tight">{t('workspace')}</h2>
                  <p className="text-[11px] font-medium text-slate-400 uppercase tracking-widest mt-1">{t('add_project_desc')}</p>
                </div>
                
                <div className="flex items-center gap-3">
                  {/* Navegação Rápida */}
                  <button 
                    onClick={() => {
                      if (step > 1) setStep(step - 1);
                      else setActiveTab('logs');
                    }}
                    className="w-10 h-10 rounded-xl bg-white border border-slate-200 flex items-center justify-center text-slate-400 hover:text-blue-600 hover:border-blue-200 transition-all shadow-sm group"
                    title={step === 1 ? t('back') : t('back')}
                  >
                    <ChevronRight className="rotate-180 group-hover:-translate-x-0.5 transition-transform" size={18} />
                  </button>

                  {/* Indicador de Passos Visual */}
                  <div className="flex items-center bg-white p-1.5 rounded-2xl border border-slate-200/60 shadow-sm">
                    <div className="flex flex-col items-center gap-1 px-4 min-w-[70px]">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-black transition-all ${step === 1 ? 'bg-blue-600 text-white shadow-lg shadow-blue-100 ring-4 ring-blue-50' : step > 1 ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-400'}`}>
                        {step > 1 ? <CheckCircle2 size={16} /> : '1'}
                      </div>
                      <span className={`text-[9px] font-black uppercase tracking-wider ${step === 1 ? 'text-blue-600' : 'text-slate-400'}`}>{t('step_1')}</span>
                    </div>

                    <div className="w-8 h-px bg-slate-100 -mt-4" />

                    <div className="flex flex-col items-center gap-1 px-4 min-w-[70px]">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-black transition-all ${step === 2 ? 'bg-blue-600 text-white shadow-lg shadow-blue-100 ring-4 ring-blue-50' : step > 2 ? 'bg-emerald-500 text-white' : 'bg-slate-100 text-slate-400'}`}>
                        {step > 2 ? <CheckCircle2 size={16} /> : '2'}
                      </div>
                      <span className={`text-[9px] font-black uppercase tracking-wider ${step === 2 ? 'text-blue-600' : 'text-slate-400'}`}>{t('step_2')}</span>
                    </div>

                    <div className="w-8 h-px bg-slate-100 -mt-4" />

                    <div className="flex flex-col items-center gap-1 px-4 min-w-[70px]">
                      <div className={`w-8 h-8 rounded-full flex items-center justify-center text-[11px] font-black transition-all ${step === 3 ? 'bg-blue-600 text-white shadow-lg shadow-blue-100 ring-4 ring-blue-50' : 'bg-slate-100 text-slate-400'}`}>
                        3
                      </div>
                      <span className={`text-[9px] font-black uppercase tracking-wider ${step === 3 ? 'text-blue-600' : 'text-slate-400'}`}>{t('step_3')}</span>
                    </div>
                  </div>

                  <button 
                    disabled={step === 3 || (step === 1 ? (!formData.language || !formData.projectName) : !formData.architecture)}
                    onClick={() => setStep(step + 1)}
                    className="w-10 h-10 rounded-xl bg-white border border-slate-200 flex items-center justify-center text-slate-400 hover:text-blue-600 hover:border-blue-200 transition-all shadow-sm group disabled:opacity-30 disabled:cursor-not-allowed"
                    title="Próximo Passo"
                  >
                    <ChevronRight className="group-hover:translate-x-0.5 transition-transform" size={18} />
                  </button>
                </div>
              </div>

              <div className="bg-white rounded-3xl border border-slate-200/60 shadow-sm p-5 space-y-6">
                {step === 1 && (
                  <div className="space-y-6 animate-in fade-in slide-in-from-right-4 duration-300">
                    {/* 1. Identificação Básica */}
                    <section>
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-1.5 bg-blue-50 text-blue-600 rounded-lg">
                          <Terminal size={18} />
                        </div>
                        <h3 className="text-md font-bold text-slate-800">{t('id_and_path')}</h3>
                      </div>
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('project_name')}</label>
                          <input 
                            type="text" 
                            placeholder="Ex: MyAwesomeApp"
                            className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all font-medium"
                            value={formData.projectName}
                            onChange={(e) => setFormData({...formData, projectName: e.target.value})}
                          />
                        </div>
                        <div>
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('project_path')}</label>
                          <div className="relative group">
                            <input 
                              type="text" 
                              className="w-full pl-4 pr-12 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all font-medium text-slate-600"
                              value={formData.path}
                              onChange={(e) => setFormData({...formData, path: e.target.value})}
                              onBlur={(e) => checkProjectStatus(e.target.value)}
                            />
                            {formData.path && !pathStatus.initialized && pathStatus.hasCode && (
                              <button
                                type="button"
                                onClick={interpretProject}
                                title={t('interpret_project')}
                                className={`absolute right-2 top-1/2 -translate-y-1/2 p-2 rounded-xl transition-all z-10 ${
                                  interpreting 
                                    ? 'text-blue-500 animate-pulse cursor-wait' 
                                    : !formData.language 
                                      ? 'text-slate-400 hover:text-indigo-600 cursor-pointer' 
                                      : 'text-indigo-600 hover:bg-indigo-50 shadow-sm cursor-pointer'
                                }`}
                              >
                                {interpreting ? (
                                  <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
                                ) : (
                                  <Sparkles size={20} />
                                )}
                              </button>
                            )}
                          </div>
                        </div>
                        <div className="md:col-span-2">
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('language_expert')}</label>
                          <select 
                            id="expert-select"
                            className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all font-bold text-slate-700"
                            value={formData.language}
                            onChange={handleLanguageChange}
                          >
                            <option value="">{t('select_expert')}</option>
                            {languages.map(lang => (
                              <option key={lang} value={lang}>{lang?.toUpperCase() || lang}</option>
                            ))}
                          </select>
                        </div>
                      </div>
                    </section>

                    <hr className="border-slate-100" />

                    {/* 2. Definição do Domínio */}
                    <section>
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-2 bg-amber-50 text-amber-600 rounded-xl">
                          <BrainCircuit size={20} />
                        </div>
                        <h3 className="text-lg font-bold text-slate-800">{t('domain_and_scope')}</h3>
                      </div>
                      <div className="space-y-6">
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest">{t('real_problem')}</label>
                            {pathStatus.hasCode && (
                              <button
                                type="button"
                                onClick={inferPurposeByAI}
                                disabled={inferringPurpose || !formData.language}
                                className={`flex items-center gap-1.5 px-2 py-1 rounded-lg text-[10px] font-bold transition-all ${
                                  inferringPurpose 
                                    ? 'bg-blue-100 text-blue-600 animate-pulse' 
                                    : 'bg-indigo-50 text-indigo-600 hover:bg-indigo-100 shadow-sm'
                                }`}
                              >
                                {inferringPurpose ? <div className="w-3 h-3 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" /> : <Wand2 size={12} />}
                                {inferringPurpose ? 'Analisando...' : 'Inferir por IA (Expert + LLM)'}
                              </button>
                            )}
                          </div>
                          <div className="flex items-center justify-between mb-2">
                            <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest">Definição do Domínio e Escopo (PRD)</label>
                            <div className="flex bg-slate-100 p-0.5 rounded-lg border border-slate-200">
                              <button 
                                onClick={() => setDomainViewMode('edit')}
                                className={`px-2 py-1 text-[10px] font-bold rounded-md transition-all ${domainViewMode === 'edit' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'}`}
                              >
                                Editar
                              </button>
                              <button 
                                onClick={() => setDomainViewMode('preview')}
                                className={`px-2 py-1 text-[10px] font-bold rounded-md transition-all ${domainViewMode === 'preview' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-500 hover:text-slate-700'}`}
                              >
                                Visualizar
                              </button>
                            </div>
                          </div>
                          
                          {domainViewMode === 'edit' ? (
                            <textarea 
                              placeholder="Qual dor o sistema resolve?"
                              className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all min-h-[250px] font-mono text-sm"
                              value={formData.domain}
                              onChange={(e) => setFormData({...formData, domain: e.target.value})}
                            />
                          ) : (
                            <div className="w-full px-6 py-5 bg-white border border-slate-200 rounded-2xl min-h-[250px] overflow-y-auto">
                              <div className="markdown-body">
                                <ReactMarkdown>
                                  {formData.domain || "_Nenhum conteúdo para visualizar._"}
                                </ReactMarkdown>
                              </div>
                            </div>
                          )}
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                          <div>
                            <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">Requisitos Funcionais</label>
                            <textarea 
                              placeholder="O que o sistema faz? (ex: cadastrar produto)"
                              className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all min-h-[100px]"
                              value={formData.functionalRequirements}
                              onChange={(e) => setFormData({...formData, functionalRequirements: e.target.value})}
                            />
                          </div>
                          <div>
                            <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">Requisitos Não-Funcionais</label>
                            <textarea 
                              placeholder="O que o sistema é? (ex: rápido, escalável)"
                              className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-2xl outline-none focus:ring-2 focus:ring-blue-500 transition-all min-h-[100px]"
                              value={formData.nonFunctionalRequirements}
                              onChange={(e) => setFormData({...formData, nonFunctionalRequirements: e.target.value})}
                            />
                          </div>
                        </div>
                      </div>
                    </section>

                    <hr className="border-slate-100" />

                    {/* 3. Persistência */}
                    <section>
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-1.5 bg-indigo-50 text-indigo-600 rounded-lg">
                          <Settings2 size={18} />
                        </div>
                        <h3 className="text-md font-bold text-slate-800">3. Persistência</h3>
                      </div>
                      <div className="grid grid-cols-1 gap-6">
                        <div>
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">Estratégia de Persistência</label>
                          <select 
                            className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 transition-all font-medium"
                            value={formData.dataStrategy}
                            onChange={(e) => setFormData({...formData, dataStrategy: e.target.value})}
                          >
                            <option value="">Selecione...</option>
                            {patterns.filter(p => p.category === 'DataStrategy').map(p => (
                              <option key={p.id} value={p.id}>
                                {p.scope === 'language' ? '✨ ' : ''}{p.name}
                              </option>
                            ))}
                            <option value="custom">Nenhum / Customizado (veja sutilezas)</option>
                          </select>
                        </div>
                      </div>
                    </section>
                  </div>
                )}

                {step === 2 && (
                  <div className="space-y-6 animate-in fade-in slide-in-from-left-4 duration-300">
                    {/* 4. Padrões e Contratos */}
                    <section>
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-1.5 bg-emerald-50 text-emerald-600 rounded-lg">
                          <Workflow size={18} />
                        </div>
                        <h3 className="text-md font-bold text-slate-800">{t('step_2')}</h3>
                      </div>
                      
                      <div className="space-y-6">
                        {/* Grade de Seleção de Patterns */}
                        <div className="space-y-1 p-4 bg-slate-50/50 rounded-2xl border border-slate-100">
                          {renderPatternSection(t('select_arch'), <Layout size={18} className="text-blue-500" />, 'Architecture', true)}
                          {renderPatternSection(t('select_philosophies'), <ShieldCheck size={18} className="text-indigo-500" />, 'Philosophy')}
                          {renderPatternSection(t('select_design_patterns'), <Zap size={18} className="text-amber-500" />, 'DesignPattern')}
                          {renderPatternSection(t('select_data_patterns'), <Database size={18} className="text-emerald-500" />, 'Data')}
                        </div>
                      </div>
                    </section>
                  </div>
                )}

                {step === 3 && (
                  <div className="space-y-6 animate-in fade-in slide-in-from-left-4 duration-300">
                    <section>
                      <div className="flex items-center gap-3 mb-4">
                        <div className="p-1.5 bg-blue-50 text-blue-600 rounded-lg">
                          <Layers size={18} />
                        </div>
                        <h3 className="text-md font-bold text-slate-800">{t('step_3')}</h3>
                      </div>

                      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div>
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('state_management')}</label>
                          <select 
                            className="w-full px-4 py-2.5 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 transition-all font-medium"
                            value={formData.stateManagement}
                            onChange={(e) => setFormData({...formData, stateManagement: e.target.value})}
                          >
                            <option value="">Selecione...</option>
                            {patterns.filter(p => p.category === 'StateManagement').map(p => (
                              <option key={p.id} value={p.id}>
                                {p.scope === 'language' ? '✨ ' : ''}{p.name}
                              </option>
                            ))}
                            <option value="custom">{t('none_custom')}</option>
                          </select>
                        </div>
                        <div>
                          <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('api_contract')}</label>
                          <textarea 
                            placeholder={t('api_placeholder')}
                            className="w-full px-4 py-2 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 transition-all min-h-[80px] text-sm"
                            value={formData.apiContract}
                            onChange={(e) => setFormData({...formData, apiContract: e.target.value})}
                          />
                        </div>
                      </div>

                      <div className="mt-6">
                        <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-2">{t('customization_label')}</label>
                        <textarea 
                          placeholder={t('customization_placeholder')}
                          className="w-full px-4 py-3 bg-slate-50 border border-slate-200 rounded-xl outline-none focus:ring-2 focus:ring-blue-500 transition-all min-h-[80px] text-sm font-mono"
                          value={formData.customization}
                          onChange={(e) => setFormData({...formData, customization: e.target.value})}
                        />
                      </div>
                    </section>

                    <hr className="border-slate-100" />

                    {/* 6. Instruções Adicionais e Advisor */}
                    <section className="space-y-6">
                      <div className="flex items-center gap-3 mb-2">
                        <div className="p-1.5 bg-slate-50 text-slate-600 rounded-lg">
                          <ListChecks size={18} />
                        </div>
                        <h3 className="text-md font-bold text-slate-800">{t('final_adjustments')}</h3>
                      </div>

                      <div className={`p-4 rounded-2xl border transition-all ${isComplex() ? 'bg-amber-50/30 border-amber-200' : 'bg-slate-50/30 border-slate-100'}`}>
                        <div className="flex items-center gap-2 mb-2">
                          <FileCode className={`w-3.5 h-3.5 ${isComplex() ? 'text-amber-600' : 'text-slate-500'}`} />
                          <h4 className={`font-bold text-xs ${isComplex() ? 'text-amber-900' : 'text-slate-700'}`}>
                            {t('implementation_instructions')} {isComplex() && <span className="text-[9px] bg-amber-200 text-amber-800 px-2 py-0.5 rounded-full ml-2 uppercase font-black tracking-tighter">{t('high_complexity')}</span>}
                          </h4>
                        </div>
                        <textarea 
                          className="w-full bg-white border border-slate-200 rounded-xl p-3 text-xs outline-none focus:ring-2 focus:ring-blue-500 min-h-[80px] placeholder:text-slate-400"
                          placeholder={t('implementation_placeholder')}
                          value={formData.additionalInstructions}
                          onChange={(e) => setFormData({...formData, additionalInstructions: e.target.value})}
                        />
                      </div>

                      {/* Pattern Advisor Panel */}
                      {getConsolidatedAdvice().length > 0 && (
                        <div className="p-4 bg-blue-50/50 rounded-2xl border border-blue-100/50 animate-in zoom-in-95 duration-300">
                          <h4 className="flex items-center gap-2 text-blue-900 font-bold mb-3 text-xs">
                            <Sparkles size={14} className="text-blue-600" /> {t('consolidated_advice')}
                          </h4>
                          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                            {getConsolidatedAdvice().map((tip, i) => (
                              <div key={i} className="flex gap-3 items-start bg-white/80 p-3 rounded-xl border border-blue-100 shadow-sm">
                                <div className="w-1 h-1 rounded-full bg-blue-500 mt-1.5 flex-shrink-0" />
                                <p className="text-[11px] font-medium text-blue-800 leading-relaxed">{tip}</p>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </section>
                  </div>
                )}

                {/* Footer de Ação */}
                <div className="flex items-center justify-between pt-8 border-t border-slate-100">
                  <div className="flex flex-col">
                    <span className="text-[10px] font-black text-slate-400 uppercase tracking-widest mb-1">{t('architecture_health')}</span>
                    <div className="flex items-center gap-3">
                      <div className="w-32 h-2 bg-slate-100 rounded-full overflow-hidden border border-slate-200/50">
                        <div 
                          className={`h-full transition-all duration-1000 ${
                            calculateHealthScore() > 70 ? 'bg-emerald-500' : 
                            calculateHealthScore() > 40 ? 'bg-amber-500' : 'bg-rose-500'
                          }`}
                          style={{ width: `${calculateHealthScore()}%` }}
                        />
                      </div>
                      <span className={`text-xs font-black ${
                        calculateHealthScore() > 70 ? 'text-emerald-600' : 
                        calculateHealthScore() > 40 ? 'text-amber-600' : 'text-rose-600'
                      }`}>
                        {calculateHealthScore()}%
                      </span>
                    </div>
                  </div>

                  <div className="flex gap-4">
                    {step < 3 ? (
                      <button 
                        onClick={() => setStep(step + 1)}
                        disabled={step === 1 ? (!formData.language || !formData.projectName) : !formData.architecture}
                        className="flex items-center gap-2 px-10 py-4 bg-slate-900 text-white rounded-2xl font-bold hover:bg-blue-600 hover:scale-105 disabled:opacity-30 disabled:hover:scale-100 transition-all shadow-xl shadow-slate-200 group"
                      >
                        {t('next_step')}
                        <ChevronRight size={18} className="group-hover:translate-x-1 transition-transform" />
                      </button>
                    ) : (
                      <button 
                        onClick={handleInitialize}
                        disabled={loading || calculateHealthScore() < 30}
                        className="flex items-center gap-2 px-10 py-4 bg-blue-600 text-white rounded-2xl font-bold hover:bg-blue-700 hover:scale-105 disabled:opacity-30 disabled:hover:scale-100 shadow-xl shadow-blue-200 transition-all"
                      >
                        {loading ? <Cpu className="animate-spin" size={18} /> : <Rocket size={18} />}
                        {calculateHealthScore() < 30 ? t('architecture_blocked') : t('launch_mission')}
                      </button>
                    )}
                  </div>
                </div>
              </div>

              {status && (
                <div className={`mt-6 p-4 rounded-2xl flex items-center gap-3 animate-in slide-in-from-top-2 duration-300 ${status.type === 'error' ? 'bg-rose-50 text-rose-600 border border-rose-100' : 'bg-blue-50 text-blue-600 border border-blue-100'}`}>
                  {status.type === 'error' ? <AlertCircle size={20} /> : <Sparkles size={20} className="animate-pulse" />}
                  <span className="font-medium text-sm">{status.message}</span>
                </div>
              )}
            </div>
          )}

          {activeTab === 'knowledge' && (
            <div className="animate-in fade-in slide-in-from-bottom-4 duration-500 bg-white p-8 rounded-[40px] border border-slate-200/60 shadow-sm">
              <KnowledgeBase 
                t={t} 
                apiRequest={apiRequest} 
                projectPath={formData.path} 
              />
            </div>
          )}

          {activeTab === 'roadmap' && roadmap && (
            <div className="animate-in fade-in slide-in-from-right-4 duration-500">
              <div className="flex items-center justify-between mb-8">
                <div>
                  <h2 className="text-3xl font-bold text-slate-900 mb-2">{t('roadmap_title')}</h2>
                  <div className="flex items-center gap-4">
                    <span className="px-3 py-1 bg-blue-50 text-blue-600 rounded-lg text-xs font-bold flex items-center gap-1">
                      <Code2 size={14} /> {(roadmap?.language?.toUpperCase() || 'N/A')}
                    </span>
                    <span className="px-3 py-1 bg-indigo-50 text-indigo-600 rounded-lg text-xs font-bold flex items-center gap-1">
                      <Layout size={14} /> {roadmap?.pattern || 'N/A'}
                    </span>
                    {roadmapLastUpdated && (
                      <span className="text-slate-400 text-[10px] font-medium flex items-center gap-1">
                        <History size={10} /> {t('last_updated')}: {roadmapLastUpdated}
                      </span>
                    )}
                    {!roadmapLastUpdated && (
                      <span className="text-amber-500 text-[10px] font-bold flex items-center gap-1 animate-pulse">
                        <AlertCircle size={10} /> {t('roadmap_outdated')}
                      </span>
                    )}
                  </div>
                </div>

                <div className="flex gap-3">
                  <button 
                    onClick={viewRoadmapPrompt}
                    className="flex items-center gap-2 px-4 py-2.5 rounded-2xl font-bold text-xs transition-all bg-slate-50 text-slate-500 hover:bg-slate-100 border border-slate-200"
                  >
                    <Eye size={14} /> {t('view_prompt')}
                  </button>

                  <button 
                    onClick={saving ? cancelRequest : generateRoadmapFromCode}
                    className={`flex items-center gap-2 px-6 py-2.5 rounded-2xl font-bold text-sm transition-all shadow-lg ${
                      saving 
                        ? 'bg-rose-50 text-rose-600 border border-rose-100 hover:bg-rose-100 shadow-rose-100/20' 
                        : 'bg-indigo-50 text-indigo-600 hover:bg-indigo-100 border border-indigo-100 shadow-indigo-100/20'
                    }`}
                  >
                    {saving ? <X size={16} /> : <Zap size={16} />}
                    {saving ? t('cancel_generation') : t('evolve_roadmap')}
                  </button>

                  <button 
                    onClick={saveProjectSpec}
                    disabled={saving || !roadmap}
                    className={`flex items-center gap-2 px-6 py-2.5 rounded-2xl font-bold text-sm transition-all shadow-lg ${
                      saving 
                        ? 'bg-slate-100 text-slate-400 cursor-not-allowed' 
                        : 'bg-emerald-50 text-emerald-600 hover:bg-emerald-100 border border-emerald-100 shadow-emerald-100/20'
                    }`}
                  >
                    {saving ? <Cpu className="animate-spin" size={16} /> : <ShieldCheck size={16} />}
                    {saving ? t('saving') : t('save_to_project')}
                  </button>
                </div>
              </div>

              <DragDropContext onDragEnd={onDragEnd}>
                <Droppable droppableId="sprints-list" type="SPRINT" direction="vertical">
                  {(provided) => (
                    <div 
                      {...provided.droppableProps}
                      ref={provided.innerRef}
                      className="space-y-12"
                    >
                {roadmap?.sprints?.map((sprint, index) => {
                  const isSprintCollapsed = collapsedSprints.includes(sprint.id);
                  return (
                    <Draggable key={sprint.id} draggableId={`sprint-${sprint.id}`} index={index}>
                      {(provided, snapshot) => (
                        <div 
                          ref={provided.innerRef}
                          {...provided.draggableProps}
                          className={`animate-in fade-in slide-in-from-bottom-4 duration-500 ${snapshot.isDragging ? 'z-[60] scale-[1.02] shadow-2xl' : ''}`}
                        >
                          <div 
                            className="flex items-center gap-4 mb-6 cursor-pointer group/sprint"
                            onClick={() => toggleSprint(sprint.id)}
                            {...provided.dragHandleProps}
                          >
                            <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold text-lg shadow-lg transition-all ${
                              isSprintCollapsed ? 'bg-slate-200 text-slate-500' : 'bg-indigo-600 text-white shadow-indigo-200'
                            }`}>
                              {isSprintCollapsed ? <ChevronRight size={20} /> : sprint.id}
                            </div>
                            
                            {editingSprint === sprint.id ? (
                              <div className="flex-1 flex gap-2" onClick={e => e.stopPropagation()}>
                                <input 
                                  autoFocus
                                  value={editedSprintGoal}
                                  onChange={e => setEditedSprintGoal(e.target.value)}
                                  onKeyDown={e => e.key === 'Enter' && saveSprintGoal()}
                                  className="flex-1 bg-white border-2 border-indigo-200 rounded-xl px-4 py-2 text-xl font-extrabold text-slate-800 outline-none"
                                />
                                <button onClick={saveSprintGoal} className="p-2 bg-indigo-600 text-white rounded-xl shadow-lg"><Check size={20} /></button>
                              </div>
                            ) : (
                              <div className="flex-1 flex items-center gap-3">
                                <h3 className="text-xl font-extrabold text-slate-800 uppercase tracking-tight">{sprint.goal}</h3>
                                <button 
                                  onClick={(e) => { e.stopPropagation(); handleEditSprint(sprint); }}
                                  className="opacity-0 group-hover/sprint:opacity-100 p-2 text-slate-400 hover:text-indigo-600 transition-all"
                                >
                                  <Edit3 size={16} />
                                </button>
                              </div>
                            )}
    
                            <div className="h-px bg-slate-200 flex-1 ml-4 opacity-50"></div>
                            <button className="p-2 text-slate-300 group-hover/sprint:text-indigo-500 transition-colors">
                              {isSprintCollapsed ? <ChevronRight size={20} /> : <ChevronDown size={20} />}
                            </button>
                          </div>
    
                          {!isSprintCollapsed && (
                            <Droppable droppableId={`tasks-${sprint.id}`} type="TASK">
                              {(provided, snapshot) => (
                                <div 
                                  {...provided.droppableProps}
                                  ref={provided.innerRef}
                                  className={`grid gap-5 min-h-[50px] transition-all rounded-3xl p-2 ${snapshot.isDraggingOver ? 'bg-indigo-50/50 ring-2 ring-indigo-200/50 ring-dashed' : ''}`}
                                >
                                  {sprint?.tasks?.map((task, taskIndex) => {
                                    const isTaskCollapsed = collapsedTasks.includes(task.id);
                                    return (
                                      <Draggable key={task.id} draggableId={`task-${task.id}`} index={taskIndex}>
                                        {(provided, snapshot) => (
                                          <div 
                                            ref={provided.innerRef}
                                            {...provided.draggableProps}
                                            {...provided.dragHandleProps}
                                            className={`bg-white rounded-3xl border transition-all relative group ${
                                              isTaskCollapsed ? 'p-4 hover:shadow-lg' : 'p-6 hover:shadow-xl hover:shadow-indigo-100/30'
                                            } ${
                                              task.status === 'completed' 
                                                ? 'border-slate-100 opacity-75' 
                                                : 'border-slate-200/60 hover:border-indigo-200'
                                            } ${snapshot.isDragging ? 'shadow-2xl border-indigo-500 ring-4 ring-indigo-500/10 z-[70] scale-[1.01]' : ''}`}
                                          >
                                            <div className="flex flex-col md:flex-row gap-5">
                                              <div 
                                                onClick={() => toggleTask(task.id)}
                                                className={`w-12 h-12 rounded-2xl flex items-center justify-center shrink-0 cursor-pointer ${
                                                  task.status === 'completed' ? 'bg-green-50 text-green-600' : 'bg-slate-50 text-slate-400 group-hover:bg-indigo-50 group-hover:text-indigo-600 transition-colors'
                                                }`}
                                              >
                                                {isTaskCollapsed ? (
                                                  <ChevronRight size={20} />
                                                ) : (
                                                  task.status === 'completed' ? <CheckCircle2 size={24} /> : <FileCode size={24} />
                                                )}
                                              </div>
            
                                              <div className="flex-1">
                                                <div 
                                                  className="flex items-start justify-between gap-4 mb-2 cursor-pointer"
                                                  onClick={() => toggleTask(task.id)}
                                                >
                                                  <h4 className={`text-lg font-bold leading-tight ${
                                                    task.status === 'completed' ? 'text-slate-400 line-through' : 'text-slate-800'
                                                  } ${isTaskCollapsed ? 'truncate max-w-xl' : ''}`}>
                                                    {task.title}
                                                  </h4>
                                                  <div className="flex items-center gap-2 shrink-0">
                                                    {task.priority && (
                                                      <span className={`text-[10px] px-2.5 py-1 rounded-lg font-black uppercase tracking-wider ${
                                                        task.priority === 'HIGH' ? 'bg-rose-100 text-rose-600' :
                                                        task.priority === 'MEDIUM' ? 'bg-amber-100 text-amber-600' :
                                                        'bg-indigo-50 text-indigo-500'
                                                      }`}>
                                                        {t(task.priority.toLowerCase() + '_priority')}
                                                      </span>
                                                    )}
                                                    <button 
                                                      onClick={(e) => { e.stopPropagation(); handleEditTask(sprint.id, task); }}
                                                      className="p-2 text-slate-400 hover:text-indigo-600 hover:bg-indigo-50 rounded-xl transition-all opacity-0 group-hover:opacity-100"
                                                    >
                                                      <Edit3 size={16} />
                                                    </button>
                                                  </div>
                                                </div>
                                                
                                                {!isTaskCollapsed && (
                                                  <div className="animate-in fade-in slide-in-from-top-2 duration-300">
                                                    <p className={`text-sm leading-relaxed mb-4 ${task.status === 'completed' ? 'text-slate-300' : 'text-slate-500'}`}>
                                                      {task.description}
                                                    </p>
            
                                                    {task.acceptance_criteria && task.acceptance_criteria.length > 0 && (
                                                      <div className="space-y-2 pt-4 border-t border-slate-50">
                                                        <span className="text-[10px] font-bold text-slate-400 uppercase tracking-widest block mb-2">{t('acceptance_criteria')}</span>
                                                        <div className="grid grid-cols-1 md:grid-cols-2 gap-x-6 gap-y-1.5">
                                                          {task.acceptance_criteria.map((c, idx) => (
                                                            <div key={idx} className="flex items-center gap-2 text-xs text-slate-500">
                                                              <div className="w-1.5 h-1.5 rounded-full bg-slate-200"></div>
                                                              {c}
                                                            </div>
                                                          ))}
                                                        </div>
                                                      </div>
                                                    )}
            
                                                    <div className="flex items-center gap-3 mt-6">
                                                      {task.status !== 'completed' && (
                                                          <button 
                                                            onClick={() => executeTask(sprint, task)}
                                                            disabled={executingTask !== null}
                                                            className="flex items-center gap-2 px-6 py-2.5 bg-blue-600 text-white rounded-2xl text-xs font-bold hover:bg-blue-700 disabled:bg-slate-200 shadow-lg shadow-blue-100 transition-all"
                                                          >
                                                            {executingTask === task.id ? t('executing') : <><Play size={14} /> {t('execute')}</>}
                                                          </button>
                                                        )}
              
                                                        <button 
                                                          onClick={() => handleViewTaskPrompt(sprint, task)}
                                                          className="flex items-center gap-2 px-4 py-2.5 bg-slate-100 text-slate-600 rounded-2xl text-xs font-bold hover:bg-slate-200 transition-all"
                                                        >
                                                          <Eye size={14} /> {t('view_task_prompt')}
                                                        </button>

                                                        <button 
                                                          onClick={() => handleAuditTask(sprint, task)}
                                                          disabled={auditingTaskIds.has(task.id)}
                                                          className="flex items-center gap-2 px-4 py-2.5 bg-indigo-50 text-indigo-600 border border-indigo-100 rounded-2xl text-xs font-bold hover:bg-indigo-100 transition-all"
                                                        >
                                                          {auditingTaskIds.has(task.id) ? (
                                                            <><RefreshCw size={14} className="animate-spin" /> {t('auditing')}</>
                                                          ) : (
                                                            <><RefreshCw size={14} /> {t('audit_status')}</>
                                                          )}
                                                        </button>
            
                                                      {task.status === 'completed' && (
                                                        <>
                                                          {taskOutputs[task.id] && (
                                                            <button 
                                                              onClick={() => setViewingCode({ title: task.title, code: taskOutputs[task.id] })}
                                                              className="flex items-center gap-2 px-4 py-2 bg-slate-100 text-slate-600 rounded-xl text-xs font-bold hover:bg-slate-200 transition-all"
                                                            >
                                                              <Code2 size={14} /> {t('ver_code')}
                                                            </button>
                                                          )}
                                                          {taskDiffs[task.id] && (
                                                            <button 
                                                              onClick={() => setViewingDiff({ title: task.title, diff: taskDiffs[task.id] })}
                                                              className="flex items-center gap-2 px-4 py-2 bg-indigo-50 text-indigo-600 rounded-xl text-xs font-bold hover:bg-indigo-100 transition-all border border-indigo-100"
                                                            >
                                                              <GitPullRequest size={14} /> {t('ver_diff')}
                                                            </button>
                                                          )}
                                                        </>
                                                      )}
                                                    </div>
                                                  </div>
                                                )}
                                              </div>
                                            </div>
                                          </div>
                                        )}
                                      </Draggable>
                                    );
                                  })}
                                  {provided.placeholder}
    
                                  {/* ADD TASK BUTTON */}
                                  <button 
                                    onClick={() => addNewTask(sprint.id)}
                                    className="mt-2 group/add py-4 border-2 border-dashed border-slate-200 rounded-3xl flex items-center justify-center gap-3 text-slate-400 font-bold hover:border-indigo-300 hover:text-indigo-600 transition-all bg-slate-50/30 hover:bg-indigo-50/30"
                                  >
                                    <div className="w-8 h-8 rounded-xl bg-white border border-slate-200 flex items-center justify-center group-hover/add:border-indigo-200 group-hover/add:text-indigo-600 shadow-sm transition-all">
                                      <Plus size={18} />
                                    </div>
                                    <span>{t('add_task')}</span>
                                  </button>
                                </div>
                              )}
                            </Droppable>
                          )}
                        </div>
                      )}
                    </Draggable>
                  );
                })}
                {provided.placeholder}

                {/* ADD SPRINT BUTTON */}
                <button 
                  onClick={addNewSprint}
                  className="w-full py-8 border-4 border-dashed border-slate-100 rounded-[40px] flex flex-col items-center justify-center gap-4 text-slate-300 hover:border-indigo-100 hover:text-indigo-400 hover:bg-indigo-50/10 transition-all group/sprint-add"
                >
                  <div className="w-16 h-16 rounded-[24px] bg-white border-2 border-slate-100 flex items-center justify-center group-hover/sprint-add:border-indigo-100 group-hover/sprint-add:text-indigo-400 shadow-xl shadow-slate-100/50 transition-all">
                    <Plus size={32} strokeWidth={3} />
                  </div>
                  <div className="text-center">
                    <p className="text-xl font-black uppercase tracking-widest">{t('new_sprint')}</p>
                    <p className="text-sm font-medium opacity-60">{t('new_sprint_desc')}</p>
                  </div>
                </button>
              </div>
            )}
          </Droppable>
        </DragDropContext>

              {/* TASK EDITOR MODAL */}
              {editingTask && editedTaskData && (
                <div className="fixed inset-0 bg-slate-900/60 backdrop-blur-sm z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300">
                  <div className="bg-white rounded-[40px] shadow-2xl w-full max-w-2xl overflow-hidden animate-in zoom-in-95 duration-300 border border-white/20">
                    <div className="bg-slate-50/50 px-10 py-8 border-b border-slate-100 flex items-center justify-between">
                      <div>
                        <h3 className="text-2xl font-black text-slate-800 tracking-tight">{t('edit_task')}</h3>
                        <p className="text-slate-400 text-sm font-medium">Sprint {editingTask.sprintId} • ID {editingTask.taskId}</p>
                      </div>
                      <button 
                        onClick={() => setEditingTask(null)}
                        className="w-12 h-12 flex items-center justify-center rounded-2xl text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all"
                      >
                        <X size={24} />
                      </button>
                    </div>

                    <div className="p-10 space-y-8 overflow-y-auto max-h-[60vh]">
                      <div className="space-y-2">
                        <label className="text-xs font-black text-slate-400 uppercase tracking-widest ml-1">{t('task_title')}</label>
                        <input 
                          value={editedTaskData.title}
                          onChange={(e) => setEditedTaskData({...editedTaskData, title: e.target.value})}
                          className="w-full bg-slate-50 border-none rounded-2xl px-6 py-4 font-bold text-slate-700 focus:ring-2 focus:ring-indigo-500/20 transition-all"
                        />
                      </div>

                      <div className="grid grid-cols-2 gap-6">
                        <div className="space-y-2">
                          <label className="text-xs font-black text-slate-400 uppercase tracking-widest ml-1">{t('priority')}</label>
                          <select 
                            value={editedTaskData.priority}
                            onChange={(e) => setEditedTaskData({...editedTaskData, priority: e.target.value})}
                            className="w-full bg-slate-50 border-none rounded-2xl px-6 py-4 font-bold text-slate-700 focus:ring-2 focus:ring-indigo-500/20 transition-all appearance-none"
                          >
                            <option value="HIGH">{t('high_priority')}</option>
                            <option value="MEDIUM">{t('medium_priority')}</option>
                            <option value="LOW">{t('low_priority')}</option>
                          </select>
                        </div>
                        <div className="space-y-2">
                          <label className="text-xs font-black text-slate-400 uppercase tracking-widest ml-1">{t('status')}</label>
                          <select 
                            value={editedTaskData.status}
                            onChange={(e) => setEditedTaskData({...editedTaskData, status: e.target.value})}
                            className="w-full bg-slate-50 border-none rounded-2xl px-6 py-4 font-bold text-slate-700 focus:ring-2 focus:ring-indigo-500/20 transition-all appearance-none"
                          >
                            <option value="pending">{t('status_pending')}</option>
                            <option value="in_progress">{t('status_in_progress')}</option>
                            <option value="completed">{t('status_completed')}</option>
                            <option value="failed">{t('status_failed')}</option>
                          </select>
                        </div>
                      </div>

                      <div className="space-y-2">
                        <label className="text-xs font-black text-slate-400 uppercase tracking-widest ml-1">{t('strategic_description')}</label>
                        <textarea 
                          rows={4}
                          value={editedTaskData.description}
                          onChange={(e) => setEditedTaskData({...editedTaskData, description: e.target.value})}
                          className="w-full bg-slate-50 border-none rounded-2xl px-6 py-4 font-medium text-slate-600 focus:ring-2 focus:ring-indigo-500/20 transition-all resize-none"
                        />
                      </div>

                      <div className="space-y-4">
                        <label className="text-xs font-black text-slate-400 uppercase tracking-widest ml-1">{t('acceptance_criteria')}</label>
                        {editedTaskData.acceptance_criteria.map((c, idx) => (
                          <div key={idx} className="flex gap-3">
                            <input 
                              value={c}
                              onChange={(e) => {
                                const newCriteria = [...editedTaskData.acceptance_criteria];
                                newCriteria[idx] = e.target.value;
                                setEditedTaskData({...editedTaskData, acceptance_criteria: newCriteria});
                              }}
                              className="flex-1 bg-slate-50 border-none rounded-xl px-4 py-3 text-sm text-slate-600 focus:ring-2 focus:ring-indigo-500/20 transition-all"
                            />
                            <button 
                              onClick={() => {
                                const newCriteria = editedTaskData.acceptance_criteria.filter((_, i) => i !== idx);
                                setEditedTaskData({...editedTaskData, acceptance_criteria: newCriteria});
                              }}
                              className="p-3 text-rose-400 hover:bg-rose-50 rounded-xl transition-all"
                            >
                              <Trash2 size={18} />
                            </button>
                          </div>
                        ))}
                        <button 
                          onClick={() => setEditedTaskData({...editedTaskData, acceptance_criteria: [...editedTaskData.acceptance_criteria, '']})}
                          className="w-full py-3 border-2 border-dashed border-slate-200 rounded-xl text-slate-400 font-bold text-xs hover:border-indigo-300 hover:text-indigo-500 transition-all"
                        >
                          + {t('add_criterion')}
                        </button>
                      </div>
                    </div>

                    <div className="p-10 bg-slate-50/50 border-t border-slate-100 flex gap-4">
                      <button 
                        onClick={() => setEditingTask(null)}
                        className="flex-1 px-8 py-4 rounded-2xl font-bold text-slate-400 hover:bg-slate-200/50 transition-all"
                      >
                        {t('discard')}
                      </button>
                      <button 
                        onClick={saveEditedTask}
                        className="flex-1 px-8 py-4 bg-indigo-600 text-white rounded-2xl font-bold shadow-xl shadow-indigo-200 hover:bg-indigo-700 transition-all flex items-center justify-center gap-2"
                      >
                        <Save size={18} /> {t('save_changes')}
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* TASK PROMPT VIEWER MODAL */}
              {viewingTaskPrompt && (
                <div className="fixed inset-0 bg-slate-900/80 backdrop-blur-md z-[110] flex items-center justify-center p-8 animate-in fade-in duration-300">
                  <div className="bg-white rounded-[40px] shadow-2xl w-full max-w-4xl overflow-hidden flex flex-col max-h-[90vh] animate-in zoom-in-95 duration-300">
                    <div className="px-10 py-8 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
                      <div>
                        <h3 className="text-2xl font-black text-slate-800 tracking-tight">{t('view_task_prompt')}</h3>
                        <p className="text-slate-400 text-sm font-medium">{viewingTaskPrompt.title}</p>
                      </div>
                      <div className="flex items-center gap-4">
                        <button 
                          onClick={() => {
                            navigator.clipboard.writeText(viewingTaskPrompt.prompt);
                            alert(t('prompt_copied'));
                          }}
                          className="flex items-center gap-2 px-6 py-3 bg-indigo-600 text-white rounded-2xl text-sm font-bold shadow-xl shadow-indigo-200 hover:bg-indigo-700 transition-all"
                        >
                          <Copy size={18} /> {t('copy_prompt')}
                        </button>
                        <button 
                          onClick={() => setViewingTaskPrompt(null)}
                          className="w-12 h-12 flex items-center justify-center rounded-2xl text-slate-400 hover:bg-slate-100 hover:text-slate-600 transition-all"
                        >
                          <X size={24} />
                        </button>
                      </div>
                    </div>
                    <div className="flex-1 overflow-y-auto p-10 bg-slate-900">
                      <pre className="text-blue-200 font-mono text-sm leading-relaxed whitespace-pre-wrap">
                        {viewingTaskPrompt.prompt}
                      </pre>
                    </div>
                    <div className="px-10 py-6 border-t border-slate-100 bg-slate-50/50 text-right">
                      <button 
                        onClick={() => setViewingTaskPrompt(null)}
                        className="px-8 py-3 bg-white border border-slate-200 text-slate-600 rounded-xl font-bold hover:bg-slate-50 transition-all"
                      >
                        {t('close')}
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {activeTab === 'logs' && (
            <div className="animate-in fade-in zoom-in-95 duration-500">
              <div className="mb-6 flex items-center justify-between">
                <h2 className="text-3xl font-bold text-slate-900">{t('logs_title')}</h2>
                <div className="flex items-center gap-2">
                  <span className="text-xs text-slate-400">{t('real_time')}</span>
                  <div className="w-2 h-2 bg-blue-500 rounded-full animate-ping" />
                </div>
              </div>
              <div className="bg-slate-900 rounded-3xl p-6 shadow-2xl shadow-slate-200 min-h-[500px] font-mono text-sm overflow-hidden flex flex-col">
                <div className="flex items-center gap-2 mb-4 border-b border-slate-800 pb-4">
                  <div className="w-3 h-3 rounded-full bg-red-500" />
                  <div className="w-3 h-3 rounded-full bg-yellow-500" />
                  <div className="w-3 h-3 rounded-full bg-green-500" />
                  <span className="ml-2 text-slate-500 text-xs">spec-wizard-v3.log</span>
                </div>
                <div className="flex-1 overflow-y-auto space-y-3 custom-scrollbar pr-2">
                  {logs.length === 0 ? (
                    <div className="text-slate-700 italic">{t('waiting_execution')}</div>
                  ) : (
                    logs.map((log, i) => (
                      <div key={i} className="flex gap-4 border-l border-slate-800 pl-4 py-1">
                        <span className="text-slate-600 shrink-0">[{log.time}]</span>
                        <span className={
                          log.type === 'error' ? 'text-red-400' : 
                          log.type === 'success' ? 'text-green-400' : 'text-blue-300'
                        }>{log.msg}</span>
                      </div>
                    ))
                  )}
                </div>
              </div>
            </div>
          )}

          {activeTab === 'harness' && (
            <div className="animate-in fade-in zoom-in-95 duration-500">
              <div className="mb-8">
                <h2 className="text-3xl font-bold text-slate-900 leading-tight">{t('harness_lab')}</h2>
                <p className="text-sm font-medium text-slate-400 uppercase tracking-widest mt-1">{t('harness_desc')}</p>
              </div>

              <div className="grid grid-cols-1 gap-8">
                <div className="bg-white rounded-[2rem] border border-slate-200/60 shadow-sm p-8">
                  <div className="flex items-center gap-3 mb-6">
                    <div className="p-2 bg-blue-50 text-blue-600 rounded-xl">
                      <Zap size={20} />
                    </div>
                    <h3 className="text-lg font-bold text-slate-800">{t('define_scenario')}</h3>
                  </div>
                  
                  <label className="block text-[10px] font-black text-slate-400 uppercase tracking-widest mb-3">{t('harness_question_label')}</label>
                  <textarea 
                    value={harnessQuestion}
                    onChange={(e) => setHarnessQuestion(e.target.value)}
                    placeholder={t('harness_placeholder')}
                    className="w-full bg-slate-50 border border-slate-100 rounded-[1.5rem] p-5 text-sm font-medium focus:ring-4 focus:ring-blue-500/10 focus:border-blue-500 transition-all outline-none min-h-[120px]"
                  />
                  
                  <div className="mt-6 flex justify-end">
                    <button 
                      onClick={generateHarnessTest}
                      disabled={isGeneratingPrompt || !harnessQuestion}
                      className="flex items-center gap-3 px-10 py-4 bg-blue-600 text-white rounded-2xl font-bold hover:bg-blue-700 hover:scale-[1.02] disabled:opacity-30 transition-all shadow-xl shadow-blue-100"
                    >
                      {isGeneratingPrompt ? <Cpu className="animate-spin" size={20} /> : <BrainCircuit size={20} />}
                      {t('build_prompt')}
                    </button>
                  </div>
                </div>

                {generatedPrompt && (
                  <div className="bg-slate-950 rounded-[2rem] border border-slate-800 shadow-2xl p-8 overflow-hidden animate-in zoom-in-95 duration-300">
                    <div className="flex items-center justify-between mb-6 border-b border-white/5 pb-6">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-slate-900 text-blue-400 rounded-xl border border-white/5">
                          <Terminal size={20} />
                        </div>
                        <div>
                          <span className="text-[10px] font-black text-slate-500 uppercase tracking-widest block">{t('harness_output')}</span>
                          <span className="text-sm font-bold text-white">{t('consolidated_prompt')}</span>
                        </div>
                      </div>
                      <button 
                        onClick={() => {
                          navigator.clipboard.writeText(generatedPrompt);
                          setStatus({ type: 'success', message: t('prompt_copied') });
                        }}
                        className="px-6 py-2.5 bg-white/5 hover:bg-white/10 text-blue-400 text-[10px] font-black uppercase tracking-widest rounded-xl transition-all flex items-center gap-2 border border-white/5"
                      >
                        {t('copy_to_chat')}
                      </button>
                    </div>
                    <div className="max-h-[600px] overflow-y-auto custom-scrollbar pr-4">
                      <pre className="text-[13px] font-mono text-slate-300 whitespace-pre-wrap leading-relaxed">
                        {generatedPrompt}
                      </pre>
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Modal de Visualização de Código */}
        {viewingCode && (
          <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in duration-200">
            <div className="bg-white w-full max-w-4xl max-h-[80vh] rounded-3xl shadow-2xl flex flex-col overflow-hidden border border-slate-200">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-blue-600 text-white rounded-xl">
                    <Code2 size={20} />
                  </div>
                  <div>
                    <h3 className="font-bold text-slate-900">{viewingCode.title}</h3>
                    <p className="text-[10px] text-slate-500 uppercase font-black tracking-widest">Código Gerado e Validado</p>
                  </div>
                </div>
                <button 
                  onClick={() => setViewingCode(null)}
                  className="w-10 h-10 rounded-full hover:bg-slate-200 flex items-center justify-center transition-colors text-slate-400"
                >
                  <AlertCircle size={20} className="rotate-45" />
                </button>
              </div>
              <div className="flex-1 overflow-auto p-6 bg-slate-900">
                <pre className="text-blue-300 font-mono text-sm leading-relaxed">
                  {viewingCode.code}
                </pre>
              </div>
              <div className="p-4 border-t border-slate-100 flex justify-end bg-white">
                <button 
                  onClick={() => setViewingCode(null)}
                  className="px-6 py-2 bg-slate-900 text-white rounded-xl font-bold text-sm hover:bg-slate-800 transition-all"
                >
                  Fechar
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Modal de Visualização de Diff */}
        {viewingDiff && (
          <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in duration-200">
            <div className="bg-white w-full max-w-4xl max-h-[80vh] rounded-3xl shadow-2xl flex flex-col overflow-hidden border border-slate-200">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-slate-50/50">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-indigo-600 text-white rounded-xl">
                    <GitPullRequest size={20} />
                  </div>
                  <div>
                    <h3 className="font-bold text-slate-900">Diff: {viewingDiff.title}</h3>
                    <p className="text-[10px] text-slate-500 uppercase font-black tracking-widest">Alterações Realizadas (Git Diff)</p>
                  </div>
                </div>
                <button 
                  onClick={() => setViewingDiff(null)}
                  className="w-10 h-10 rounded-full hover:bg-slate-200 flex items-center justify-center transition-colors text-slate-400"
                >
                  <AlertCircle size={20} className="rotate-45" />
                </button>
              </div>
              <div className="flex-1 overflow-auto p-0 bg-slate-900 font-mono text-xs">
                <div className="min-w-full inline-block">
                  {viewingDiff.diff.split('\n').map((line, i) => {
                    let bgColor = 'transparent';
                    let textColor = 'text-slate-400';
                    if (line.startsWith('+')) {
                      bgColor = 'bg-emerald-900/30';
                      textColor = 'text-emerald-400';
                    } else if (line.startsWith('-')) {
                      bgColor = 'bg-rose-900/30';
                      textColor = 'text-rose-400';
                    } else if (line.startsWith('@@')) {
                      bgColor = 'bg-blue-900/20';
                      textColor = 'text-blue-400';
                    }
                    return (
                      <div key={i} className={`${bgColor} px-6 py-0.5 whitespace-pre`}>
                        <span className={`select-none mr-4 opacity-30 inline-block w-8 text-right`}>{i + 1}</span>
                        <span className={textColor}>{line}</span>
                      </div>
                    );
                  })}
                </div>
              </div>
              <div className="p-4 border-t border-slate-100 flex justify-end bg-white">
                <button 
                  onClick={() => setViewingDiff(null)}
                  className="px-6 py-2 bg-slate-900 text-white rounded-xl font-bold text-sm hover:bg-slate-800 transition-all"
                >
                  Fechar
                </button>
              </div>
            </div>
          </div>
        )}
        {isAiSettingsOpen && llmConfig && (
          <div className="fixed inset-0 bg-slate-900/60 backdrop-blur-sm z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300">
            <div className="bg-white rounded-3xl w-full max-w-2xl shadow-2xl overflow-hidden border border-slate-100 flex flex-col max-h-[80vh]">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-bold text-slate-800 flex items-center gap-2">
                    <Cpu size={20} className="text-blue-600" /> {t('ai_orchestration')}
                    <button 
                      onClick={async () => {
                        try {
                          const response = await apiRequest(`/llm/reload`, { method: 'POST' });
                          const data = await response.json();
                          setLlmConfig(data);
                          setLogs(prev => [...prev, { id: Date.now(), msg: t('sync_success'), type: 'success' }]);
                        } catch (err) {
                          setLogs(prev => [...prev, { id: Date.now(), msg: t('sync_error'), type: 'error' }]);
                        }
                      }}
                      className="p-1.5 hover:bg-slate-100 rounded-lg text-slate-400 transition-all ml-2"
                      title={t('sync_disk')}
                    >
                      <RefreshCw size={14} />
                    </button>

                  </h2>
                  <p className="text-xs text-slate-500">{t('brain_select')}</p>
                </div>

                <button 
                  onClick={() => setIsAiSettingsOpen(false)}
                  className="p-2 hover:bg-slate-100 rounded-xl transition-all text-slate-400"
                >
                  <X size={20} />
                </button>
              </div>

              <div className="p-6 overflow-auto custom-scrollbar flex-1 space-y-6">
                {llmConfig.providers.map(provider => (
                  <div key={provider.name} className={`p-4 rounded-2xl border transition-all ${provider.enabled ? 'border-slate-200 bg-white' : 'border-slate-100 bg-slate-50 opacity-60'}`}>
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className={`w-8 h-8 rounded-lg flex items-center justify-center ${provider.name === 'ollama' ? 'bg-orange-50 text-orange-600' : 'bg-blue-50 text-blue-600'}`}>
                           <span className="font-black text-[10px] uppercase">{provider.name.substring(0, 2)}</span>
                        </div>
                        <div>
                          <h3 className="font-bold text-slate-800 capitalize">{provider.name}</h3>
                          <p className="text-[10px] text-slate-400 font-medium">{provider.api_url}</p>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={async () => {
                            try {
                              await apiRequest(`/llm/provider/status`, {
                                method: 'POST',
                                body: JSON.stringify({ name: provider.name, enabled: !provider.enabled })
                              });
                              // Atualiza localmente
                              setLlmConfig(prev => ({
                                ...prev,
                                providers: prev.providers.map(p => 
                                  p.name === provider.name ? { ...p, enabled: !p.enabled } : p
                                )
                              }));
                              checkLlmStatus();
                            } catch (err) {
                              setLogs(prev => [...prev, { id: Date.now(), msg: t('update_provider_error'), type: 'error' }]);
                            }
                          }}
                          className={`px-3 py-1 rounded-full text-[9px] font-black uppercase tracking-tighter transition-all ${
                            provider.enabled 
                              ? 'bg-emerald-100 text-emerald-700 hover:bg-emerald-200' 
                              : 'bg-slate-200 text-slate-500 hover:bg-slate-300'
                          }`}
                        >
                          {provider.enabled ? t('active') : t('inactive')}
                        </button>
                      </div>

                    </div>

                    <div className="grid grid-cols-1 gap-2">
                      {provider.models.map(model => {
                        const isActive = llmConfig.active.provider === provider.name && llmConfig.active.model === model.name;
                        return (
                          <button
                            key={model.name}
                            disabled={!provider.enabled || !model.enabled}
                            onClick={async () => {
                              try {
                                await apiRequest(`/llm/active`, {
                                  method: 'POST',
                                  body: JSON.stringify({ provider: provider.name, model: model.name })
                                });
                                // Atualiza localmente para feedback imediato
                                setLlmConfig(prev => ({
                                  ...prev,
                                  active: { provider: provider.name, model: model.name }
                                }));
                                checkLlmStatus();
                              } catch (err) {
                                setLogs(prev => [...prev, { id: Date.now(), msg: t('change_model_error'), type: 'error' }]);
                              }
                            }}
                            className={`flex items-center justify-between p-3 rounded-xl border transition-all ${
                              isActive 
                                ? 'border-blue-500 bg-blue-50/50 ring-2 ring-blue-500/20' 
                                : 'border-slate-100 hover:border-blue-200 bg-white'
                            } disabled:opacity-50 disabled:cursor-not-allowed`}
                          >
                            <div className="flex items-center gap-3">
                              <div className={`w-2 h-2 rounded-full ${isActive ? 'bg-blue-500 animate-pulse' : 'bg-slate-200'}`} />
                              <div className="text-left">
                                <p className={`text-xs font-bold ${isActive ? 'text-blue-700' : 'text-slate-700'}`}>{model.label}</p>
                                <p className="text-[10px] text-slate-400 font-mono">{model.name}</p>
                              </div>
                            </div>
                            {isActive && <CheckCircle2 size={16} className="text-blue-500" />}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>

              <div className="p-6 border-t border-slate-100 bg-slate-50/50 flex justify-end">
                <button 
                  onClick={() => setIsAiSettingsOpen(false)}
                  className="px-8 py-2.5 bg-slate-900 text-white rounded-xl font-bold text-sm hover:bg-slate-800 transition-all shadow-lg shadow-slate-200"
                >
                  {t('done')}
                </button>
              </div>
            </div>
          </div>
        )}
        {showPromptModal && (

          <div className="fixed inset-0 bg-slate-900/60 backdrop-blur-sm z-[100] flex items-center justify-center p-8 animate-in fade-in duration-300">
            <div className="bg-white rounded-3xl w-full max-w-5xl max-h-[85vh] flex flex-col shadow-2xl overflow-hidden border border-slate-100">
              <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-white">
                <div>
                  <h2 className="text-xl font-bold text-slate-800">{t('strategic_prompt')}</h2>
                  <p className="text-xs text-slate-500">{t('exact_content_desc')}</p>
                </div>
                <button 
                  onClick={() => setShowPromptModal(false)}
                  className="p-2 hover:bg-slate-100 rounded-xl transition-all text-slate-400"
                >
                  <X size={20} />
                </button>
              </div>
              <div className="flex-1 overflow-auto bg-slate-50 p-6 custom-scrollbar">
                <div className="bg-[#1e293b] rounded-2xl p-6 font-mono text-sm leading-relaxed text-indigo-100 border border-slate-800 shadow-inner">
                  {promptContent.split('\n').map((line, i) => (
                    <div key={i} className="min-h-[1.5rem]">
                      <span className="select-none mr-4 opacity-30 inline-block w-8 text-right">{i + 1}</span>
                      {line}
                    </div>
                  ))}
                </div>
              </div>
              <div className="p-6 border-t border-slate-100 flex justify-end gap-3 bg-white">
                <button 
                  onClick={() => {
                    navigator.clipboard.writeText(promptContent);
                    setStatus({ type: 'success', message: t('prompt_copied_msg') });
                  }}
                  className="px-6 py-2.5 bg-indigo-50 text-indigo-600 rounded-xl font-bold text-sm hover:bg-indigo-100 transition-all flex items-center gap-2"
                >
                  <Copy size={16} /> {t('copy_prompt')}
                </button>
                <button 
                  onClick={() => setShowPromptModal(false)}
                  className="px-6 py-2.5 bg-slate-900 text-white rounded-xl font-bold text-sm hover:bg-slate-800 transition-all"
                >
                  {t('close')}
                </button>
              </div>
            </div>
          </div>
        )}
      </main>

      {/* Modern styles for Scrollbar */}
      <style dangerouslySetInnerHTML={{ __html: `
        .custom-scrollbar::-webkit-scrollbar { width: 4px; }
        .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
        .custom-scrollbar::-webkit-scrollbar-thumb { background: #334155; border-radius: 10px; }
      `}} />
    </div>
  );
}

export default App;