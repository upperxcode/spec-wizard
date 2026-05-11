import sys
import json
from http.server import BaseHTTPRequestHandler, HTTPServer

class DummyCrushHandler(BaseHTTPRequestHandler):
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
                "architectures": [{"id": "clean", "name": "Clean Architecture", "description": "Dummy Clean"}],
                "state_managements": [],
                "data_strategies": [],
                "stack_templates": []
            }).encode())
        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == '/Chat':
            content_length = int(self.headers['Content-Length'])
            post_data = self.rfile.read(content_length)
            req = json.loads(post_data)
            
            print(f"DEBUG: Received Chat request: {req}")
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            
            response = {
                "content": "Olá! Eu sou o Crush CLI Dummy. Recebi sua mensagem e estou respondendo via Plugin Expert.",
                "role": "assistant",
                "usage": {"total_tokens": 10}
            }
            self.wfile.write(json.dumps(response).encode())
        else:
            self.send_error(404)

def run(port):
    server_address = ('', port)
    httpd = HTTPServer(server_address, DummyCrushHandler)
    print(f"Starting dummy_crush on port {port}...")
    httpd.serve_forever()

if __name__ == "__main__":
    if len(sys.argv) < 2:
        print("Usage: python dummy_crush.py <port>")
        sys.exit(1)
    
    port = int(sys.argv[-1]) # O PluginManager passa o port como último argumento
    run(port)
