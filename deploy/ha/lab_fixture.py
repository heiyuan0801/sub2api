"""Local synthetic model provider, never sends data to external services."""

import hashlib
import json
import os
import pathlib
import threading
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = pathlib.Path(os.environ["SUB2API_HA_LAB_DIR"])
PRICING = (
    pathlib.Path(__file__).resolve().parents[2]
    / "backend/resources/model-pricing/model_prices_and_context_window.json"
).read_bytes()
LOCK = threading.Lock()


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *args):
        pass

    def send(self, body, kind="application/json", status=200):
        if not isinstance(body, bytes):
            body = json.dumps(body).encode()
        self.send_response(status)
        self.send_header("Content-Type", kind)
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path == "/pricing.json":
            return self.send(PRICING)
        if self.path == "/pricing.sha256":
            return self.send(hashlib.sha256(PRICING).hexdigest().encode(), "text/plain")
        if self.path.endswith("/models"):
            return self.send(
                {
                    "object": "list",
                    "data": [
                        {
                            "id": "gpt-4.1-mini",
                            "object": "model",
                            "created": 1,
                            "owned_by": "local-fixture",
                        }
                    ],
                }
            )
        return self.send({"error": "local fixture route not found"}, status=404)

    def do_POST(self):
        data = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))))
        rid = "local_" + uuid.uuid4().hex
        with LOCK:
            with (ROOT / "fixture-requests.jsonl").open("a") as f:
                f.write(
                    json.dumps(
                        {
                            "id": rid,
                            "path": self.path,
                            "stream": bool(data.get("stream")),
                            "model": data.get("model"),
                            "time": time.time(),
                        }
                    )
                    + "\n"
                )
        if "local-force-failure" in json.dumps(data):
            return self.send(
                {
                    "error": {
                        "message": "intentional local upstream error",
                        "type": "server_error",
                    }
                },
                status=500,
            )
        model = data.get("model", "gpt-4.1-mini")
        now = int(time.time())
        delay = 8 if "local-drain-stream" in json.dumps(data) else 1.5
        if self.path.endswith("/chat/completions"):
            result = {
                "id": "chatcmpl-" + rid,
                "object": "chat.completion",
                "created": now,
                "model": model,
                "choices": [
                    {
                        "index": 0,
                        "message": {"role": "assistant", "content": "LOCAL_CLUSTER_OK"},
                        "finish_reason": "stop",
                    }
                ],
                "usage": {
                    "prompt_tokens": 100,
                    "completion_tokens": 10,
                    "total_tokens": 110,
                    "prompt_tokens_details": {"cached_tokens": 20},
                },
            }
        else:
            result = {
                "id": "resp_" + rid,
                "object": "response",
                "created_at": now,
                "status": "completed",
                "model": model,
                "output": [
                    {
                        "id": "msg_" + rid,
                        "type": "message",
                        "status": "completed",
                        "role": "assistant",
                        "content": [
                            {
                                "type": "output_text",
                                "text": "LOCAL_CLUSTER_OK",
                                "annotations": [],
                            }
                        ],
                    }
                ],
                "usage": {
                    "input_tokens": 100,
                    "output_tokens": 10,
                    "total_tokens": 110,
                    "input_tokens_details": {"cached_tokens": 20},
                },
            }
        if data.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.send_header("Connection", "close")
            self.end_headers()

            def event(obj, name=None):
                self.wfile.write(
                    (
                        (("event: " + name + "\n") if name else "")
                        + "data: "
                        + json.dumps(obj)
                        + "\n\n"
                    ).encode()
                )
                self.wfile.flush()

            try:
                if self.path.endswith("/chat/completions"):
                    base = {
                        k: v for k, v in result.items() if k not in ("choices", "usage")
                    }
                    base["object"] = "chat.completion.chunk"
                    event(
                        {
                            **base,
                            "choices": [
                                {
                                    "index": 0,
                                    "delta": {"role": "assistant", "content": "LOCAL_"},
                                    "finish_reason": None,
                                }
                            ],
                        }
                    )
                    time.sleep(delay)
                    event(
                        {
                            **base,
                            "choices": [
                                {
                                    "index": 0,
                                    "delta": {"content": "CLUSTER_OK"},
                                    "finish_reason": None,
                                }
                            ],
                        }
                    )
                    event(
                        {
                            **base,
                            "choices": [
                                {"index": 0, "delta": {}, "finish_reason": "stop"}
                            ],
                            "usage": result["usage"],
                        }
                    )
                    self.wfile.write(b"data: [DONE]\n\n")
                    self.wfile.flush()
                else:
                    event(
                        {
                            "type": "response.created",
                            "response": {
                                **result,
                                "status": "in_progress",
                                "output": [],
                            },
                        },
                        "response.created",
                    )
                    event(
                        {
                            "type": "response.output_text.delta",
                            "item_id": "msg_" + rid,
                            "output_index": 0,
                            "content_index": 0,
                            "delta": "LOCAL_CLUSTER_OK",
                        },
                        "response.output_text.delta",
                    )
                    time.sleep(delay)
                    event(
                        {"type": "response.completed", "response": result},
                        "response.completed",
                    )
            except (BrokenPipeError, ConnectionResetError):
                pass
            self.close_connection = True
        else:
            time.sleep(0.2)
            self.send(result)


ThreadingHTTPServer(("127.0.0.1", 28190), Handler).serve_forever()
