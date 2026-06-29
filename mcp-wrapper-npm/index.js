#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');

// Localização padrão do binário Go compilado
const binaryPath = path.join(os.homedir(), '.local/bin/mcp-wizard');

// Repassa todos os argumentos e mantém a comunicação via STDIN/STDOUT
const child = spawn(binaryPath, process.argv.slice(2), {
    stdio: 'inherit',
    env: {
        ...process.env,
        // Garante que o binário saiba que está rodando via wrapper se necessário
        WZ_WRAPPER: 'npm'
    }
});

child.on('error', (err) => {
    console.error(`❌ Erro ao iniciar o MCP Wizard: ${err.message}`);
    console.error(`Certifique-se de que o binário existe em: ${binaryPath}`);
    process.exit(1);
});

child.on('exit', (code) => {
    process.exit(code);
});
