---
worth: no
where: main.go:112
added: 2026-11-15
---
# Bluetooth mode is blocked while Karabiner modifies the board

Probed on macOS 14 with the board paired over Bluetooth Low Energy. The rejected fix is adding a Bluetooth path to `openBoard`. Karabiner has to grab the board to apply remaps, and that grab locks out `fun60ctl`. The board is unusable without Karabiner, so a Bluetooth path would never be used.

## What the probe found

- The board enumerates as `3151:5027` ("FUN60 PRO-1"), bus Bluetooth. Wired is `0x5030`.
- It has a vendor collection on page `0xFF55`, usage `0x202`. Wired uses page `0xFFFF`, usage `2`, so the command protocol over Bluetooth may differ. Nobody has sent a frame to it yet.
- macOS exposes all six collections (keyboard, consumer, mouse, vendor and others) as ONE IOHIDDevice. Over the cable, the vendor interface is a separate USB interface.
- Opening it without Input Monitoring fails with `(iokit/common) not permitted`.
- With Input Monitoring granted, opening fails with `exclusive access and device already open`. Karabiner-Elements seizes every keyboard whose events it modifies, and that seizure covers the vendor collection too.

## What would reopen it

A user who runs `fun60ctl` without Karabiner modifying the board. Then the next steps are: grant Input Monitoring, dump the `0xFF55` report descriptor, and try a read-only `GetInfo` before any layer switch.
