# box-ssh-bridge

Use a normal SSH client with [Simular Box](https://box.simular.ai) even though
Box currently exposes `box rsh` rather than an SSH daemon.

`box-ssh-bridge` is a small local SSH server. Each SSH session is forwarded to
`npx @simular-ai/box rsh` and opens an interactive Bash login shell in the
selected Box. It was built for clients such as ShellRS, but any OpenSSH client
can use it.

```text
SSH client -> localhost:22220 -> box-ssh-bridge -> box rsh -> Simular Box
```

## Features

- Standard SSH password or public-key authentication.
- Interactive PTY sessions with window resizing.
- Non-interactive `ssh host command` executions and correct exit codes.
- Public-key authentication with a dedicated, local-only host.
- No changes or packages installed inside the Box.
- Small Go implementation with no runtime dependencies other than the official
  `@simular-ai/box` CLI.

## Requirements

- Go 1.24+ to build.
- Node.js 18.17+ and the official Box CLI:

  ```bash
  npx @simular-ai/box login
  npx @simular-ai/box rsh --box main -- uname -a
  ```

The second command matters: if it does not work, `box-ssh-bridge` cannot work.

## Install from source

```bash
git clone https://github.com/zengjunzhe/box-ssh-bridge.git
cd box-ssh-bridge
go build -o box-ssh-bridge ./cmd/box-ssh-bridge
```

Prebuilt macOS, Linux, and Windows binaries are on the
[Releases](https://github.com/zengjunzhe/box-ssh-bridge/releases) page.

### Public-key authentication

Generate a dedicated client key and authorize it:

```bash
mkdir -p "$HOME/Library/Application Support/box-ssh-bridge"
ssh-keygen -q -t ed25519 -N '' -C box-ssh-bridge \
  -f "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519"
cp "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519.pub" \
  "$HOME/Library/Application Support/box-ssh-bridge/authorized_keys"
./box-ssh-bridge --authorized-key "$HOME/Library/Application Support/box-ssh-bridge/authorized_keys"
```

Connect:

```bash
ssh -i "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519" \
  -p 22220 user@127.0.0.1
```

### Password authentication

Without `--authorized-key`, the bridge generates a random password and prints
it once at startup:

```bash
./box-ssh-bridge --password "$(openssl rand -base64 24)"
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

Then open the host. The bridge is only reachable on loopback by default.

## macOS background service

The repository includes a launchd installer that uses a dedicated client key
and keeps the bridge running:

```bash
./scripts/install-launchd.sh
```

Test it:

```bash
ssh -i "$HOME/Library/Application Support/box-ssh-bridge/client_ed25519" \
  -p 22220 user@127.0.0.1 'whoami; pwd; hostname'
```

Stop or remove it:

```bash
launchctl bootout "gui/$(id -u)/com.zengjunzhe.box-ssh-bridge"
```

## CLI

```text
box-ssh-bridge [flags]

  --listen string          local SSH listen address (default "127.0.0.1:22220")
  --box string             Simular Box name or ID (default "main")
  --user string            SSH username (default "user")
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

## Limitations

This is an SSH gateway to the `box rsh` execution API, not a replacement for
`sshd` inside Box. Consequently:

- Interactive shells and `exec` requests work.
- SFTP works when `--sftp-command` is configured.
- X11 forwarding, agent forwarding, and TCP port forwarding are not implemented.
- A shell session starts `bash -li` by default inside `/workspace`.
- Keep the listener on loopback unless you intentionally add access controls.

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/box-ssh-bridge
```

The integration boundary is tested with an SSH client against a fake Box CLI in
`internal/bridge`.

## SFTP

The bridge can expose SSH's `sftp` subsystem without teaching SFTP about
`box rsh`. Point `--sftp-command` at a local wrapper that connects to a
loopback-only SFTP server reachable from the machine running the bridge.

For a Simular Box with a loopback-only `sshd` and a dedicated key:

```sh
#!/bin/sh
exec /usr/local/bin/npx --yes @simular-ai/box rsh --box main -- \
  ssh -T -s -p 22222 \
  -i /home/user/.ssh/box_bridge_client_ed25519 \
  -o StrictHostKeyChecking=no -o BatchMode=yes \
  user@127.0.0.1 sftp
```

Start the bridge with `--sftp-command /path/to/that-wrapper`. The same SSH
host now supports terminal sessions and SFTP; SFTP paths are the paths inside
the Box.

## License

MIT
