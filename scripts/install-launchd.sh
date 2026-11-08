#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
state="$HOME/Library/Application Support/box-ssh-bridge"
bin="$HOME/.local/bin/box-ssh-bridge"
plist="$HOME/Library/LaunchAgents/com.zengjunzhe.box-ssh-bridge.plist"
label="com.zengjunzhe.box-ssh-bridge"
node_bin=$(command -v node || true)
box_cli=$(command -v simular-box || true)
users_file="$state/users.json"

cd "$root"

if [ -z "$node_bin" ]; then
  echo "node is required" >&2
  exit 1
fi
if [ -z "$box_cli" ]; then
  box_cli=$(find "$HOME/.npm/_npx" -path '*/node_modules/@simular-ai/box/bin/simular-box.js' -print -quit 2>/dev/null || true)
fi
if [ -z "$box_cli" ]; then
  echo "install the Box CLI first: npx @simular-ai/box login" >&2
  exit 1
fi

mkdir -p "$state" "$HOME/.local/bin" "$HOME/Library/LaunchAgents" "$state/logs"

if [ ! -f "$state/client_ed25519" ]; then
  ssh-keygen -q -t ed25519 -N '' -C box-ssh-bridge \
    -f "$state/client_ed25519"
  cp "$state/client_ed25519.pub" "$state/authorized_keys"
fi

go build -trimpath -o "$bin" "$root/cmd/box-ssh-bridge"
if [ ! -f "$users_file" ]; then
  printf '{\n  "user": "main"\n}\n' > "$users_file"
fi

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key>
  <array>
    <string>$bin</string>
    <string>--listen</string><string>127.0.0.1:22220</string>
    <string>--users-file</string><string>$users_file</string>
    <string>--state-dir</string><string>$state</string>
    <string>--authorized-key</string><string>$state/authorized_keys</string>
    <string>--box-cli</string><string>$box_cli</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$state/logs/stdout.log</string>
  <key>StandardErrorPath</key><string>$state/logs/stderr.log</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>$(dirname "$node_bin"):/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
  </dict>
  <key>ThrottleInterval</key><integer>5</integer>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
launchctl kickstart -k "gui/$(id -u)/$label"
echo "installed $label"
echo "client key: $state/client_ed25519"
echo "users: $users_file"
