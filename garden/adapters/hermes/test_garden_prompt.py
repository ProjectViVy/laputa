import hashlib
import importlib.util
import json
import os
from pathlib import Path
from tempfile import TemporaryDirectory
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import threading
import time
from types import ModuleType, SimpleNamespace
import unittest
from unittest.mock import patch
from urllib.error import HTTPError

from garden_prompt import GardenPromptConfig, make_prompt_hook


class GardenPromptTests(unittest.TestCase):
    def setUp(self):
        self.temp = TemporaryDirectory(prefix="garden-hermes-")
        self.addCleanup(self.temp.cleanup)
        self.home = Path(self.temp.name)
        self.secret = self.home / "credential"
        self.secret.write_text("fixture-private-token")
        self.secret.chmod(0o600)
        self.soul = self.home / "SOUL.md"
        self.soul.write_text("ORIGINAL_NATIVE_SOUL")
        self.config = GardenPromptConfig("http://127.0.0.1:4321", "binding-A", str(self.secret), "existing-host", str(self.home), "0.21.7")
        self.projection = {"binding_id": "binding-A", "source_revisions": {"identity": 3}, "hash": "sha256:fixture", "markdown": "[Garden Persona]\n## IDENTITY.MD (revision 3)\nLATEST\n[/Garden Persona]"}

    def test_additive_context_exact_native_metadata_without_transcript_or_file_write(self):
        sent = []
        def transport(req, timeout):
            sent.append((req, timeout))
            return self.projection
        hook = make_prompt_hook(self.config, transport)
        result = hook(session_id="session-A", platform="cli", parent_session_id="", user_message="PRIVATE_PROMPT", conversation_history=["PRIVATE_HISTORY"], future_native_kwarg=True)
        self.assertEqual(result, {"context": self.projection["markdown"]})
        req, timeout = sent[0]
        body = json.loads(req.data)
        self.assertEqual(body, {"adapter_version": "garden-native-prompt/v1", "host_version": "0.21.7", "host_instance": "existing-host", "session_id": "session-A", "scope_fingerprint": hashlib.sha256(str(self.home.resolve()).encode()).hexdigest()})
        self.assertEqual(req.get_header("Authorization"), "Bearer fixture-private-token")
        self.assertEqual(req.get_header("X-garden-session"), "session-A")
        self.assertEqual(timeout, 1.0)
        self.assertEqual(self.soul.read_text(), "ORIGINAL_NATIVE_SOUL")
        with patch.dict(os.environ, {"HERMES_HOME": "/unrelated/ambient-profile"}):
            self.assertEqual(hook(session_id="session-B"), result)
        self.assertEqual(json.loads(sent[-1][0].data)["scope_fingerprint"], body["scope_fingerprint"])

    def test_parent_ephemeral_invalid_session_and_version_omit_before_transport(self):
        calls = []
        hook = make_prompt_hook(self.config, lambda *args: calls.append(args))
        wrong = GardenPromptConfig(self.config.endpoint, "binding-A", str(self.secret), "existing-host", str(self.home), "unknown")
        with self.assertLogs("garden.hermes", level="WARNING"):
            for kwargs in [{}, {"session_id": ""}, {"session_id": "bad\nheader"}, {"session_id": "A", "parent_session_id": "parent"}, {"session_id": "A", "_persist_disabled": True}, {"session_id": "A", "agent_id": "worker"}]:
                self.assertEqual(hook(**kwargs), {})
            self.assertEqual(make_prompt_hook(wrong, lambda *args: calls.append(args))(session_id="A"), {})
        self.assertEqual(calls, [])

    def test_failure_after_success_has_no_cached_body_or_private_diagnostic(self):
        outcome = [self.projection]
        def transport(req, timeout):
            value = outcome[0]
            if isinstance(value, Exception):
                raise value
            return value
        hook = make_prompt_hook(self.config, transport)
        self.assertEqual(hook(session_id="A"), {"context": self.projection["markdown"]})
        with self.assertLogs("garden.hermes", level="WARNING") as logs:
            for failure in [HTTPError(self.config.endpoint, 403, "PRIVATE_RESPONSE", {}, None), HTTPError(self.config.endpoint, 409, "PRIVATE_RESPONSE", {}, None), TimeoutError("fixture-private-token PRIVATE_RESPONSE"), {**self.projection, "markdown": "😄" * 6001}, {**self.projection, "binding_id": "binding-B"}]:
                outcome[0] = failure
                self.assertEqual(hook(session_id="A"), {})
        self.assertNotIn("PRIVATE_RESPONSE", " ".join(logs.output))
        self.assertNotIn("private-token", " ".join(logs.output))
        self.assertNotIn("LATEST", " ".join(logs.output))

    def test_credential_requires_private_regular_owner_file(self):
        calls = []
        hook = make_prompt_hook(self.config, lambda *args: calls.append(args))
        with self.assertLogs("garden.hermes", level="WARNING"):
            self.secret.chmod(0o644)
            self.assertEqual(hook(session_id="A"), {})
            self.secret.unlink()
            self.secret.symlink_to(self.soul)
            self.assertEqual(hook(session_id="A"), {})
        self.assertEqual(calls, [])

    def test_registration_is_inert_until_configured_and_uses_actual_native_home_version(self):
        spec = importlib.util.spec_from_file_location("garden_hermes_test_plugin", Path(__file__).with_name("__init__.py"), submodule_search_locations=[str(Path(__file__).parent)])
        plugin = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(plugin)
        hooks = []
        empty = SimpleNamespace(get_config=lambda key, default=None: default, register_hook=lambda *args: hooks.append(args))
        plugin.register(empty)
        self.assertEqual(hooks, [])
        fields = {key: getattr(self.config, key) for key in ["endpoint", "binding_id", "credential_file", "host_instance", "hermes_home"]}
        native = ModuleType("hermes_constants")
        native.get_hermes_home = lambda: self.home
        version = ModuleType("hermes_cli.version_info")
        version.get_version_info = lambda: SimpleNamespace(display_version="0.21.7")
        package = ModuleType("hermes_cli")
        package.__path__ = []
        ctx = SimpleNamespace(get_config=lambda key, default=None: fields.get(key, default), register_hook=lambda *args: hooks.append(args))
        # There is deliberately no register_system_prompt_section or memory API.
        with patch.dict("sys.modules", {spec.name: plugin, "hermes_constants": native, "hermes_cli": package, "hermes_cli.version_info": version}):
            plugin.register(ctx)
            self.assertEqual([x[0] for x in hooks], ["pre_llm_call"])
            native.get_hermes_home = lambda: self.home / "other-profile"
            plugin.register(ctx)
            self.assertEqual(len(hooks), 1)

    def test_actual_http_timeout_omits_context_within_host_budget(self):
        release = threading.Event()
        class Slow(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass
            def do_POST(self):
                release.wait(2)
                self.send_response(200)
                self.end_headers()
        server = ThreadingHTTPServer(("127.0.0.1", 0), Slow)
        thread = threading.Thread(target=server.serve_forever, daemon=True)
        thread.start()
        cfg = GardenPromptConfig(f"http://127.0.0.1:{server.server_port}", "binding-A", str(self.secret), "existing-host", str(self.home), "0.21.7")
        try:
            started = time.monotonic()
            with self.assertLogs("garden.hermes", level="WARNING"):
                self.assertEqual(make_prompt_hook(cfg)(session_id="A"), {})
            self.assertLess(time.monotonic() - started, 1.5)
        finally:
            release.set()
            server.shutdown()
            server.server_close()

    def test_loopback_request_never_uses_environment_proxy(self):
        observed = {"target": 0, "proxy": 0}
        body = json.dumps(self.projection).encode()
        def handler(kind):
            class Capture(BaseHTTPRequestHandler):
                def log_message(self, *args):
                    pass
                def do_POST(self):
                    observed[kind] += 1
                    self.send_response(200)
                    self.end_headers()
                    self.wfile.write(body)
            return Capture
        target = ThreadingHTTPServer(("127.0.0.1", 0), handler("target"))
        proxy = ThreadingHTTPServer(("127.0.0.1", 0), handler("proxy"))
        for server in [target, proxy]:
            threading.Thread(target=server.serve_forever, daemon=True).start()
        cfg = GardenPromptConfig(f"http://127.0.0.1:{target.server_port}", "binding-A", str(self.secret), "existing-host", str(self.home), "0.21.7")
        try:
            proxy_url = f"http://127.0.0.1:{proxy.server_port}"
            with patch.dict(os.environ, {"http_proxy": proxy_url, "HTTP_PROXY": proxy_url, "no_proxy": "", "NO_PROXY": ""}):
                self.assertEqual(make_prompt_hook(cfg)(session_id="A"), {"context": self.projection["markdown"]})
            self.assertEqual(observed, {"target": 1, "proxy": 0})
        finally:
            for server in [target, proxy]:
                server.shutdown()
                server.server_close()

    def test_slow_stream_cannot_extend_total_http_deadline(self):
        body = json.dumps(self.projection).encode()
        class Drip(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass
            def do_POST(self):
                self.send_response(200)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers()
                try:
                    for piece in [body[:10], body[10:20], body[20:]]:
                        time.sleep(0.6)
                        self.wfile.write(piece)
                        self.wfile.flush()
                except OSError:
                    pass  # Expected after the bounded client closes its socket.
        server = ThreadingHTTPServer(("127.0.0.1", 0), Drip)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        cfg = GardenPromptConfig(f"http://127.0.0.1:{server.server_port}", "binding-A", str(self.secret), "existing-host", str(self.home), "0.21.7")
        try:
            started = time.monotonic()
            with self.assertLogs("garden.hermes", level="WARNING"):
                result = make_prompt_hook(cfg)(session_id="A")
            self.assertEqual(result, {})
            self.assertLess(time.monotonic() - started, 1.5)
        finally:
            server.shutdown()
            server.server_close()

    def test_localhost_works_when_name_service_is_unavailable(self):
        body = json.dumps(self.projection).encode()
        class Local(BaseHTTPRequestHandler):
            def log_message(self, *args):
                pass
            def do_POST(self):
                self.send_response(200)
                self.end_headers()
                self.wfile.write(body)
        server = ThreadingHTTPServer(("127.0.0.1", 0), Local)
        threading.Thread(target=server.serve_forever, daemon=True).start()
        cfg = GardenPromptConfig(f"http://localhost:{server.server_port}", "binding-A", str(self.secret), "existing-host", str(self.home), "0.21.7")
        try:
            with patch("socket.getaddrinfo", side_effect=OSError("name service unavailable")):
                self.assertEqual(make_prompt_hook(cfg)(session_id="A"), {"context": self.projection["markdown"]})
        finally:
            server.shutdown()
            server.server_close()


if __name__ == "__main__":
    unittest.main()
