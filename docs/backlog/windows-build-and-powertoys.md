---
worth: yes
added: 2026-11-15
---
# Windows: dongle untried, PowerToys setup unconfirmed

Over the cable, the Windows binary works: `info`, `actuation` and `light`. `openBoard` picks the `MI_02` vendor interface (page `0xFFFF`, usage 2), and the 65-byte feature reports go through. The README now has a PowerToys section.

Left to do on Windows:

- Run `fun60ctl -v -dongle info`, `actuation 2` and `light 1` over the 2.4 GHz dongle. Check the `FC` fetch works there too.
- Set up the PowerToys shortcuts as the README describes and confirm the hidden window and full path work. Then add Windows to the hardware results table.
