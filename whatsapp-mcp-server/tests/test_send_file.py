"""Tests for send_file's base64 input.

The path-only interface could not send a file the *client* holds: MCP clients
generally run on a different machine from this server, so a locally produced
file (a screenshot, say) has no way onto this filesystem. Accepting the bytes
inline fixes that.
"""

import base64
import os
import stat
from pathlib import Path

import pytest

import whatsapp

# A real 1x1 PNG, so the magic bytes can be asserted on.
PNG_BYTES = base64.b64decode(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg=="
)


@pytest.fixture
def captured_send(monkeypatch, tmp_path):
    """Captures the media_path the bridge would have been asked to send."""
    seen: dict[str, object] = {}
    monkeypatch.setattr(whatsapp, "MEDIA_TEMP_DIR", str(tmp_path))
    monkeypatch.setattr(whatsapp, "_get_headers", lambda: {})

    class FakeResponse:
        status_code = 200

        @staticmethod
        def json():
            return {"success": True, "message_id": "MSG1", "recipient": "synthetic-recipient"}

    def fake_post(url, json=None, headers=None, timeout=None):
        seen["media_path"] = json["media_path"]
        seen["payload"] = json
        seen["mode"] = stat.S_IMODE(os.stat(json["media_path"]).st_mode)
        # Read it here: the real bridge reads during this call, so this is the
        # only point at which the temp file is guaranteed to still exist.
        with open(json["media_path"], "rb") as f:
            seen["content"] = f.read()
        return FakeResponse()

    monkeypatch.setattr(whatsapp.requests, "post", fake_post)
    return seen


def test_default_staging_root_is_tmp():
    assert whatsapp.MEDIA_TEMP_DIR == "/tmp"


def test_sends_a_file_given_as_base64(captured_send):
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="screenshot.png",
    )

    assert result["success"] is True
    assert result["message_id"] == "MSG1"
    # The bridge received the real bytes, not a re-encoding.
    assert captured_send["content"] == PNG_BYTES
    # And the name is preserved, since the extension drives how WhatsApp renders it.
    assert Path(captured_send["media_path"]).name == "screenshot.png"
    assert captured_send["mode"] == 0o600
    assert captured_send["payload"] == {
        "recipient": "synthetic-recipient",
        "media_path": captured_send["media_path"],
    }


def test_removes_the_temp_file_afterwards(captured_send):
    whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="screenshot.png",
    )

    # Otherwise every send leaks a copy into the media volume.
    written = Path(captured_send["media_path"])
    assert not written.exists()
    assert not written.parent.exists()
    assert written.parent.parent.is_dir()
    assert list(written.parent.parent.iterdir()) == []


@pytest.mark.parametrize("inputs", [{}, {"filename": None}])
def test_rejects_base64_without_a_filename(captured_send, inputs):
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        **inputs,
    )

    assert result["success"] is False
    assert "filename" in result["error"]
    assert "media_path" not in captured_send


def test_rejects_invalid_base64(captured_send):
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64="not!valid!base64",
        filename="x.png",
    )

    assert result["success"] is False
    assert "base64" in result["error"].lower()


def test_rejects_both_inputs_at_once(captured_send):
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        media_path="/some/path.png",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="x.png",
    )

    assert result["success"] is False
    assert "not both" in result["error"]


def test_rejects_neither_input(captured_send):
    result = whatsapp.send_file(recipient="synthetic-recipient")

    assert result["success"] is False
    assert "media_path" in result["error"]


def test_writes_somewhere_the_bridge_will_read_from(captured_send):
    """The bridge validates media_path against an allowlist before reading it."""
    whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="screenshot.png",
    )

    written = Path(captured_send["media_path"])
    assert written.parent.parent == Path(whatsapp.MEDIA_TEMP_DIR) / "whatsapp-outgoing"


@pytest.mark.parametrize("filename", ["../../etc/passwd", "/absolute/folder/passwd"])
def test_a_filename_cannot_escape_the_temp_directory(captured_send, filename):
    whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename=filename,
    )

    # Only the basename is used, so the write stays put. The bridge separately
    # rejects any path containing "..", so this must not produce one.
    written = str(captured_send["media_path"])
    assert ".." not in written
    assert Path(written).name == "passwd"
    assert Path(written).parent.parent == Path(whatsapp.MEDIA_TEMP_DIR) / "whatsapp-outgoing"


def test_still_accepts_a_server_side_path(captured_send, tmp_path):
    existing = tmp_path / "already-here.png"
    existing.write_bytes(PNG_BYTES)

    result = whatsapp.send_file(recipient="synthetic-recipient", media_path=str(existing))

    assert result["success"] is True
    assert captured_send["media_path"] == str(existing)
    # A caller-owned file must survive the send.
    assert existing.exists()
    assert existing.read_bytes() == PNG_BYTES
    assert not (tmp_path / "whatsapp-outgoing").exists()


@pytest.mark.parametrize("filename", ["", ".", "..", "/", "folder/", "folder/.", "folder/.."])
def test_directory_or_empty_names_fall_back_to_upload(captured_send, filename):
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename=filename,
    )

    assert result["success"] is True
    assert Path(captured_send["media_path"]).name == "upload"
    assert captured_send["content"] == PNG_BYTES
    assert captured_send["mode"] == 0o600
    assert not Path(captured_send["media_path"]).parent.exists()


def test_same_filename_uses_distinct_directories(captured_send, monkeypatch):
    original_post = whatsapp.requests.post
    paths = []

    def nested_post(*args, **kwargs):
        paths.append(Path(kwargs["json"]["media_path"]))
        if len(paths) == 1:
            inner = whatsapp.send_file(
                recipient="synthetic-recipient",
                file_content_base64=base64.b64encode(b"inner content").decode(),
                filename="same.txt",
            )
            assert inner["success"] is True
            assert not paths[1].parent.exists()
            assert paths[0].read_bytes() == PNG_BYTES
        return original_post(*args, **kwargs)

    monkeypatch.setattr(whatsapp.requests, "post", nested_post)
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="same.txt",
    )

    assert result["success"] is True
    assert len(paths) == 2
    assert paths[0].parent != paths[1].parent
    assert all(path.name == "same.txt" for path in paths)
    assert all(not path.parent.exists() for path in paths)


@pytest.mark.parametrize("failure", ["bridge", "http", "transport", "json"])
@pytest.mark.parametrize("path_only", [False, True])
def test_cleanup_on_bridge_failure(captured_send, monkeypatch, tmp_path, failure, path_only):
    original_post = whatsapp.requests.post

    class ErrorResponse:
        status_code = 503 if failure == "http" else 200
        text = "synthetic bridge failure"

        def json(self):
            if failure == "json":
                raise whatsapp.json.JSONDecodeError("invalid response", "", 0)
            return {"success": False, "message": self.text}

    def failing_post(*args, **kwargs):
        original_post(*args, **kwargs)
        if failure == "transport":
            raise whatsapp.requests.ConnectionError("synthetic transport failure")
        return ErrorResponse()

    monkeypatch.setattr(whatsapp.requests, "post", failing_post)
    existing = tmp_path / "caller-owned.txt"
    existing.write_bytes(PNG_BYTES)
    inputs = (
        {"media_path": str(existing)}
        if path_only
        else {"file_content_base64": base64.b64encode(PNG_BYTES).decode(), "filename": "upload.txt"}
    )
    result = whatsapp.send_file(recipient="synthetic-recipient", **inputs)

    assert result["success"] is False
    assert result["error"]
    assert existing.read_bytes() == PNG_BYTES
    written = Path(captured_send["media_path"])
    if path_only:
        assert written == existing
        assert not (tmp_path / "whatsapp-outgoing").exists()
    else:
        assert not written.exists()
        assert not written.parent.exists()
        assert list(written.parent.parent.iterdir()) == []


@pytest.mark.parametrize("failure", ["open", "write"])
def test_cleanup_on_staging_failure(captured_send, monkeypatch, tmp_path, failure):
    real_open = open
    staged = []

    class FailingWriter:
        def __init__(self, file):
            self.file = file

        def __enter__(self):
            return self

        def __exit__(self, *args):
            self.file.close()

        def fileno(self):
            return self.file.fileno()

        def write(self, content):
            self.file.write(content[:3])
            raise OSError("synthetic partial write failure")

    def failing_open(path, *args, **kwargs):
        staged.append(Path(path))
        if failure == "open":
            raise OSError("synthetic open failure")
        return FailingWriter(real_open(path, *args, **kwargs))

    monkeypatch.setattr(whatsapp, "open", failing_open, raising=False)
    result = whatsapp.send_file(
        recipient="synthetic-recipient",
        file_content_base64=base64.b64encode(PNG_BYTES).decode(),
        filename="staged.txt",
    )

    assert result["success"] is False
    assert "synthetic" in result["error"]
    assert len(staged) == 1
    assert not staged[0].exists()
    assert not staged[0].parent.exists()
    assert list((tmp_path / "whatsapp-outgoing").iterdir()) == []
    assert "media_path" not in captured_send
