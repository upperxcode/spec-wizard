import json
import os
import subprocess
import sys
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

class CrushExpertHandler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok"}).encode())
        elif self.path == '/options':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps({
                "architectures": [],
                "state_managements": [],
                "data_strategies": [],
                "stack_templates": []
            }).encode())
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == '/rpc' or self.path == '/Chat':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            req = json.loads(post_data)

            if req.get('method') == 'Chat':
                self.handle_chat_rpc(req)
            elif 'messages' in req:
                self.handle_chat_direct(req)
            else:
                self.send_error_rpc(req.get('id'), -32601, "Method not found")
        else:
            self.send_response(404)
            self.end_headers()

    def handle_chat_rpc(self, req):
        params = req.get('params', {})
        content = self.call_crush(params.get('messages', []))
        
        response = {
            "jsonrpc": "2.0",
            "result": {
                "role": "assistant",
                "content": content
            },
            "id": req.get('id')
        }
        self.send_json(response)

    def handle_chat_direct(self, req):
        content = self.call_crush(req.get('messages', []))
        response = {
            "role": "assistant",
            "content": content
        }
        self.send_json(response)

    def call_crush(self, messages):
        full_prompt = ""
        for msg in messages:
            role = msg.get('role', 'user')
            content = msg.get('content', '')
            full_prompt += f"{role.upper()}: {content}\n"
        
        full_prompt += "\nASSISTANT: "

        print(f"📡 [CrushExpert] Calling CLI with prompt length: {len(full_prompt)}")
        sys.stdout.flush()

        try:
            # Garante que o PATH inclua o diretório de binários do usuário
            env = os.environ.copy()
            # Adiciona caminhos comuns onde o crush pode estar instalado
            paths = [
                os.path.expanduser("~/.local/bin"),
                os.path.expanduser("~/.npm-global/bin"),
                "/usr/local/bin"
            ]
            for p in paths:
                if p not in env.get("PATH", ""):
                    env["PATH"] = f"{p}:{env.get('PATH', '')}"

            # Executa a CLI passando o prompt via STDIN
            cmd = ["crush", "run"]
            result = subprocess.run(
                cmd,
                input=full_prompt,
                text=True,
                capture_output=True,
                env=env,
                timeout=540 # Aumentado para 9 minutos
            )

            if result.returncode != 0:
                print(f"❌ Crush CLI Error (code {result.returncode}): {result.stderr}")
                sys.stdout.flush()
                # Tenta retornar o stdout se o stderr estiver vazio ou for apenas aviso
                if result.stdout:
                    return result.stdout.strip()
                return f"Error calling Crush CLI: {result.stderr}"

            output = result.stdout.strip()
            
            # Filtra possíveis avisos de terminal se necessário
            lines = output.split('\n')
            filtered_lines = []
            for line in lines:
                # Ajustar filtros conforme o output real do crush
                skip = any(x in line for x in [
                    "Warning:", "DEBUG"
                ])
                if not skip:
                    filtered_lines.append(line)

            final_output = "\n".join(filtered_lines).strip()
            
            if not final_output:
                if output: return output
                return "No response from Crush CLI"

            return final_output

        except Exception as e:
            print(f"❌ Exception running Crush CLI: {str(e)}")
            sys.stdout.flush()
            return f"Exception calling Crush CLI: {str(e)}"

    def send_json(self, data):
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())

    def send_error_rpc(self, id, code, message):
        response = {
            "jsonrpc": "2.0",
            "error": {"code": code, "message": message},
            "id": id
        }
        self.send_json(response)

def run(port):
    server_address = ('', port)
    httpd = HTTPServer(server_address, CrushExpertHandler)
    print(f"Crush Expert started on port {port}")
    httpd.serve_forever()

if __name__ == '__main__':
    port = 8080
    if len(sys.argv) > 1:
        try:
            port = int(sys.argv[-1])
        except:
            pass
    
    run(port)
