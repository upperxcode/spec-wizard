import json
import os
import subprocess
import sys
import time
from http.server import HTTPServer, BaseHTTPRequestHandler

class GeminiExpertHandler(BaseHTTPRequestHandler):
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
        # O Spec Wizard pode chamar via /rpc ou /Chat dependendo do adaptador
        if self.path == '/rpc' or self.path == '/Chat':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            req = json.loads(post_data)

            # Suporte a JSON-RPC e POST direto
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
        content = self.call_gemini(params.get('messages', []))
        
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
        content = self.call_gemini(req.get('messages', []))
        response = {
            "role": "assistant",
            "content": content
        }
        self.send_json(response)

    def call_gemini(self, messages):
        full_prompt = ""
        for msg in messages:
            role = msg.get('role', 'user')
            content = msg.get('content', '')
            full_prompt += f"{role.upper()}: {content}\n"
        
        full_prompt += "\nASSISTANT: "

        # Debug: Log prompt length
        print(f"📡 [GeminiExpert] Calling CLI with prompt length: {len(full_prompt)}")
        sys.stdout.flush()

        try:
            # Garante que o PATH inclua o diretório do npm global
            env = os.environ.copy()
            npm_bin = os.path.expanduser("~/.npm-global/bin")
            if npm_bin not in env.get("PATH", ""):
                env["PATH"] = f"{npm_bin}:{env.get('PATH', '')}"

            # Executa a CLI passando o prompt via STDIN
            # --approval-mode yolo para aceitar ferramentas automaticamente
            # --skip-trust para não pedir confirmação de diretório
            cmd = ["gemini", "-o", "text", "--approval-mode", "yolo", "--skip-trust"]
            result = subprocess.run(
                cmd,
                input=full_prompt,
                text=True,
                capture_output=True,
                env=env,
                timeout=540 # Aumentado para 9 minutos
            )

            if result.returncode != 0:
                print(f"❌ Gemini CLI Error (code {result.returncode}): {result.stderr}")
                sys.stdout.flush()
                return f"Error calling Gemini CLI: {result.stderr}"

            output = result.stdout.strip()
            
            # Filtra avisos de terminal
            lines = output.split('\n')
            filtered_lines = []
            for line in lines:
                skip = any(x in line for x in [
                    "Warning:", "YOLO mode", "ripgrep", "GrepTool"
                ])
                if not skip:
                    filtered_lines.append(line)

            final_output = "\n".join(filtered_lines).strip()
            
            if not final_output:
                if output: return output
                return "No response from Gemini CLI"

            return final_output

        except Exception as e:
            print(f"❌ Exception running Gemini CLI: {str(e)}")
            sys.stdout.flush()
            return f"Exception calling Gemini CLI: {str(e)}"

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
    httpd = HTTPServer(server_address, GeminiExpertHandler)
    print(f"Gemini Expert started on port {port}")
    httpd.serve_forever()

if __name__ == '__main__':
    # PluginManager passa o port como argumento
    port = 8080
    if len(sys.argv) > 1:
        try:
            port = int(sys.argv[-1])
        except:
            pass
    
    run(port)
