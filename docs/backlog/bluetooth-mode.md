---
worth: yes
where: main.go:112
added: 2026-11-15
---
# Nobody has tried `fun60ctl` in Bluetooth mode

The board has a third connection mode, Bluetooth, and it's untested. Wired works fully. The 2.4 GHz dongle works for switches only (see `dongle-reads.md`).

Things to find out:

1. **Does the vendor interface exist over BLE?** With the board paired over Bluetooth, list every HID device, not only VID `0x3151`. `openBoard` currently enumerates `0x3151` only. Over BLE, the board may report a different VID/PID, or a Bluetooth vendor-ID source. Look for a device with `BusType` Bluetooth and usage page `0xFFFF`.
2. **If it exists, try a switch.** Send `04 <layer>` as a feature report, the same as wired, and check for the blink, the feel and the vendor app.
3. **If it doesn't exist, check how the vendor driver does it.** ry5088-flasher's `docs/cloud-api.md` mentions `upgradeOTAGATT`, which streams firmware over a custom BLE GATT service. Config commands may use the same GATT path instead of HID. On macOS, a custom GATT service needs CoreBluetooth, which is a much bigger change than hidapi. Use the vendor web app's WebHID capture snippet (in `dongle-reads.md`) to see if the app reaches the board over BLE at all.

Worth it because a Bluetooth user currently has no hotkey path at all.
