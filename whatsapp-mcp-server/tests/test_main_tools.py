import importlib


def test_legacy_tool_profile_keeps_compat_tools(monkeypatch):
    monkeypatch.delenv("WHATSAPP_MCP_TOOL_PROFILE", raising=False)

    import main

    main = importlib.reload(main)
    tool_names = {tool.name for tool in main.mcp._tool_manager.list_tools()}

    assert "get_contact_chats" in tool_names
    assert "manage_group" in tool_names
    assert "manage_newsletter" in tool_names


def test_core_tool_profile_hides_deprecated_and_advanced_tools(monkeypatch):
    monkeypatch.setenv("WHATSAPP_MCP_TOOL_PROFILE", "core")

    import main

    main = importlib.reload(main)
    tool_names = {tool.name for tool in main.mcp._tool_manager.list_tools()}

    assert "get_contact_context" in tool_names
    assert "manage_nickname" in tool_names
    assert "get_contact_chats" not in tool_names
    assert "create_group" not in tool_names
    assert "manage_group" not in tool_names
    assert len(tool_names) <= 20


def test_merged_tool_invalid_action_guides_model(monkeypatch):
    monkeypatch.setenv("WHATSAPP_MCP_TOOL_PROFILE", "core")

    import main

    main = importlib.reload(main)
    result = main.manage_nickname("rename", jid="123@s.whatsapp.net", nickname="Felix")

    assert result["success"] is False
    assert result["use_tool"] == "manage_nickname"
    assert result["allowed_actions"] == ["set", "get", "remove", "list"]
