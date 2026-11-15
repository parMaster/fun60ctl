---
worth: yes
added: 2026-11-15
---
# Windows build is untested, and there's no PowerToys hotkey setup

The README gives a cross-compile command, but nobody has built or run a Windows binary yet:

```
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="zig cc -target x86_64-windows-gnu" go build -o fun60ctl.exe .
```

go-hid uses cgo and bundles hidapi, so the build needs a C cross-compiler (zig or mingw).

To do:

- Build the `.exe`, then run `fun60ctl -v info` and `fun60ctl actuation 2` on Windows, wired and over the dongle.
- Watch the feature-report buffer. The code sends 65 bytes (report ID 0 + 64). Windows hidapi wants exactly the report length plus one, which should match, but check.
- Check how Windows enumerates the interfaces. Usage page `0xFFFF` and usage 2 should still identify the vendor config interface.
- Set up PowerToys Keyboard Manager: map ctrl+alt+1..4 to "Run program" `fun60ctl.exe actuation 1..4`. Document it in the README.
