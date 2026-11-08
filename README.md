# box-ssh-bridge: SSH and SFTP for Simular Box, ShellRS, and InstaBox

[![CI](https://github.com/zengjunzhe/box-ssh-bridge/actions/workflows/ci.yml/badge.svg)](https://github.com/zengjunzhe/box-ssh-bridge/actions/workflows/ci.yml)

`box-ssh-bridge` lets a normal SSH client use a
[Simular Box](https://box.simular.ai) cloud computer even though Box exposes
the official `box rsh` command API rather than a public SSH daemon.

It is a local SSH gateway written in Go. ShellRS, OpenSSH, SFTP clients, IDEs,
and automation tools can connect to a normal `host:port` while the bridge runs
`@simular-ai/box rsh` underneath.

This project is also useful if you are looking for:

- Simular Box SSH access
- `box rsh` SSH bridge or adapter
- `box rsh` ProxyCommand alternative
- ShellRS connection to Simular Box
- Simular Box SFTP or `/workspace` file access
- InstaBox to Simular Box SSH tunnel
- SSH gateway for an AI cloud desktop

> This is for **box by Simular**, not Box.com file storage.

> 中文简介：把 Simular Box 的 `box rsh` 封装成标准 SSH/SFTP 入口，
> 支持 ShellRS，也可以部署在本机或 InstaBox 云主机上。适用于寻找
> “Simular Box SSH”“box rsh 转 SSH”“ShellRS 连接 Simular Box”、
> “Simular Box 文件传输”和“InstaBox SSH 隧道”的场景。

```text
SSH / SFTP client
        |
        v
box-ssh-bridge
        |
        +-- terminal / exec --> box rsh --> Simular Box
        |
        +-- SFTP subsystem --> loopback-only SFTP --> /workspace
```

## Features

- Standard SSH password or public-key authentication.
- Interactive PTY sessions with terminal resizing.
- Non-interactive `ssh host command`, with the remote exit code preserved.
- Optional SSH `sftp` subsystem for the same SSH host.
- Dedicated, loopback-only host key and client key.
- No fixed dependency on a control-plane vendor.
- Small Go binary; the only runtime dependency is the official
  `@simular-ai/box` CLI.
- macOS `launchd`, Linux `systemd`, and InstaBox deployment examples.

## Requirements

- Go 1.24+ to build, or use a
  [prebuilt release](https://github.com/zengjunzhe/box-ssh-bridge/releases).
- Node.js 18.17+ and the official Box CLI:

  ```bash
  npx @simular-ai/box login
  npx @simular-ai/box rsh --box main -- uname -a
  ```

If the second command does not work, the bridge cannot work. Fix the Box login
or Box network path first.

## Quick start: local deployment

```bash
git clone https://github.com/zengjunzhe/box-ssh-bridge.git
cd box-ssh-bridge
go build -o box-ssh-bridge ./cmd/box-ssh-bridge
```

Generate a dedicated client key:

```bash
mkdir -p "$HOME/Library/Application Support/box-ssh-bridge"
ssh-keygen -q -t ed25519 -N '' -C box-ssh-bridge \
  -f "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519"
cp "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519.pub" \
  "$HOME/Library/Application Support/box-ssh-bridge/authorized_keys"
```

Start the bridge:

```bash
./box-ssh-bridge \
  --listen 127.0.0.1:22220 \
  --box main \
  --authorized-key "$HOME/Library/Application Support/box-ssh-bridge/authorized_keys"
```

Connect:

```bash
ssh -i "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519" \
  -p 22220 user@127.0.0.1
```

### Password authentication

Without `--authorized-key`, the bridge generates a random password and prints
it once:

```bash
./box-ssh-bridge --password "$(openssl rand -base64 24)"
```

## Local macOS background service

The included launchd installer creates a dedicated key and keeps the bridge
running across logins:

```bash
./scripts/install-launchd.sh
```

Test:

```bash
ssh -i "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519" \
  -p 22220 user@127.0.0.1 'whoami; pwd; hostname'
```

Stop:

```bash
launchctl bootout "gui/$(id -u)/com.zengjunzhe.box-ssh-bridge"
```

## ShellRS

Create a normal SSH host:

| Field | Value |
| --- | --- |
| Name | `Simular Box main` |
| Address | `127.0.0.1` |
| Port | `22220` |
| User | `user` |
| Authentication | Private key `client_ed25519` |

Open the host. For SFTP, enable the Files sidebar after the host is connected.
The SFTP path is the path in the Simular Box, normally `/workspace` or
`/home/user`.

## Multiple Boxes

One bridge process can route several Simular Boxes. Create a `users.json`
map from SSH usernames to Box names or IDs:

```json
{
  "work": "box-work",
  "personal": "box-personal",
  "main": "main"
}
```

If the Boxes belong to different Simular accounts, use an object per user and
give each account its own Box CLI home:

```json
{
  "work": {
    "box": "main",
    "boxHome": "/Users/me/.simular-box-work"
  },
  "personal": {
    "box": "main",
    "boxHome": "/Users/me/.simular-box-personal"
  }
}
```

The bridge sets `BOX_HOME` for that session only. Log in once with the same
value:

```bash
BOX_HOME="$HOME/.simular-box-work" npx @simular-ai/box login
```

Start with:

```bash
./box-ssh-bridge \
  --listen 127.0.0.1:22220 \
  --users-file "$HOME/Library/Application Support/box-ssh-bridge/users.json" \
  --authorized-key "$HOME/Library/Application Support/box-ssh-bridge/authorized_keys"
```

Each SSH username selects its own Box:

```bash
ssh -i box-bridge-key -p 22220 work@127.0.0.1
ssh -i box-bridge-key -p 22220 personal@127.0.0.1
```

In ShellRS, create one host per Box, using the same address and port but a
different username. For SFTP, include `{box}` in `--sftp-command`; it is
replaced per SSH session:

```text
--sftp-command "/path/to/box-sftp.sh {box}"
```

Without `--users-file`, the original single-Box `--box` and `--user` behavior
is unchanged.

## Cloud deployment: InstaBox as the SSH gateway

The bridge can run on an InstaBox cloud machine and reach Simular Box from
there. The tested topology is:

```text
ShellRS / OpenSSH / SFTP
        |
        | InstaBox port-forward (official CLI)
        v
InstaBox: box-ssh-bridge
        |
        | box rsh
        v
Simular Box /workspace
```

### 1. Deploy the bridge on InstaBox

Build a Linux binary and copy it to the machine:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -o /tmp/box-ssh-bridge ./cmd/box-ssh-bridge

scp /tmp/box-ssh-bridge ins:/tmp/box-ssh-bridge
ssh ins 'mkdir -p ~/box-ssh-bridge && \
  install -m 0755 /tmp/box-ssh-bridge ~/box-ssh-bridge/box-ssh-bridge'
```

Log the Box CLI in **on InstaBox**. Its credential is separate from the
credential on your laptop:

```bash
ssh -t ins 'npx --yes @simular-ai/box login'
ssh ins 'npx --yes @simular-ai/box rsh --box main -- uname -a'
```

Create a client key on InstaBox:

```bash
ssh ins '
  cd ~/box-ssh-bridge
  test -f user_ed25519 || ssh-keygen -q -t ed25519 -N "" -C box-ssh-bridge -f user_ed25519
  cp user_ed25519.pub authorized_keys
  chmod 600 user_ed25519
'
```

Run as a systemd service:

```ini
[Unit]
Description=box-ssh-bridge
After=network-online.target
Wants=network-online.target

[Service]
User=user
Group=user
Environment=HOME=/data/home
Environment=PATH=/usr/local/bin:/usr/bin:/bin
WorkingDirectory=/data/home/box-ssh-bridge
ExecStart=/data/home/box-ssh-bridge/box-ssh-bridge \
  --listen 127.0.0.1:22221 \
  --box main \
  --user user \
  --state-dir /data/home/box-ssh-bridge/state \
  --authorized-key /data/home/box-ssh-bridge/authorized_keys \
  --box-cli /usr/local/bin/npx \
  --box-package @simular-ai/box \
  --sftp-command /data/home/box-ssh-bridge/sftp-command.sh
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Keep the bridge on `127.0.0.1`; do not expose it directly to the internet.

### 2. Expose it only through InstaBox

InstaBox's server currently does not enable raw public TCP ports. Use its
official private forwarding:

```bash
instabox port-forward <machine-id> 22223:22221
```

Then the SSH endpoint on your laptop is:

```text
127.0.0.1:22223
```

Test:

```bash
ssh -i instabox-client_ed25519 -p 22223 \
  user@127.0.0.1 'whoami; pwd; hostname'
```

[`scripts/install-instabox-forward.sh`](scripts/install-instabox-forward.sh)
turns this into a macOS launchd service.

### 3. Enable SFTP on InstaBox and Cloud Box

Cloud Box needs a loopback-only OpenSSH server because the bridge only has the
`box rsh` command channel, not arbitrary TCP. This is the one case where the
project installs a package inside the Box:

```bash
# Run inside Simular Box, as the normal user
sudo apt-get update
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y openssh-server
```

Use a dedicated host key and a dedicated client key, and bind only to
`127.0.0.1`. A minimal `sshd_config` is:

```text
Port 22222
ListenAddress 127.0.0.1
HostKey /home/user/.ssh/box_bridge_host_ed25519
AuthorizedKeysFile /home/user/.ssh/box_bridge_authorized_keys
PasswordAuthentication no
PubkeyAuthentication yes
UsePAM no
StrictModes no
Subsystem sftp internal-sftp
PidFile /home/user/.ssh/box_bridge_sshd.pid
```

Start it as a user service, not a public service:

```bash
systemctl --user enable --now box-bridge-sftp
```

Verify from inside the Box:

```bash
printf 'pwd\nls /workspace\n' | sftp -b - \
  -i ~/.ssh/box_bridge_client_ed25519 \
  -P 22222 -o StrictHostKeyChecking=no user@127.0.0.1
```

On InstaBox, create the SFTP wrapper:

```sh
#!/bin/sh
exec /usr/local/bin/npx --yes @simular-ai/box rsh --box main -- \
  ssh -T -s -p 22222 \
  -i /home/user/.ssh/box_bridge_client_ed25519 \
  -o StrictHostKeyChecking=no -o BatchMode=yes \
  user@127.0.0.1 sftp
```

Then pass it with `--sftp-command`. The same SSH host used for the terminal
now also serves the SFTP subsystem. The file tree belongs to Simular Box, not
InstaBox.

## CLI

```text
box-ssh-bridge [flags]

  --listen string          local SSH listen address (default "127.0.0.1:22220")
  --box string             Simular Box name or ID (default "main")
  --user string            SSH username (default "user")
  --users-file string      JSON map of SSH usernames to Box names or IDs
  --password string        SSH password
  --authorized-key string  authorized_keys file
  --host-key string        SSH host key path
  --box-cli string         path to the simular-box CLI entrypoint
  --box-package string     package used when --box-cli is npx
  --sftp-command string    local command that serves the SFTP subsystem
  --state-dir string       state directory
  --version                print version
```

State defaults to the OS user configuration directory under `box-ssh-bridge`;
override it with `--state-dir` or `BOX_SSH_BRIDGE_HOME`.

## Security notes

- Keep the bridge listener on loopback.
- Use public-key authentication for unattended deployments.
- Keep Box credentials on the machine that runs `box rsh`; never commit them.
- Keep the Cloud Box SFTP daemon on `127.0.0.1`.
- Do not publish the bridge as a raw TCP port without an additional access
  layer.

## Limitations

- The bridge is an SSH gateway to `box rsh`; it is not a full hidden SSH daemon
  inside Box.
- X11 forwarding, agent forwarding, and TCP port forwarding are not
  implemented.
- SFTP requires the optional loopback-only SFTP backend described above.
- Multiple Boxes can share one listener; each needs its own SSH username and
  `users.json` entry.
- A shell session starts `bash -li` inside `/workspace`.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/box-ssh-bridge
```

The integration boundary is tested with an SSH client against a fake Box CLI in
`internal/bridge`.

## License

MIT
