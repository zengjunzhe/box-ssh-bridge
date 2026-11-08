#!/bin/sh
set -eu

machine="${INSTABOX_MACHINE:-m_agqycbnqlpzn3q7b7nmiz2xtzm}"
root="$HOME/.local/share/box-instabox-forward"
plist="$HOME/Library/LaunchAgents/com.zengjunzhe.box-instabox-forward.plist"
label="com.zengjunzhe.box-instabox-forward"
npx_bin=$(command -v npx)
node_dir=$(dirname "$(command -v node)")

mkdir -p "$root/logs" "$HOME/Library/LaunchAgents"

cat > "$root/forward.sh" <<EOF
#!/bin/sh
set -eu
while :; do
  "$npx_bin" --yes instabox@latest port-forward "$machine" 22223:22221 ||
    sleep 5
done
EOF
chmod 0755 "$root/forward.sh"

cat > "$plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>$label</string>
  <key>ProgramArguments</key>
  <array><string>$root/forward.sh</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>$root/logs/stdout.log</string>
  <key>StandardErrorPath</key><string>$root/logs/stderr.log</string>
  <key>EnvironmentVariables</key>
  <dict>
    <key>PATH</key>
    <string>$node_dir:/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin</string>
  </dict>
  <key>ThrottleInterval</key><integer>5</integer>
</dict>
</plist>
EOF

launchctl bootout "gui/$(id -u)/$label" 2>/dev/null || true
launchctl bootstrap "gui/$(id -u)" "$plist"
launchctl kickstart -k "gui/$(id -u)/$label"
echo "installed $label"
