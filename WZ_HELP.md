# 🧙‍♂️ Spec Wizard - Comandos Disponíveis

## 🧭 Governança & Arquitetura
  init        Inicializa a estrutura OpenSpec com um navegador de engenharia
  import      Importa um projeto existente e gera o DESIGN.md de governança
  new         Cria uma nova proposta de mudança
  roadmap     Gera um roadmap estratégico baseado no DESIGN.md
  status      Verifica o status das mudanças ativas e saúde da arquitetura
  archive     Arquiva uma tarefa concluída (move para changes/archive)

## ✅ Gestão de Tarefas
  refine      Refina uma tarefa criando sua especificação e isolamento
  code        Prepara o contexto para a IA codificar a tarefa
  fix         Prepara o contexto para a IA resolver um bugfix
  append      Anexa uma nova descrição de problema ao plano ativo (fix/feat)
  apply       Consolida as especificações para implementação pela IA
  verify      Valida a conformidade arquitetural ou as tarefas de uma branch
  set         Atualiza o status de uma tarefa

## 🌿 Controle de Versão Git
  branch      Cria uma nova branch sequencial e a pasta de planejamento correspondente
  commit      Gera o prompt de commit ou executa o commit com a mensagem fornecida
  pull        Sincroniza o repositório (pull --rebase) e sobe as mudanças (push)
  push        Envia as alterações para o repositório remoto (com auto set-upstream)

## 🤖 IA & Sistema
  mcp         Inicia o servidor MCP Profissional
  sandbox     Inicia o Sandbox de Governança para visualizar prompts da IA
  expert      Interages com os experts de linguagem
  lang        Muda o idioma da CLI de forma interativa
  agent       Instala ou força a atualização de regras de governança para assistentes de IA (Cursor, Cline, Zed, Aider)
  configure-mcp Configura o servidor MCP no arquivo de configuração do projeto
  clear-omniroute-cache Limpa o cache do OmniRoute (registros fantasmas)
  feed-sliding-window Alimenta a janela deslizante com as últimas interações
  update-macro-resume Atualiza o resumo macro das tarefas
  assemble-slim-prompt Monta um prompt slim para a IA
  slim-prompt Executa todos os comandos para montar um prompt slim

## 📚 Comandos Adicionais
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

## 🏁 Flags
  -h, --help   help for sw

Use "sw [command] --help" for more information about a command.