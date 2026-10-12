from dataclasses import dataclass
import hashlib
from http.client import HTTPConnection
import json
import logging
import os
from pathlib import Path
import platform
import socket
import stat
import time
from urllib.error import HTTPError
from urllib.parse import quote, urlsplit
from urllib.request import Request

logger = logging.getLogger("garden.hermes")


@dataclass(frozen=True)
class GardenPromptConfig:
    endpoint: str
    binding_id: str
    credential_file: str
    host_instance: str
    hermes_home: str
    host_version: str


class _DeadlineSocket(socket.socket):
    """SocketIO uses recv_into; each read shares the original request deadline."""
    def __init__(self, *args, deadline, **kwargs):
        super().__init__(*args, **kwargs)
        self.deadline = deadline

    def remaining(self):
        remaining = self.deadline - time.monotonic()
        if remaining <= 0:
            raise TimeoutError("request deadline")
        self.settimeout(remaining)

    def recv_into(self, *args, **kwargs):
        self.remaining()
        return super().recv_into(*args, **kwargs)

    def sendall(self, *args, **kwargs):
        self.remaining()
        return super().sendall(*args, **kwargs)


def _transport(req, timeout):
    # A direct stdlib HTTP connection never consults environment proxies and
    # never follows redirects. No background fetch can outlive this callback.
    deadline = time.monotonic() + timeout
    url = urlsplit(req.full_url)
    connection = HTTPConnection(url.hostname, url.port, timeout=timeout)
    response = None
    try:
        last_error = None
        port = url.port if url.port is not None else 80
        # The accepted names are loopback only. Numeric candidates avoid an
        # unbounded name-service lookup before the socket deadline starts.
        addresses = [(socket.AF_INET, ("127.0.0.1", port)), (socket.AF_INET6, ("::1", port, 0, 0))] if url.hostname == "localhost" else [(socket.AF_INET6, ("::1", port, 0, 0))] if url.hostname == "::1" else [(socket.AF_INET, ("127.0.0.1", port))]
        for family, address in addresses:
            sock = _DeadlineSocket(family, socket.SOCK_STREAM, deadline=deadline)
            try:
                sock.remaining()
                sock.connect(address)
                connection.sock = sock
                break
            except OSError as error:
                sock.close()
                last_error = error
        if connection.sock is None:
            raise last_error or OSError("loopback connection unavailable")
        connection.request(req.get_method(), url.path or "/", body=req.data, headers=dict(req.header_items()))
        response = connection.getresponse()
        if not 200 <= response.status < 300:
            raise HTTPError(req.full_url, response.status, "Garden response", response.headers, None)
        body = response.read(96 * 1024 + 1)
        if len(body) > 96 * 1024 or time.monotonic() >= deadline:
            raise ValueError("projection size")
        return json.loads(body)
    finally:
        if response:
            response.close()
        connection.close()


def _credential(path):
    path = Path(path)
    if not path.is_absolute():
        raise ValueError("credential path")
    directory = path.parent.resolve(strict=True).stat()
    if directory.st_mode & 0o077 or directory.st_uid != os.getuid():
        raise ValueError("credential directory")
    fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW)
    with os.fdopen(fd, "rb") as file:
        info = os.fstat(file.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_mode & 0o077 or info.st_uid != os.getuid() or info.st_size > 4096:
            raise ValueError("credential file")
        token = file.read(4097).decode().strip()
    if not token or len(token) > 512 or any(c in token for c in "\r\n\0"):
        raise ValueError("credential format")
    return token


def make_prompt_hook(config: GardenPromptConfig, transport=None):
    """Capture the native profile once; never infer it from later ambient env."""
    send = transport or _transport
    endpoint = fingerprint = ""
    try:
        url = urlsplit(config.endpoint)
        if url.scheme != "http" or url.hostname not in ("localhost", "127.0.0.1", "::1") or url.username or url.password or url.path not in ("", "/") or url.query or url.fragment:
            raise ValueError("endpoint")
        home = Path(config.hermes_home).resolve(strict=True)
        if not home.is_dir() or not home.is_absolute():
            raise ValueError("home")
        endpoint = config.endpoint.rstrip("/")
        fingerprint = hashlib.sha256(str(home).encode()).hexdigest()
    except (OSError, ValueError):
        pass

    def callback(**kwargs):
        try:
            if platform.system() != "Linux" or platform.machine() not in ("x86_64", "amd64") or config.host_version != "0.21.7" or not endpoint or not fingerprint:
                logger.warning("Garden Persona omitted: unsupported_host")
                return {}
            session = kwargs.get("session_id")
            if not isinstance(session, str) or not session or len(session) > 512 or any(ord(c) < 32 or ord(c) == 127 for c in session) or kwargs.get("parent_session_id") or kwargs.get("_persist_disabled") or kwargs.get("agent_id"):
                logger.warning("Garden Persona omitted: unsupported_scope")
                return {}
            token = _credential(config.credential_file)
            body = {"adapter_version": "garden-native-prompt/v1", "host_version": config.host_version, "host_instance": config.host_instance, "session_id": session, "scope_fingerprint": fingerprint}
            req = Request(endpoint + "/v2/host-bindings/" + quote(config.binding_id, safe="") + "/projection/prompt", data=json.dumps(body).encode(), headers={"Content-Type": "application/json", "Authorization": "Bearer " + token, "X-Garden-Session": session}, method="POST")
            result = send(req, 1.0)
            if not isinstance(result, dict) or result.get("binding_id") != config.binding_id or not isinstance(result.get("markdown"), str) or not isinstance(result.get("hash"), str) or not isinstance(result.get("source_revisions"), dict) or len(result["markdown"]) > 6000:
                logger.warning("Garden Persona omitted: invalid_projection")
                return {}
            return {"context": result["markdown"]}
        except HTTPError as error:
            logger.warning("Garden Persona omitted: http_%d", error.code)
        except Exception:
            logger.warning("Garden Persona omitted: unavailable")
        return {}

    return callback
