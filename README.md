# fun60ctl

Switch the actuation layer (the 4 sets of magnetic-switch settings) on a MonsGeek FUN60 Pro from the command line.
The stock firmware dropped the Fn+F/G/H/J shortcut, so this does the same switch over USB in about 160 ms.
A hotkey tool like Karabiner or PowerToys can call it.

It talks to the stock firmware. Nothing gets flashed.

## Commands

```
fun60ctl info               # VID:PID, wired/dongle, dev_id, firmware
fun60ctl actuation          # print the active actuation layer (1-4)
fun60ctl actuation <1-4>    # switch to that layer
fun60ctl actuation next     # switch to the next layer, wrapping 4 -> 1
```

Flags: `-v` prints HID interfaces, raw reports and timings. `-dongle` forces the 2.4 GHz dongle. `-force` skips the FUN60 Pro `dev_id` check.

## Install

With Go installed:

```
go install github.com/parMaster/fun60ctl@latest
```

The binary lands in `$(go env GOPATH)/bin`, usually `~/go/bin`.
go-hid uses cgo, so this needs a C compiler. On macOS, run `xcode-select --install` once if you don't have one.

Without Go, download a binary from [Releases](https://github.com/parMaster/fun60ctl/releases): macOS arm64 and amd64, Windows amd64.
The macOS binaries are unsigned, so Gatekeeper blocks the first run. Clear the quarantine flag and make the file executable:

```
xattr -d com.apple.quarantine fun60ctl-darwin-arm64
chmod +x fun60ctl-darwin-arm64
```

The Windows binary is untested.

## Hotkeys with Karabiner

This [Karabiner-Elements](https://karabiner-elements.pqrs.org/) rule maps ctrl+opt+1..4 to the four layers.
Karabiner runs `shell_command` with a minimal PATH, so use the binary's full path.
Add it under `complex_modifications.rules` in `~/.config/karabiner/karabiner.json`:

```json
{
  "description": "Ctrl + Opt + {1, 2, 3, 4} = FUN60 actuation layer 1-4",
  "manipulators": [
    { "type": "basic", "from": { "key_code": "1", "modifiers": { "mandatory": ["control", "option"] } }, "to": [{ "shell_command": "/Users/<you>/go/bin/fun60ctl actuation 1" }] },
    { "type": "basic", "from": { "key_code": "2", "modifiers": { "mandatory": ["control", "option"] } }, "to": [{ "shell_command": "/Users/<you>/go/bin/fun60ctl actuation 2" }] },
    { "type": "basic", "from": { "key_code": "3", "modifiers": { "mandatory": ["control", "option"] } }, "to": [{ "shell_command": "/Users/<you>/go/bin/fun60ctl actuation 3" }] },
    { "type": "basic", "from": { "key_code": "4", "modifiers": { "mandatory": ["control", "option"] } }, "to": [{ "shell_command": "/Users/<you>/go/bin/fun60ctl actuation 4" }] }
  ]
}
```

## Build from source

macOS (go-hid bundles hidapi, so nothing else is needed):

```
go build -o fun60ctl .
```

Windows, cross-compiled from macOS. go-hid uses cgo, so it needs a C cross-compiler, for example zig:

```
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC="zig cc -target x86_64-windows-gnu" go build -o fun60ctl.exe .
```

## How it works

```
enumerate HID (VID 0x3151)
  → PID 0x5030 (wired), usage page 0xFFFF, usage 2
  → GET_INFOR 0x8F → dev_id must be a FUN60 Pro (2304, 2305, 2464, 2600, 2785)
  → SET_PROFILE 0x04 <layer 0-3>
  → GET_PROFILE 0x84 → confirm the new layer
```

Each report is 64 bytes, sent as a HID feature report with report ID 0.
Bytes 0..7 must sum to 0xFF, and the checksum goes in byte 7.
The board needs about 50 ms before its reply is ready.

## Hardware results

Tested on a FUN60 Pro, `dev_id` 2785, firmware v407, on macOS.

| Test | Wired | 2.4 GHz dongle |
|---|---|---|
| `info` | works, ~60 ms | not supported (no readable reply) |
| `actuation` (read) | works, ~60 ms | not supported (no readable reply) |
| `actuation <n>` | works, ~160 ms, confirmed by read-back | works, send-only, not confirmed |
| `actuation next` | works | not supported (needs a read) |

- `SET_PROFILE` (`0x04`) is the actuation layer. Layer 2 was set to 0.55 mm in the vendor app. After `actuation 2`, keys triggered at 0.55 mm and the app showed layer 2 active.
- The backlight blinks once on a switch, the same as a switch from the app.
- The dongle shows up as PID `0x5026` ("MonsGeek 2.4G Wireless Keyboard"), not the `0x5038` in ry5088-flasher's hardware notes. It has its own vendor interface (page `0xFFFF`, usage 2).
- Writes pass through the dongle unchanged. The vendor app sends the same `04 <layer>` frame over the dongle as over the cable. The keyboard applies it, blinks, and the app shows the new layer.
- Reads don't come back the same way. The dongle answers every `GET_REPORT` with its own status frame, for example `00 64 64 00 01 01 01 7B`. The second byte (`0x64` = 100) is probably the battery level. The first byte flips to `01` some time after a command, so it may mean "reply waiting".
- The vendor app polls that status every 2 s with an `F7` frame (no checksum). Before a switch it sends `FE 40` (no checksum). It never reads a feature reply for the switch. Instead the keyboard announces the new layer with input report ID 5: `0F <layer 0-3>`.
- Reading over the dongle needs the command that fetches the keyboard's reply. Capturing the app while it loads a config would show it.

### Bluetooth isn't supported

Over Bluetooth the board shows up as `3151:5027`. Its vendor collection is on page `0xFF55` and uses report ID 6 as input and output reports, not feature reports. `fun60ctl` doesn't speak that.

On macOS there's a second problem. The vendor collection shares one HID device with the keyboard. Opening it needs the Input Monitoring permission. And when Karabiner modifies the board, it seizes that device, so no other program can open it.

## Credits

The frame format, the `GET_INFOR` layout and the FUN60 Pro `dev_id` list come from [ry5088-flasher](https://github.com/dot-agi/ry5088-flasher), which documents the RY5088 vendor protocol.

## License

MIT, see [LICENSE](LICENSE).
