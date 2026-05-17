import importlib
from pathlib import Path

DEFAULT_TOOLS = {
    "search_contacts",
    "list_messages",
    "list_chats",
    "get_chat",
    "get_message_context",
    "send_message",
    "send_file",
    "send_audio_message",
    "download_media",
    "list_all_contacts",
    "get_contact_context",
    "send_reaction",
    "get_group_info",
    "create_poll",
    "get_profile_picture",
}

ALL_TOOLS = DEFAULT_TOOLS | {
    "manage_nickname",
    "edit_message",
    "delete_message",
    "mark_read",
    "manage_group",
    "request_history",
    "set_presence",
    "subscribe_presence",
    "get_blocklist",
    "manage_blocklist",
    "manage_newsletter",
}


def reload_main(monkeypatch, toolsets: str | None = None):
    if toolsets is None:
        monkeypatch.delenv("WHATSAPP_MCP_TOOLSETS", raising=False)
    else:
        monkeypatch.setenv("WHATSAPP_MCP_TOOLSETS", toolsets)
    monkeypatch.delenv("WHATSAPP_MCP_TOOLS", raising=False)

    import main

    return importlib.reload(main)


def tool_names(main) -> set[str]:
    return {tool.name for tool in main.mcp._tool_manager.list_tools()}


def test_default_tool_surface_is_small_and_safe(monkeypatch):
    main = reload_main(monkeypatch)

    assert tool_names(main) == DEFAULT_TOOLS
    assert "manage_group" not in tool_names(main)
    assert "delete_message" not in tool_names(main)
    assert "manage_blocklist" not in tool_names(main)
    assert "manage_newsletter" not in tool_names(main)


def test_all_toolsets_expose_full_curated_surface(monkeypatch):
    main = reload_main(monkeypatch, "all")

    assert tool_names(main) == ALL_TOOLS
    assert "get_contact_chats" not in tool_names(main)
    assert "create_group" not in tool_names(main)
    assert "block_user" not in tool_names(main)
    assert "follow_newsletter" not in tool_names(main)


def test_explicit_tool_allowlist_can_add_single_advanced_tool(monkeypatch):
    monkeypatch.delenv("WHATSAPP_MCP_TOOLSETS", raising=False)
    monkeypatch.setenv("WHATSAPP_MCP_TOOLS", "manage_group")

    import main

    main = importlib.reload(main)
    names = tool_names(main)

    assert "manage_group" in names
    assert "manage_blocklist" not in names


def test_all_exposed_tools_have_titles_and_annotations(monkeypatch):
    main = reload_main(monkeypatch, "all")

    for exposed_tool in main.mcp._tool_manager.list_tools():
        assert exposed_tool.title, exposed_tool.name
        assert exposed_tool.annotations is not None, exposed_tool.name
        assert exposed_tool.annotations.title == exposed_tool.title


def test_enum_schemas_for_action_and_sort_fields(monkeypatch):
    main = reload_main(monkeypatch, "all")

    manage_group = main.mcp._tool_manager.get_tool("manage_group")
    set_presence = main.mcp._tool_manager.get_tool("set_presence")
    list_chats = main.mcp._tool_manager.get_tool("list_chats")
    manage_blocklist = main.mcp._tool_manager.get_tool("manage_blocklist")

    assert manage_group.parameters["properties"]["action"]["enum"] == [
        "create",
        "add_members",
        "remove_members",
        "promote_admin",
        "demote_admin",
        "leave",
        "update",
    ]
    assert set_presence.parameters["properties"]["presence"]["enum"] == ["available", "unavailable"]
    assert list_chats.parameters["properties"]["sort_by"]["enum"] == ["last_active", "name"]
    assert manage_blocklist.parameters["properties"]["action"]["enum"] == ["block", "unblock"]


def test_merged_tool_invalid_action_guides_model(monkeypatch):
    main = reload_main(monkeypatch, "all")
    result = main.manage_nickname("rename", jid="123@s.whatsapp.net", nickname="Felix")

    assert result["success"] is False
    assert result["use_tool"] == "manage_nickname"
    assert result["allowed_actions"] == ["set", "get", "remove", "list"]


def test_manage_group_forwards_create_action(monkeypatch):
    main = reload_main(monkeypatch, "all")
    called = {}

    def fake_create_group(name, participants):
        called["args"] = (name, participants)
        return {"success": True, "group_jid": "1@g.us"}

    monkeypatch.setattr(main, "whatsapp_create_group", fake_create_group)

    result = main.manage_group("create", name="Ops", participants=["1@s.whatsapp.net"])

    assert result["success"] is True
    assert called["args"] == ("Ops", ["1@s.whatsapp.net"])


def test_docker_mcp_entrypoint_uses_curated_main_server():
    root = Path(__file__).resolve().parents[2]
    dockerfile = (root / "Dockerfile.mcp").read_text(encoding="utf-8")
    compose = (root / "docker-compose.yaml").read_text(encoding="utf-8")

    assert "python main.py" in dockerfile
    assert "python gradio-main.py" not in dockerfile
    assert "MCP_TRANSPORT=streamable-http" in compose
    assert "WHATSAPP_MCP_TOOLSETS=${WHATSAPP_MCP_TOOLSETS:-core,send,media}" in compose
