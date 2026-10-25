import socket
import socketserver
import threading
import time
import unittest
import lab_network
class NetworkTests(unittest.TestCase):
    def test_delay_and_pre_forward_disconnect(self):
        class Echo(socketserver.BaseRequestHandler):
            def handle(self):
                while data:=self.request.recv(4096):self.request.sendall(data)
        with socketserver.ThreadingTCPServer(('127.0.0.1',0),Echo) as echo:
            threading.Thread(target=echo.serve_forever,daemon=True).start()
            try:
                with lab_network.proxy(echo.server_address,delay_ms=20,reset_every=2) as proxy:
                    with socket.create_connection(proxy.server_address,timeout=2) as s:
                        start=time.monotonic();s.sendall(b'hello');self.assertEqual(s.recv(5),b'hello');self.assertGreaterEqual(time.monotonic()-start,.03)
                    with socket.create_connection(proxy.server_address,timeout=2) as s:
                        self.assertEqual(s.recv(1),b'')
            finally:echo.shutdown()
