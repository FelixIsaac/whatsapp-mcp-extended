"""Semantic search ("recall") over WhatsApp message history.

Embeds every text message into a vector store kept inside messages.db itself,
then ranks matches to a natural-language query by cosine similarity.

Multilingual: uses paraphrase-multilingual-MiniLM-L12-v2 (50+ languages
including Croatian, German, Italian, French, Spanish, etc.).

Optional feature — install with `uv sync --extra recall`. The hard deps
(sentence-transformers, numpy) are lazy-imported so importing this module on
a default install does NOT fail; tool calls return a structured install hint
instead.

Design choices:
- Brute-force cosine search at query time — fine up to ~100k messages.
- Embeddings stored normalized, so dot product == cosine similarity.
- Daemon thread incrementally embeds pending messages in batches of 64.
- Indexer is started lazily on first user-facing call (not on import).
"""

from __future__ import annotations

import sqlite3
import threading
import time
from typing import Any

from .utils import MESSAGES_DB_PATH, logger

MODEL_NAME = "sentence-transformers/paraphrase-multilingual-MiniLM-L12-v2"
EMBED_DIM = 384
INDEX_BATCH_SIZE = 64

_INSTALL_HINT = (
    "The `recall` optional extra is not installed. Install with "
    "`uv sync --extra recall` (~470 MB of model weights will be cached "
    "to ~/.cache/huggingface on first use)."
)

_model = None
_model_lock = threading.Lock()
_indexer_thread: threading.Thread | None = None
_indexer_lock = threading.Lock()
_indexer_state: dict[str, Any] = {"status": "idle", "indexed": 0, "pending": 0}


def is_available() -> bool:
    """Return True if both sentence-transformers and numpy can be imported."""
    try:
        import numpy  # noqa: F401
        import sentence_transformers  # noqa: F401
    except ImportError:
        return False
    return True


def _get_model() -> Any:
    """Lazy-load the multilingual sentence-transformer model.

    First call downloads ~470 MB of model weights to ~/.cache/huggingface.
    """
    global _model
    if _model is not None:
        return _model
    with _model_lock:
        if _model is None:
            from sentence_transformers import SentenceTransformer

            logger.info("[recall] loading embedding model %s", MODEL_NAME)
            t0 = time.time()
            _model = SentenceTransformer(MODEL_NAME)
            logger.info("[recall] embedding model loaded in %.1fs", time.time() - t0)
    return _model


def _embed(texts: list[str]) -> Any:
    """Encode texts to normalized embeddings. Returns numpy.ndarray of shape (N, D)."""
    model = _get_model()
    return model.encode(
        texts,
        normalize_embeddings=True,
        convert_to_numpy=True,
        show_progress_bar=False,
        batch_size=INDEX_BATCH_SIZE,
    )


def _connect() -> sqlite3.Connection:
    return sqlite3.connect(MESSAGES_DB_PATH, timeout=10.0, isolation_level=None)


def _ensure_table(conn: sqlite3.Connection) -> None:
    conn.execute(
        """
        CREATE TABLE IF NOT EXISTS message_embeddings (
            message_id  TEXT NOT NULL,
            chat_jid    TEXT NOT NULL,
            embedding   BLOB NOT NULL,
            model       TEXT NOT NULL,
            embedded_at INTEGER NOT NULL,
            PRIMARY KEY (message_id, chat_jid)
        )
        """
    )
    conn.execute(
        "CREATE INDEX IF NOT EXISTS idx_msgemb_chat ON message_embeddings(chat_jid)"
    )


def _index_pending() -> None:
    """Walk through unembedded messages in batches and embed them.

    Runs in a daemon thread. All exceptions are caught so the thread exits
    cleanly without polluting stderr; the failure mode is recorded in
    _indexer_state for index_status() to surface.
    """
    import numpy as np

    try:
        conn = _connect()
        _ensure_table(conn)
    except Exception as exc:
        logger.warning("[recall] indexer could not open store: %s", exc)
        _indexer_state["status"] = "error"
        _indexer_state["last_error"] = str(exc)
        return

    _indexer_state["status"] = "running"
    try:
        while True:
            rows = conn.execute(
                """
                SELECT m.id, m.chat_jid, m.content
                FROM messages m
                LEFT JOIN message_embeddings e
                    ON e.message_id = m.id AND e.chat_jid = m.chat_jid
                WHERE m.content IS NOT NULL
                  AND TRIM(m.content) != ''
                  AND e.message_id IS NULL
                LIMIT ?
                """,
                (INDEX_BATCH_SIZE,),
            ).fetchall()
            if not rows:
                break

            texts = [r[2] for r in rows]
            try:
                embeddings = _embed(texts)
            except Exception as exc:
                logger.exception("[recall] embedding batch failed: %s", exc)
                _indexer_state["status"] = "error"
                _indexer_state["last_error"] = str(exc)
                return

            now = int(time.time())
            params = [
                (r[0], r[1], emb.astype(np.float32).tobytes(), MODEL_NAME, now)
                for r, emb in zip(rows, embeddings, strict=False)
            ]
            conn.executemany(
                """
                INSERT OR REPLACE INTO message_embeddings
                    (message_id, chat_jid, embedding, model, embedded_at)
                VALUES (?, ?, ?, ?, ?)
                """,
                params,
            )
            _indexer_state["indexed"] = (_indexer_state.get("indexed", 0) or 0) + len(rows)
        _indexer_state["status"] = "done"
        logger.info("[recall] indexer finished, %d messages indexed total",
                    _indexer_state["indexed"])
    finally:
        try:
            conn.close()
        except Exception:
            pass


def _ensure_indexer_running() -> None:
    """Start the daemon indexer thread if not already running."""
    global _indexer_thread
    with _indexer_lock:
        if _indexer_thread is not None and _indexer_thread.is_alive():
            return
        try:
            conn = _connect()
            _ensure_table(conn)
            row = conn.execute(
                """
                SELECT COUNT(*) FROM messages m
                LEFT JOIN message_embeddings e
                    ON e.message_id = m.id AND e.chat_jid = m.chat_jid
                WHERE m.content IS NOT NULL AND TRIM(m.content) != ''
                  AND e.message_id IS NULL
                """
            ).fetchone()
            _indexer_state["pending"] = row[0] if row else 0
            conn.close()
        except Exception:
            pass

        _indexer_thread = threading.Thread(
            target=_index_pending, daemon=True, name="recall-indexer"
        )
        _indexer_thread.start()


def index_status() -> dict[str, Any]:
    """Return how far the background indexer has gotten.

    Returns a structured install hint if the `recall` extra isn't installed.
    """
    if not is_available():
        return {"success": False, "message": _INSTALL_HINT}

    _ensure_indexer_running()
    try:
        conn = _connect()
        _ensure_table(conn)
        embedded = conn.execute("SELECT COUNT(*) FROM message_embeddings").fetchone()[0]
        total = conn.execute(
            "SELECT COUNT(*) FROM messages WHERE content IS NOT NULL AND TRIM(content) != ''"
        ).fetchone()[0]
        conn.close()
    except Exception as exc:
        return {"success": False, "message": str(exc)}
    return {
        "success": True,
        "status": _indexer_state.get("status", "idle"),
        "embedded": embedded,
        "total": total,
        "remaining": max(total - embedded, 0),
        "model": MODEL_NAME,
    }


def recall(
    query: str,
    chat_jid: str | None = None,
    sender: str | None = None,
    after: str | None = None,
    before: str | None = None,
    is_from_me: bool | None = None,
    limit: int = 10,
) -> dict[str, Any]:
    """Semantic search over WhatsApp message history.

    Args:
        query: Natural-language query, in any supported language.
        chat_jid: Optional, restrict to one chat.
        sender: Optional sender JID or LID, restrict to one author.
        after: Optional ISO-8601 datetime; only messages with timestamp >= this.
        before: Optional ISO-8601 datetime; only messages with timestamp <= this.
        is_from_me: Optional, True for outgoing only, False for incoming only.
        limit: Max results to return (default 10).

    Returns:
        Dict with `results` (list of matches with similarity score) and
        `index_status` (so the caller can see if indexing is still in flight).
        Returns a structured install hint if the `recall` extra isn't installed.
    """
    if not is_available():
        return {"success": False, "message": _INSTALL_HINT}

    import numpy as np

    _ensure_indexer_running()
    qvec = _embed([query])[0]

    conn = _connect()
    _ensure_table(conn)

    sql_parts = [
        """
        SELECT m.id, m.chat_jid, m.sender, m.content, m.timestamp, m.is_from_me,
               c.name AS chat_name, e.embedding
        FROM message_embeddings e
        JOIN messages m
            ON m.id = e.message_id AND m.chat_jid = e.chat_jid
        LEFT JOIN chats c ON c.jid = m.chat_jid
        WHERE 1=1
        """
    ]
    params: list[Any] = []
    if chat_jid:
        sql_parts.append(" AND m.chat_jid = ?")
        params.append(chat_jid)
    if sender:
        sql_parts.append(" AND m.sender = ?")
        params.append(sender)
    if after:
        sql_parts.append(" AND m.timestamp >= ?")
        params.append(after)
    if before:
        sql_parts.append(" AND m.timestamp <= ?")
        params.append(before)
    if is_from_me is not None:
        sql_parts.append(" AND m.is_from_me = ?")
        params.append(1 if is_from_me else 0)

    rows = conn.execute("".join(sql_parts), params).fetchall()
    conn.close()

    if not rows:
        return {
            "success": True,
            "results": [],
            "index_status": index_status(),
            "warning": (
                "No embedded messages match your filters yet. The indexer may still"
                " be warming up — call `recall_index_status` to check progress."
            ),
        }

    embeddings = np.stack([np.frombuffer(r[7], dtype=np.float32) for r in rows])
    sims = embeddings @ qvec  # both normalized -> dot == cosine
    top = np.argsort(-sims)[:limit]

    results = []
    for i in top:
        r = rows[int(i)]
        results.append(
            {
                "message_id": r[0],
                "chat_jid": r[1],
                "chat_name": r[6],
                "sender": r[2],
                "is_from_me": bool(r[5]),
                "timestamp": r[4],
                "content": r[3],
                "similarity": round(float(sims[int(i)]), 4),
            }
        )

    return {"success": True, "results": results, "index_status": index_status()}
