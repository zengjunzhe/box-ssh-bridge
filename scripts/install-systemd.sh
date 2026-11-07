#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
install_dir="${BOX_SSH_BRIDGE_DIR:-$HOME/box-ssh-bridge}"
listen="${BOX_SSH_BRIDGE_LISTEN:-127.0.0.1:22221}"
box="${BOX_SSH_BRIDGE_BOX:-main}"
user="${BOX_SSH_BRIDGE_USER:-user}"

mkdir -p "$install_dir/state"

go build -trimpath -o "$install_dir/box-ssh-bridge" "$root/cmd/box-ssh-bridge"

if [ ! -f "$install_dir/user_ed25519" ]; then
  ssh-keygen -q -t ed25519 -N '' -C box-ssh-bridge -f "$install_dir/user_ed25519"
  cp "$install_dir/user_ed25519.pub" "$install_dir/authorized_keys"
fi
chmod 600 "$install_dir/user_ed25519"
chmod 644 "$install_dir/authorized_keys"

cat > "$install_dir/sftp-command.sh" <<EOF
#!/bin/sh
exec /usr/local/bin/npx --yes @simular-ai/box rsh --box $box -- \\
  ssh -T -s -p 22222 \\
  -i /home/user/.ssh/box_bridge_client_ed25519 \\
  -o StrictHostKeyChecking=no -o BatchMode=yes \\
  user@127.0.0.1 sftp
EOF
chmod 0755 "$install_dir/sftp-command.sh"

unit=/etc/systemd/system/box-ssh-bridge.service
sudo tee "$unit" >/dev/null <<EOF
[Unit]
Description=box-ssh-bridge
After=network-online.target
Wants=network-online.target

[Service]
User=$user
Group=$user
Environment=HOME=$HOME
Environment=PATH=/usr/local/bin:/usr/bin:/bin
WorkingDirectory=$install_dir
ExecStart=$install_dir/box-ssh-bridge --listen $listen --box $box --user $user --state-dir $install_dir/state --authorized-key $install_dir/authorized_keys --box-cli /usr/local/bin/npx --box-package @simular-ai/box --sftp-command $install_dir/sftp-command.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now box-ssh-bridge.service
sudo systemctl status box-ssh-bridge.service --no-pager
echo
echo "client key: $install_dir/user_ed25519"
echo "listen: $listen"
