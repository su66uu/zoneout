#!/usr/bin/env bash
set -euo pipefail

ZONEOUT_HOME="$HOME/.zoneout"
BIN_DIR="$ZONEOUT_HOME/bin"
AGENT_BIN="$BIN_DIR/zoneout-agent"
SSH_DIR="$HOME/.ssh"
SSH_CONFIG="$SSH_DIR/config"

START_MARKER="# >>> zoneout.local >>>"
END_MARKER="# <<< zoneout.local <<<"

mkdir -p "$BIN_DIR"
mkdir -p "$SSH_DIR"

if [[ -e "$AGENT_BIN" && ! -x "$AGENT_BIN" ]]; then
  echo "Refusing to overwrite non-executable file: $AGENT_BIN" >&2
  exit 1
fi

go build -o "$AGENT_BIN" ./agent
chmod +x "$AGENT_BIN"

if [[ ! -f "$SSH_CONFIG" ]]; then
  touch "$SSH_CONFIG"
  chmod 600 "$SSH_CONFIG"
fi

BACKUP="$SSH_CONFIG.zoneout.bak.$(date +%Y%m%d%H%M%S)"
cp "$SSH_CONFIG" "$BACKUP"

TMP="$(mktemp "$SSH_CONFIG.zoneout.XXXXXX")"

awk -v start="$START_MARKER" -v end="$END_MARKER" '
  $0 == start { skip=1; next }
  $0 == end { skip=0; next }
  !skip { print }
' "$SSH_CONFIG" > "$TMP"

cat >> "$TMP" <<EOF

$START_MARKER
Host zoneout.local
    HostName 127.0.0.1
    Port 23234
    RequestTTY force
    PermitLocalCommand yes
    LocalCommand $AGENT_BIN --ensure-running
    RemoteForward 127.0.0.1:27777 127.0.0.1:17777
$END_MARKER
EOF

mv "$TMP" "$SSH_CONFIG"

"$AGENT_BIN" --ensure-running

echo "Installed. Try: ssh zoneout.local"
echo "Previous SSH config backed up at: $BACKUP"
