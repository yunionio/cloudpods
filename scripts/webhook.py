#!/usr/bin/env python3

from http.server import BaseHTTPRequestHandler, HTTPServer
import time


class MyServer(BaseHTTPRequestHandler):

    def do_POST(self):
        print("event", self.headers.get('X-Yunion-Event'))
        contlenstr = self.headers.get('Content-Length')
        contlen = 0
        if len(contlenstr) > 0:
            contlen = int(contlenstr)
        print(self.rfile.read(contlen))
        self.send_response(200)
        self.send_header("Content-type", "application/json")
        self.end_headers()
        self.wfile.write(bytes('{"result":"ok"}', encoding='utf-8'))


def serve(hostName, serverPort):
    webServer = HTTPServer((hostName, serverPort), MyServer)
    print("Server started http://%s:%s" % (hostName, serverPort))
    try:
        webServer.serve_forever()
    except KeyboardInterrupt:
        pass

    webServer.server_close()
    print("Server stopped.")


if __name__ == "__main__":        
    serve('0.0.0.0', 20888)
