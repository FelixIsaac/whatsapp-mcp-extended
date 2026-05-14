"""Local voice-message transcription via mlx-whisper.

Optional feature — install with `uv sync --extra transcribe`. Apple Silicon only.
The mlx-whisper dependency is gated behind a platform marker in pyproject.toml,
so it's a no-op on non-arm64-Darwin machines.

On first call per model the underlying library downloads weights to
~/.cache/huggingface (~1.5 GB for whisper-large-v3-turbo). Subsequent calls
are local.
"""

from typing import Any

WHISPER_MODEL = "mlx-community/whisper-large-v3-turbo"

_INSTALL_HINT = (
    "mlx-whisper is not installed. Install with `uv sync --extra transcribe` "
    "(Apple Silicon / macOS arm64 only — the dependency is platform-gated)."
)


def _load_mlx_whisper() -> Any | None:
    """Lazy-import mlx_whisper. Returns the module or None if unavailable."""
    try:
        import mlx_whisper
    except ImportError:
        return None
    return mlx_whisper


def transcribe_file(file_path: str, language: str | None = None) -> dict[str, Any]:
    """Transcribe a local audio file.

    Args:
        file_path: Absolute path to an audio file (any format ffmpeg can decode).
        language: Optional ISO-639-1 code (e.g. "en", "hr", "de"). Auto-detected if omitted.

    Returns:
        Dict with {success, text, language} on success, or {success: False, message}.
    """
    mlx_whisper = _load_mlx_whisper()
    if mlx_whisper is None:
        return {"success": False, "message": _INSTALL_HINT}

    try:
        result = mlx_whisper.transcribe(
            file_path,
            path_or_hf_repo=WHISPER_MODEL,
            language=language,
        )
    except Exception as e:
        return {"success": False, "message": f"Transcription failed: {e}"}

    return {
        "success": True,
        "text": (result.get("text") or "").strip(),
        "language": result.get("language"),
    }
