from types import SimpleNamespace

import main


def _settings(allowed_hosts=None, enabled=True):
    security = SimpleNamespace(allowed_hosts=list(allowed_hosts or [])) if enabled else None
    return SimpleNamespace(transport_security=security)


def test_extend_allowed_hosts_appends_trimmed_values():
    settings = _settings(["127.0.0.1:*", "localhost:*"])

    added = main.extend_allowed_hosts(settings, " whatsapp_mcp:8081, mcp.example.com ,")

    assert added == ["whatsapp_mcp:8081", "mcp.example.com"]
    assert settings.transport_security.allowed_hosts == [
        "127.0.0.1:*",
        "localhost:*",
        "whatsapp_mcp:8081",
        "mcp.example.com",
    ]


def test_extend_allowed_hosts_empty_value_is_noop():
    settings = _settings(["127.0.0.1:*"])

    assert main.extend_allowed_hosts(settings, "") == []
    assert settings.transport_security.allowed_hosts == ["127.0.0.1:*"]


def test_extend_allowed_hosts_without_transport_security_is_noop():
    settings = _settings(enabled=False)

    assert main.extend_allowed_hosts(settings, "whatsapp_mcp:8081") == []
    assert settings.transport_security is None


def test_default_fastmcp_settings_accept_extension():
    """The module-level FastMCP instance has a mutable Host allowlist."""
    security = main.mcp.settings.transport_security
    assert security is not None
    before = list(security.allowed_hosts)
    try:
        main.extend_allowed_hosts(main.mcp.settings, "whatsapp_mcp:8081")
        assert "whatsapp_mcp:8081" in security.allowed_hosts
    finally:
        security.allowed_hosts[:] = before
