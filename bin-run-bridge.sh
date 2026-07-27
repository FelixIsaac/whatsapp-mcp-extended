#!/usr/bin/env bash
# Author: Fabian Bitter (fabian@bitter.de)
# Launch the WhatsApp bridge with its .env, for the launchd KeepAlive wrapper.
set -euo pipefail
DIR="/Volumes/OWC Express/Projekte/Softwarenetwicklung/Repositories/GitHub/whatsapp-mcp-extended"
cd "$DIR/whatsapp-bridge"
set -a; . "$DIR/.env"; set +a
exec "$DIR/whatsapp-bridge/whatsapp-bridge"
