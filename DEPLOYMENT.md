# Deployment Field Notes

This document records the deployment that was actually exercised against
InstaBox and Simular Box. It is intentionally more concrete than the README:
commands, paths, state transitions, and failure modes.

## Current topology

```text
ShellRS
   |
   | SSH / SFTP 127.0.0.1:22223
   v
instabox port-forward       (laptop; the only local bridge-related service)
   |
   v
InstaBox box-ssh-bridge     (systemd, 127.0.0.1:22221)
   |
   +-- user@...              -> Simular account A / box main
   +-- zengjunzhedsb@...     -> Simular account B / box main
   |
   | box rsh
   v
Simular Box /workspace
```

The bridge process is on InstaBox. The laptop runs ShellRS and the official
InstaBox port-forward client; it does not run the bridge.

## InstaBox installation

The tested machine is Debian 13, systemd, 4 vCPU / 8 GiB, with `/data` as the
persistent disk. The bridge binary lives at:

```text
/data/home/box-ssh-bridge/box-ssh-bridge
```

The service is:

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
  --users-file /data/home/box-ssh-bridge/users.json \
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

The service listens only on loopback. InstaBox's `port-forward` is the only
network path into it.

## Multiple Simular accounts

Each SIMULAR account needs its own Box CLI home. The InstaBox route file:

```json
{
  "user": {
    "box": "main",
    "boxHome": "/data/home/.simular-box"
  },
  "zengjunzhedsb": {
    "box": "main",
    "boxHome": "/data/home/.simular-box-zengjunzhedsb"
  }
}
```

was tested with both accounts. The bridge sets `BOX_HOME` only for the child
`box rsh` process selected by the SSH username; credentials are not mixed
between sessions.

## SFTP

Cloud Box runs a loopback-only OpenSSH daemon on `127.0.0.1:22222`. The
InstaBox `sftp-command.sh` launches the Cloud Box SFTP subsystem through
`box rsh`; the same SSH hostname and client key serve terminal and SFTP.

Tested result:

```text
sftp> pwd
Remote working directory: /home/user
sftp> ls /workspace
/workspace/flows
```

Do not bind the SFTP daemon to `0.0.0.0`. The loopback-only design is part of
the security boundary.

## Auto-pause and wake behavior

Two independent mechanisms are involved.

### InstaBox

- Free-plan idle pause: fixed at 30 minutes.
- Current machine maximum run: 240 minutes.
- A self-paused machine wakes on SSH, `port-forward`, file, or public-port
  access.
- Memory and processes are preserved while paused.
- SSH wake waits up to about 15 seconds; if resume is still running, retry.
- Maximum-run pauses have a cooldown; repeated free-plan max-run pauses can
  produce long cooldowns.

The laptop forwarding wrapper retries `instabox port-forward` every five
seconds:

```sh
while :; do
  npx --yes instabox@latest port-forward "<machine-id>" 22223:22221 ||
    sleep 5
done
```

The forward itself wakes an idle-paused InstaBox.

### Simular Box

Simular Box sleeps independently. A `box rsh` invocation wakes it, normally
quickly. Files and `/workspace` persist. The terminal and SFTP sessions are
re-established after wake; they are not long-lived through the sleep.

## Failure modes seen in practice

### `box agent unreachable`

The Simular control plane can report `running` while the in-box agent is
unresponsive. In that case:

1. Check the live desktop or built-in terminal.
2. Restart only the guest agent from the Box terminal:

   ```bash
   sudo systemctl restart cocoonbox-agent.service
   ```

3. Re-run `box rsh`.

Do not reset the Box or delete it; it may destroy files.

### SSH host key changes

Two different bridges using the same `127.0.0.1:port` produce host-key
conflicts. Give each service a distinct port or deliberately distribute one
stable host key. Update the client's known-host entry only after verifying the
new service.

### launchd PATH

`npx` uses `env node`. A launchd job with a minimal PATH fails with:

```text
env: node: No such file or directory
```

Include the Node directory in the launchd job's `PATH`.

### Port already in use

When replacing a local adapter, stop its old launchd job before starting the
new one. `launchctl kickstart` does not kill a service owned by a different
label.

### Box CLI cold start

When the bridge uses `npx` instead of a resolved JS entrypoint, pass the
package explicitly:

```text
--box-cli /usr/local/bin/npx --box-package @simular-ai/box
```

## Verification checklist

```bash
# InstaBox service
ssh -o BatchMode=yes ins \
  'sudo systemctl is-active box-ssh-bridge'

# InstaBox routes
ssh -o BatchMode=yes ins \
  'cat /data/home/box-ssh-bridge/users.json'

# Forward is listening locally
lsof -nP -iTCP:22223 -sTCP:LISTEN

# Old and new account routes
ssh -i ~/.ssh/instabox-client_ed25519 -p 22223 \
  user@127.0.0.1 'whoami; pwd; hostname'
ssh -i ~/.ssh/instabox-client_ed25519 -p 22223 \
  zengjunzhedsb@127.0.0.1 'whoami; pwd; hostname'

# SFTP
printf 'pwd\nls /workspace\n' | sftp -b - \
  -i ~/.ssh/instabox-client_ed25519 -P 22223 \
  -o BatchMode=yes zengjunzhedsb@127.0.0.1
```

## Operational guidance

- Keep the InstaBox bridge on loopback.
- Keep the Cloud Box SFTP daemon on loopback.
- Do not copy OAuth device credentials into a repository or issue.
- Rotate any pasted `cb_...` key and revoke it in Box Keys.
- Treat the free-plan 240-minute run limit as a hard boundary.
- Use a paid plan's `auto-suspend ... 0` only if you accept continuous runtime
  billing.
- Use an activity lease for long silent operations instead of keeping a
  terminal attached.
