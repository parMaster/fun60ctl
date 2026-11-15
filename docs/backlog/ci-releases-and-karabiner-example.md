---
worth: yes
added: 2026-11-15
---
# No CI, no release binaries, and no Karabiner example in the README

Today you build from source, and the README doesn't show how to bind hotkeys.

- **CI:** a GitHub Actions workflow that runs `go test ./...` and `golangci-lint run ./...`.
- **Releases:** binaries for macOS (arm64, amd64) and Windows (amd64). go-hid needs cgo, so either build each OS on its own runner or cross-compile with `zig cc`.
- **README Karabiner example:** the working rule maps ctrl+opt+1..4. `shell_command` runs with a minimal PATH, so the rule needs the binary's full path. Each of the four manipulators looks like this:

```json
{
  "from": { "key_code": "1", "modifiers": { "mandatory": ["control", "option"] } },
  "to": [{ "shell_command": "/Users/<you>/go/bin/fun60ctl actuation 1" }],
  "type": "basic"
}
```

- Mention `go install github.com/parMaster/fun60ctl@latest` as the install path.
