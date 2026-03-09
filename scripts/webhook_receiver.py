#!/usr/bin/env python3

import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer


PORT = int(os.environ.get("WEBHOOK_RECEIVER_PORT", "18081"))
LOG_PATH = os.environ.get("WEBHOOK_RECEIVER_LOG", "/tmp/crm_webhook_receiver.log")


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length).decode("utf-8")

        record = {
            "path": self.path,
            "headers": dict(self.headers.items()),
            "body": json.loads(body),
        }

        with open(LOG_PATH, "a", encoding="utf-8") as fh:
            fh.write(json.dumps(record) + "\n")

        self.send_response(200)
        self.end_headers()
        self.wfile.write(b"ok")

    def log_message(self, _format, *_args):
        return


if __name__ == "__main__":
    server = HTTPServer(("127.0.0.1", PORT), Handler)
    server.serve_forever()
