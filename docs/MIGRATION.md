# Database Migration Guide

This guide covers how to run database migrations safely for existing WhatsApp MCP installations.

---

## Overview

**What:** When we add new features (metadata fields, indexes, new tables), we provide migration scripts instead of requiring you to rebuild containers from scratch.

**Why:** Preserves your existing message history and chat data. Safe, non-destructive updates.

**Safety:** All migrations are **append-only** - they only add columns, indexes, and tables. No deletions or destructive changes.

---

## Before You Start

### Backup Your Database

```bash
# Stop the bridge
docker-compose down

# Backup the database
cp whatsapp-bridge/store/messages.db whatsapp-bridge/store/messages.db.backup-$(date +%Y%m%d-%H%M%S)

# Or in Docker container
docker cp whatsapp-bridge:/app/whatsapp-bridge/store/messages.db ./messages.db.backup
```

### Verify You Have the Migration File

```bash
# Check if migration exists
ls -la whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Or for Docker:
ls -la migrations/001_add_metadata_fields.sql
```

---

## Running Migrations

### Option 1: Local Development (Direct File Access)

**Using sqlite3 CLI:**

```bash
# Navigate to project root
cd whatsapp-mcp-extended

# Run migration
sqlite3 whatsapp-bridge/store/messages.db < whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Verify success
sqlite3 whatsapp-bridge/store/messages.db ".schema messages" | head -20
```

**Expected output:** Should show new columns like `quoted_message_id`, `edit_count`, `is_forwarded`

### Option 2: Docker (Container Running)

**While container is running:**

```bash
# Copy migration into container
docker cp whatsapp-bridge/migrations/001_add_metadata_fields.sql whatsapp-bridge:/tmp/

# Execute migration inside container
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db < whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Verify (check logs for any errors)
docker logs whatsapp-bridge | tail -20
```

**Or in one command:**

```bash
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db \
  "SELECT name FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%' ORDER BY name;"
```

### Option 3: Docker (Stop Container First)

**Safest approach:**

```bash
# Stop containers (saves state, doesn't destroy)
docker-compose down

# Run migration (database accessible locally in volume)
sqlite3 whatsapp-bridge/store/messages.db < whatsapp-bridge/migrations/001_add_metadata_fields.sql

# Start containers
docker-compose up -d

# Verify bridge is healthy
docker-compose logs -f whatsapp-bridge | grep -E "(Connected|ERROR)" | head -5
```

---

## Verifying Migrations

### Check Indexes Were Created

```bash
sqlite3 whatsapp-bridge/store/messages.db \
  "SELECT name, sql FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%' ORDER BY name;"
```

**Expected output:** Should list indexes like:
- `idx_messages_chat_timestamp`
- `idx_messages_sender`
- `idx_messages_media_type`
- `idx_chats_last_message_time`

### Check New Columns Were Added

```bash
sqlite3 whatsapp-bridge/store/messages.db "PRAGMA table_info(messages);" | grep -E "(quoted|edit|forward)"
```

**Expected output:** Rows containing `quoted_message_id`, `edit_count`, `is_forwarded`, etc.

### Check New Tables Were Created

```bash
sqlite3 whatsapp-bridge/store/messages.db \
  "SELECT name FROM sqlite_master WHERE type='table' AND name IN ('message_edits', 'message_reactions', 'group_members');"
```

**Expected output:** Should list 3 tables:
- `message_edits`
- `message_reactions`
- `group_members`

---

## Troubleshooting

### Error: "database is locked"

**Cause:** Bridge is still writing to the database

**Solution:**
```bash
# Option 1: Stop containers first
docker-compose down

# Option 2: Or give it more time to release lock
sleep 5 && sqlite3 whatsapp-bridge/store/messages.db < migrations/001_add_metadata_fields.sql
```

### Error: "column already exists"

**Cause:** Migration already ran successfully

**Solution:**
- This is OK - migrations are idempotent (use `IF NOT EXISTS`)
- Re-running is safe: `sqlite3 whatsapp-bridge/store/messages.db < migrations/001_add_metadata_fields.sql`

### Error: "no such table: messages"

**Cause:** Database path is wrong or file doesn't exist

**Solution:**
```bash
# Check file exists
ls -la whatsapp-bridge/store/messages.db

# Check path in docker-compose.yml
grep -A5 "volumes:" docker-compose.yml | grep store
```

### Migration Ran but No Changes Visible

**Verify it actually ran:**

```bash
# Check schema generation time
sqlite3 whatsapp-bridge/store/messages.db \
  "SELECT name, sql FROM sqlite_master WHERE type='index' AND name = 'idx_messages_chat_timestamp';"
```

If this returns a row, the index exists.

**If still not visible:**

```bash
# Might be Docker volume sync issue - restart containers
docker-compose restart whatsapp-bridge

# Check again
docker exec whatsapp-bridge sqlite3 /app/whatsapp-bridge/store/messages.db \
  "SELECT count(*) FROM sqlite_master WHERE type='index' AND name LIKE 'idx_%';"
```

---

## Rollback (If Something Goes Wrong)

### Option 1: Restore from Backup

```bash
# Stop containers
docker-compose down

# Restore backup
cp whatsapp-bridge/store/messages.db.backup-<timestamp> whatsapp-bridge/store/messages.db

# Restart
docker-compose up -d
```

### Option 2: Drop New Elements

If you only want to remove specific additions:

```bash
# Remove an index (doesn't affect data)
sqlite3 whatsapp-bridge/store/messages.db "DROP INDEX idx_messages_chat_timestamp;"

# Remove a table (deletes data in that table)
sqlite3 whatsapp-bridge/store/messages.db "DROP TABLE message_edits;"
```

---

## Performance Impact

### Immediate (After Running Migration)

- ✅ Indexes created: Database queries faster (especially list_messages, list_chats)
- ✅ New columns added: Minimal storage impact (one extra integer per message = ~4 bytes × message count)
- ⚠️ First index creation: Might take 10-30 seconds on large message histories (1M+ messages)

### Long-term

- ✅ Faster queries = Lower CPU usage
- ✅ Better index coverage = Reduced memory pressure
- ⚠️ Slightly larger database file (new columns + indexes)

**Storage estimate:**
- Per 1M messages: ~5MB additional (new columns)
- Per 1K chats: ~10KB additional (denormalized counts)
- Indexes: ~50MB-100MB depending on data volume

---

## FAQ

**Q: Can I run migrations while the bridge is running?**

A: Not recommended, but possible with Docker exec. Better to stop containers first.

**Q: What if the migration fails halfway?**

A: SQLite won't partially apply statements. Either the whole migration succeeds or fails completely. Check your backup.

**Q: Do I need to rebuild the Docker image?**

A: No. Migrations run on the existing database. Only rebuild if Go/Python code changes.

**Q: Will migration affect my API responses?**

A: Yes - after migration, API responses will include new metadata fields. This is intentional (that's what Phase 1 is for).

**Q: How often will there be migrations?**

A: Only when database schema changes (new features, optimizations). Not every release.

**Q: Can I skip a migration?**

A: Not recommended. Migration 002 might depend on 001. But theoretically yes - all are idempotent.

---

## Running Migrations in Production

**Recommended sequence:**

1. Stop all services (`docker-compose down`)
2. Backup database (`cp store/messages.db store/messages.db.backup`)
3. Run migration
4. Verify with queries above
5. Start services (`docker-compose up -d`)
6. Wait for bridge to reconnect to WhatsApp (30-60s)
7. Test API endpoints
8. Keep backup for 7 days before deletion

---

## Next Steps

After running migration:

1. ✅ Restart bridge: `docker-compose restart whatsapp-bridge`
2. ✅ Check connection: `docker-compose logs whatsapp-bridge | grep "Connected"`
3. ✅ Test API: `curl http://localhost:8180/api/chats` (should work faster now)
4. ✅ Test MCP: Use Claude Code or Cursor with the WhatsApp MCP tools
5. ✅ Verify new fields: Check if responses include `sender_contact_info`, `media_count_by_type`, etc.

---

## Questions?

See [PHASE_1_PLAN.md](./PHASE_1_PLAN.md) for detailed schema information or check logs:

```bash
docker-compose logs whatsapp-bridge --tail=50
```
