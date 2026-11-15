---
worth: yes
added: 2026-11-15
---
# The Windows binary has never run, and there's no PowerToys hotkey setup

The release workflow builds `fun60ctl-windows-amd64.exe`, and it also cross-compiles locally with zig. Nobody has run it on Windows yet, so the README marks it untested.

To do on a Windows machine:

- Run `fun60ctl -v info` and `fun60ctl actuation 2`, wired and over the dongle.
- Check that `-v` lists a vendor interface with usage page `0xFFFF`, usage 2. That's how `openBoard` finds it.
- Watch the feature-report size. The code sends 65 bytes (report ID 0 + 64). Windows hidapi wants exactly the report length plus one, which should match.
- Set up PowerToys Keyboard Manager: map ctrl+alt+1..4 to "Run program" `fun60ctl.exe actuation 1..4`. Document it in the README next to the Karabiner example, and drop the "untested" notes.
