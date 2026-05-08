import sys
import os
import yaml # pip install pyyaml
import re
from fastapi import FastAPI
import uvicorn
from typing import List, Dict, Any
import pdfplumber
import pandas as pd
from bs4 import BeautifulSoup
import requests
from markdownify import markdownify as md
import json
import traceback

app = FastAPI()

# Definições de Opções Suportadas pelo Expert Flutter
SUPPORTED_OPTIONS = {
    "architectures": [
        {"id": "clean_architecture", "name": "Clean Architecture", "description": "Camadas separadas: Domain, Data, Presentation"},
        {"id": "mvc", "name": "MVC", "description": "Model-View-Controller tradicional"},
        {"id": "mvvm", "name": "MVVM", "description": "Model-View-ViewModel (comum com Provider/Riverpod)"},
        {"id": "bloc", "name": "BLoC Architecture", "description": "Arquitetura baseada estritamente em Streams"}
    ],
    "state_managements": [
        {"id": "bloc", "name": "BLoC/Cubit", "libs": ["flutter_bloc", "bloc"]},
        {"id": "riverpod", "name": "Riverpod", "libs": ["flutter_riverpod", "riverpod"]},
        {"id": "provider", "name": "Provider", "libs": ["provider"]},
        {"id": "getx", "name": "GetX", "libs": ["get"]},
        {"id": "mobx", "name": "MobX", "libs": ["mobx", "flutter_mobx"]},
        {"id": "signals", "name": "Signals", "libs": ["signals_flutter"]}
    ],
    "data_strategies": [
        {"id": "sql", "name": "Relacional (SQL)", "libs": ["sqflite", "drift", "floor", "supabase_flutter"]},
        {"id": "nosql", "name": "NoSQL", "libs": ["hive", "isar", "shared_preferences", "firebase_core"]},
        {"id": "remote", "name": "Apenas Remoto (API)", "libs": ["dio", "http", "chopper"]}
    ]
}

# Especializações para o campo de Customização/Sutilezas
DATA_SPECIALIZATIONS = {
    "supabase_flutter": "Supabase (Cloud SQL)",
    "firebase_core": "Firebase (Cloud NoSQL)",
    "sqflite": "SQLite (Local SQL)",
    "drift": "Drift (Local SQL)",
    "floor": "Floor (Local SQL)",
    "hive": "Hive (Local NoSQL)",
    "isar": "Isar (Local NoSQL)",
    "shared_preferences": "Shared Preferences (Local Cache)",
    "dio": "Dio HTTP Client",
    "http": "Standard HTTP Client"
}

@app.get("/health")
def health():
    return {"status": "ok"}

@app.get("/options")
def get_options():
    return SUPPORTED_OPTIONS

def get_directory_tree(path: str, max_depth: int = 4) -> str:
    tree = []
    base_path = path.rstrip(os.sep)
    start_level = base_path.count(os.sep)

    for root, dirs, files in os.walk(base_path):
        # Filtra diretórios ocultos e indesejados (Git, Github, etc) para não descer neles
        dirs[:] = [d for d in dirs if not d.startswith('.') and d not in ['node_modules', 'build', 'vendor', '.dart_tool']]
        
        level = root.count(os.sep) - start_level
        if level > max_depth: continue
        
        indent = '  ' * level
        tree.append(f"{indent}{os.path.basename(root)}/")
    return "\n".join(tree)

def infer_domain(project_path: str, pubspec: dict) -> str:
    """Infere o domínio do projeto via pubspec e README."""
    desc = pubspec.get('description', '')
    if len(desc) > 10:
        return desc
    
    readme_path = os.path.join(project_path, "README.md")
    if os.path.exists(readme_path):
        try:
            with open(readme_path, 'r', encoding='utf-8') as f:
                content = f.read(500)
                # Pega a primeira linha que não seja título ou vazia
                lines = [l.strip() for l in content.split('\n') if l.strip() and not l.startswith('#')]
                if lines: return lines[0]
        except: pass
    return "Domínio não identificado"

def infer_functional_requirements(project_path: str) -> List[str]:
    """Mapeia requisitos funcionais baseados na estrutura de pastas e métodos públicos."""
    reqs = []
    
    # 1. Pastas de Features/Screens
    features_path = os.path.join(project_path, "lib", "features")
    if os.path.exists(features_path):
        features = [d for d in os.listdir(features_path) if os.path.isdir(os.path.join(features_path, d)) and not d.startswith('.')]
        for feat in features:
            reqs.append(f"Módulo de {feat.replace('_', ' ').title()}")
    
    # 2. Análise de métodos públicos em Services/Controllers
    lib_path = os.path.join(project_path, "lib")
    method_reqs = set()
    if os.path.exists(lib_path):
        for root, dirs, files in os.walk(lib_path):
            dirs[:] = [d for d in dirs if not d.startswith('.')]
            for file in files:
                if "service" in file.lower() or "controller" in file.lower():
                    try:
                        with open(os.path.join(root, file), 'r', encoding='utf-8') as f:
                            lines = f.readlines()
                            for line in lines:
                                # Regex simples para métodos públicos Dart: tipo nome(args) {
                                match = re.search(r'^\s+(?:Future<|Stream<|[\w\d]+)?\s*([\w\d]+)\(.*\)\s*(?:async)?\s*\{', line)
                                if match:
                                    method_name = match.group(1)
                                    if not method_name.startswith('_') and method_name not in ['build', 'initState', 'dispose']:
                                        # Converte camelCase para legível
                                        readable = re.sub(r'(?<!^)(?=[A-Z])', ' ', method_name).title()
                                        method_reqs.add(f"Capacidade de {readable}")
                    except: pass
            if len(method_reqs) > 15: break
    
    reqs.extend(list(method_reqs))
    
    if not reqs:
        reqs = ["Análise de código necessária para detalhamento"]
    return reqs[:12] # Aumentado para 12


def infer_non_functional_requirements(deps: dict, project_path: str) -> List[str]:
    """Detecta requisitos não funcionais via libs e configs."""
    nfr = ["Performance", "Manutenibilidade"]
    if "flutter_secure_storage" in deps or "local_auth" in deps:
        nfr.append("Segurança de Dados")
    if "sentry" in deps or "firebase_crashlytics" in deps:
        nfr.append("Monitoramento em Tempo Real")
    if os.path.exists(os.path.join(project_path, "analysis_options.yaml")):
        nfr.append("Qualidade de Código Estrita")
    return nfr

def infer_patterns(project_path: str) -> List[str]:
    """Busca padrões de projeto no código fonte."""
    patterns = ["solid", "dry"]
    found = set()
    
    # Amostragem de arquivos para busca
    lib_path = os.path.join(project_path, "lib")
    if os.path.exists(lib_path):
        for root, dirs, files in os.walk(lib_path):
            dirs[:] = [d for d in dirs if not d.startswith('.')]
            for file in files:
                if not file.endswith('.dart'): continue
                try:
                    with open(os.path.join(root, file), 'r', encoding='utf-8') as f:
                        content = f.read()
                        if "factory" in content: found.add("factory_pattern")
                        if "static final" in content and "_instance" in content.lower(): found.add("singleton")
                        if "Repository" in content: found.add("repository_pattern")
                        if "Adapter" in content: found.add("adapter_pattern")
                except: pass
            if len(found) >= 5: break # Performance
    
    return list(found) if found else ["design_patterns_gerais"]

def infer_api_contract(project_path: str) -> str:
    """Extrai informações de contrato de API via código fonte."""
    contracts = []
    lib_path = os.path.join(project_path, "lib")
    if not os.path.exists(lib_path):
        return "Contrato não identificado"

    # Palavras-chave que indicam arquivos de rede/api
    api_keywords = ['api', 'client', 'datasource', 'repository', 'network', 'provider']

    for root, dirs, files in os.walk(lib_path):
        dirs[:] = [d for d in dirs if not d.startswith('.')]
        for file in files:
            if not file.endswith('.dart'): continue
            # Pula arquivos gerados
            if file.endswith('.g.dart') or file.endswith('.freezed.dart'): continue
            
            # Prioriza arquivos com palavras-chave de API
            is_potential_api = any(kw in file.lower() for kw in api_keywords)
            
            try:
                with open(os.path.join(root, file), 'r', encoding='utf-8') as f:
                    content = f.read()
                    
                    # 1. Busca Retrofit/Chopper Annotations (Ex: @GET("/users"))
                    matches = re.findall(r'@(?:GET|POST|PUT|DELETE|PATCH)\(["\']([^"\']+)["\']\)', content)
                    for m in matches:
                        contracts.append(f"REST: {m}")
                    
                    # 2. Busca Base URLs ou endpoints constantes
                    # Ex: static const String baseUrl = "https://api.exemplo.com"
                    base_url = re.search(r'(?:baseUrl|BASE_URL|endpoint)\s*[:=]\s*["\']([^"\']+)["\']', content, re.IGNORECASE)
                    if base_url:
                        contracts.append(f"Host/Base: {base_url.group(1)}")
                        
                    # 3. Busca chamadas diretas de Dio/Http com paths literais
                    # Ex: dio.get("/login")
                    paths = re.findall(r'\.(?:get|post|put|delete|patch|request)\(["\'](/[^"\']+)["\']', content)
                    for p in paths:
                        contracts.append(f"Path: {p}")
                    
                    # 4. Detecção de GraphQL (se houver a função gql() ou estrutura clara de query/mutation)
                    if "gql(" in content or "query {" in content.lower() or "mutation {" in content.lower():
                        contracts.append("Protocolo: GraphQL detectado")

                    # 5. Busca Supabase URL
                    supabase = re.search(r'Supabase\.initialize\(\s*url:\s*["\']([^"\']+)["\']', content)
                    if supabase:
                        contracts.append(f"Host/Base: {supabase.group(1)} (Supabase)")
            except: pass
            
            if len(contracts) > 30: break
    
    if not contracts:
        # Se não achou nada explícito, tenta inferir pelo nome dos arquivos de DataSource
        data_sources = []
        for root, dirs, files in os.walk(lib_path):
            dirs[:] = [d for d in dirs if not d.startswith('.')]
            for file in files:
                if "datasource" in file.lower() and file.endswith('.dart'):
                    name = file.replace('_data_source.dart', '').replace('_datasource.dart', '').replace('.dart', '')
                    data_sources.append(name.title())
        if data_sources:
            return f"Endpoints inferidos (DataSources): {', '.join(data_sources[:5])}"
        return "REST API (Padrão de mercado)"
    
    # Organização por categoria
    hosts = sorted(list(set([c.split(": ")[1] for c in contracts if c.startswith("Host/Base")])))
    paths = sorted(list(set([c.split(": ")[1] for c in contracts if c.startswith("Path")])))
    is_graphql = any("GraphQL" in c for c in contracts)
    
    summary = []
    if hosts: summary.append(f"Hosts: {', '.join(hosts[:3])}")
    if paths: summary.append(f"Endpoints: {', '.join(paths[:6])}")
    if is_graphql: summary.append("Protocolo: GraphQL")
    
    if not summary:
        unique_contracts = list(dict.fromkeys(contracts))[:8]
        return " | ".join(unique_contracts)
        
    return " | ".join(summary)

def detect_customization(project_path: str, pubspec: dict) -> str:
    """Analisa sutilezas como Design System."""
    hints = []
    assets = pubspec.get('flutter', {}).get('assets', [])
    fonts = pubspec.get('flutter', {}).get('fonts', [])
    
    if fonts: hints.append(f"Tipografia Customizada ({len(fonts)} fontes)")
    if any("images/" in str(a) for a in assets): hints.append("Assets visuais organizados")
    
    main_path = os.path.join(project_path, "lib/main.dart")
    if os.path.exists(main_path):
        try:
            with open(main_path, 'r') as f:
                content = f.read()
                if "ThemeData" in content:
                    if "useMaterial3: true" in content: hints.append("Material Design 3")
                    if "ColorScheme" in content: hints.append("Sistema de Cores Dinâmico")
        except: pass
        
    return " | ".join(hints) if hints else "Padrão de framework"

def full_heuristic_analysis(project_path: str) -> Dict[str, Any]:
    """Executa a perícia técnica completa solicitada pelo usuário."""
    pubspec_path = os.path.join(project_path, "pubspec.yaml")
    pubspec = {}
    if os.path.exists(pubspec_path):
        try:
            with open(pubspec_path, 'r') as f:
                pubspec = yaml.safe_load(f) or {}
        except: pass
    
    deps = pubspec.get('dependencies', {})
    description = pubspec.get('description', '')
    
    readme_content = ""
    for name in ['README.md', 'readme.md', 'README.txt']:
        readme_path = os.path.join(project_path, name)
        if os.path.exists(readme_path):
            try:
                with open(readme_path, 'r', encoding='utf-8') as f:
                    readme_content = f.read(1000) # Primeiros 1000 caracteres
                    break
            except: pass

    tree = get_directory_tree(project_path)
    
    # Heurística de Arquitetura
    arch = "custom"
    struct_lower = tree.lower()
    if "domain/" in struct_lower and "data/" in struct_lower: arch = "clean_architecture"
    elif "viewmodels/" in struct_lower or "view_models/" in struct_lower: arch = "mvvm"
    elif "blocs/" in struct_lower: arch = "bloc"
    elif "controllers/" in struct_lower: arch = "mvc"

    # Heurística de Estado
    state = "change_notifier"
    for sm in SUPPORTED_OPTIONS["state_managements"]:
        if any(lib in deps for lib in sm["libs"]):
            state = sm["id"]
            break
            
    # Heurística de Dados
    data_strat = "remote"
    active_specializations = []
    
    # Identifica especializações primeiro
    for lib, spec in DATA_SPECIALIZATIONS.items():
        if lib in deps:
            active_specializations.append(spec)
            
    # Determina a categoria principal (SQL > NoSQL > Remote)
    found_categories = []
    for ds in SUPPORTED_OPTIONS["data_strategies"]:
        if any(lib in deps for lib in ds["libs"]):
            found_categories.append(ds["id"])
    
    if found_categories:
        data_strat = found_categories[0] # Segue a ordem de prioridade definida em SUPPORTED_OPTIONS
    
    # Requisitos Não Funcionais
    nfrs = infer_non_functional_requirements(deps, project_path)
    if "sql" in found_categories and len(active_specializations) > 1:
        nfrs.append("Persistência Híbrida/Local Cache")
    
    # Detecção de Customização (Sutilezas)
    customs = detect_customization(project_path, pubspec)
    spec_text = " + ".join(list(dict.fromkeys(active_specializations))) # Remove duplicatas mantendo ordem
    
    if spec_text:
        if customs == "Padrão de framework":
            customs = f"Especialidade: {spec_text}"
        else:
            customs += f" | Especialidade: {spec_text}"

    return {
        "projectName": pubspec.get('name', os.path.basename(project_path)),
        "domain": infer_domain(project_path, pubspec),
        "functionalRequirements": infer_functional_requirements(project_path),
        "nonFunctionalRequirements": nfrs,
        "architecture": arch,
        "patterns": infer_patterns(project_path),
        "dataStrategy": data_strat,
        "stateManagement": state,
        "apiContract": infer_api_contract(project_path),
        "customization": customs,
        "semantic_context": {
            "pubspec_description": description,
            "readme_preview": readme_content
        }
    }

@app.post("/AnalyzeCodebase")
async def analyze(payload: dict):
    data = payload.get("data", {})
    project_path = data.get("project_path", "")
    
    if not project_path or not os.path.exists(project_path):
        return {"error": f"Caminho inválido: {project_path}"}
    
    # Agora retornamos a análise completa como 'inferred_properties'
    full_analysis = full_heuristic_analysis(project_path)
    
    return {
        "project_name": full_analysis["projectName"],
        "structure": get_directory_tree(project_path),
        "dependencies": {}, # Já processadas internamente
        "inferred_properties": full_analysis,
        "supported_options": SUPPORTED_OPTIONS
    }

@app.post("/GetPatterns")
async def get_patterns(payload: dict):
    return {"patterns": [
        {"id": "bloc", "name": "BLoC", "category": "State Management"},
        {"id": "repository_pattern", "name": "Repository", "category": "Data"},
        {"id": "singleton", "name": "Singleton", "category": "Pattern"}
    ]}

@app.post("/ProcessPDF")
async def process_pdf(payload: dict):
    data = payload.get("data", {})
    file_path = data.get("file_path", "")
    output_path = data.get("output_path", "")
    
    if not file_path or not os.path.exists(file_path):
        return {"error": f"Caminho do arquivo inválido: {file_path}"}
    
    try:
        content = []
        with pdfplumber.open(file_path) as pdf:
            for i, page in enumerate(pdf.pages):
                content.append(f"### Page {i+1}\n")
                text = page.extract_text()
                if text:
                    content.append(text)
                
                # Tenta extrair tabelas
                tables = page.extract_tables()
                for table in tables:
                    if table:
                        df = pd.DataFrame(table[1:], columns=table[0])
                        content.append("\n" + df.to_markdown(index=False) + "\n")
        
        full_text = "\n\n".join(content)
        
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(full_text)
            
        return {"status": "success", "output_path": output_path}
    except Exception as e:
        print(traceback.format_exc())
        return {"error": str(e)}

@app.post("/ProcessExcel")
async def process_excel(payload: dict):
    data = payload.get("data", {})
    file_path = data.get("file_path", "")
    output_path = data.get("output_path", "")
    
    if not file_path or not os.path.exists(file_path):
        return {"error": f"Caminho do arquivo inválido: {file_path}"}
    
    try:
        xl = pd.ExcelFile(file_path)
        content = []
        for sheet_name in xl.sheet_names:
            content.append(f"## Sheet: {sheet_name}\n")
            df = xl.parse(sheet_name)
            content.append(df.to_markdown(index=False))
            
        full_text = "\n\n".join(content)
        
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(full_text)
            
        return {"status": "success", "output_path": output_path}
    except Exception as e:
        return {"error": str(e)}

@app.post("/ProcessLink")
async def process_link(payload: dict):
    data = payload.get("data", {})
    url = data.get("url", "")
    output_path = data.get("output_path", "")
    
    try:
        headers = {'User-Agent': 'Mozilla/5.0'}
        response = requests.get(url, headers=headers, timeout=10)
        response.raise_for_status()
        
        soup = BeautifulSoup(response.text, 'html.parser')
        
        # Remove scripts e styles
        for script in soup(["script", "style"]):
            script.decompose()
            
        html = str(soup.find('body') or soup)
        markdown = md(html, heading_style="ATX")
        
        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, 'w', encoding='utf-8') as f:
            f.write(f"# Source: {url}\n\n" + markdown)
            
        return {"status": "success", "output_path": output_path}
    except Exception as e:
        return {"error": str(e)}

if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8081
    uvicorn.run(app, host="0.0.0.0", port=port)