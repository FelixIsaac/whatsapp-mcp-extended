import importlib


def test_curated_tool_surface_hides_deprecated_wrappers():
    import main

    main = importlib.reload(main)
    tool_names = {tool.name for tool in main.mcp._tool_manager.list_tools()}

    assert len(tool_names) == 25
    assert "get_contact_context" in tool_names
    assert "manage_nickname" in tool_names
    assert "manage_group" in tool_names
    assert "manage_newsletter" in tool_names
    assert "get_contact_chats" not in tool_names
    assert "create_group" not in tool_names
    assert "block_user" not in tool_names
    assert "follow_newsletter" not in tool_names


def test_all_exposed_tools_have_annotations():
    import main

    main = importlib.reload(main)
    for tool in main.mcp._tool_manager.list_tools():
        assert tool.annotations is not None, tool.name


def test_merged_tool_invalid_action_guides_model():
    import main

    main = importlib.reload(main)
    result = main.manage_nickname("rename", jid="123@s.whatsapp.net", nickname="Felix")

    assert result["success"] is False
    assert result["use_tool"] == "manage_nickname"
    assert result["allowed_actions"] == ["set", "get", "remove", "list"]
