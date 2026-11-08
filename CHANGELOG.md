# Changelog

## Unreleased

- Document the cloud-first InstaBox deployment and field-tested failure modes.
- Add retry behavior to the InstaBox port-forward wrapper.

## v0.3.0

- Add multi-Box routing by SSH username.
- Add per-route `BOX_HOME` support for multiple Simular accounts.
- Keep the original single-Box `--box`/`--user` mode.

## v0.2.0

- Add SFTP subsystem bridging with `--sftp-command`.
- Add InstaBox port-forward helper.

## v0.1.0

- Initial local SSH bridge.
- PTY, `exec`, public-key and password authentication.
- macOS launchd installer, Linux build, and three-platform CI.
