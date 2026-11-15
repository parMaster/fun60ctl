# fun60ctl

Switch the actuation layer (the 4 sets of magnetic-switch settings) on a MonsGeek FUN60 Pro from the command line.
The stock firmware dropped the Fn+F/G/H/J shortcut, so this does the same switch over the cable in about 160 ms, or over the 2.4 GHz dongle in about 250 ms.
It can also bring back your custom lighting layers after Fn+Enter has switched the board to the built-in lighting modes.
A hotkey tool like Karabiner or PowerToys can call it.

It talks to the stock firmware. Nothing gets flashed.

## Commands

```
fun60ctl info               # VID:PID, wired/dongle, dev_id, firmware
fun60ctl actuation          # print the active actuation layer (1-4)
fun60ctl actuation <1-4>    # switch to that layer
fun60ctl actuation next     # switch to the next layer, wrapping 4 -> 1
fun60ctl light              # print the active lighting: "custom 1-5" or "built-in mode N"
fun60ctl light <1-5>        # select that custom lighting layer
fun60ctl light next         # select the next custom layer, wrapping 5 -> 1; from a built-in mode, custom 1
```

Flags: `-v` prints HID interfaces, raw reports and timings. `-dongle` forces the 2.4 GHz dongle. `-force` skips the FUN60 Pro `dev_id` check.

The board has 5 custom lighting layers, which Fn+/ cycles. Fn+Enter switches to the built-in lighting modes, and no key switches back. `light <1-5>` does.

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

A rule for `light <1-5>` looks the same, with the command swapped.

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
  → PID 0x5030 (wired) or 0x5026 (dongle), usage page 0xFFFF, usage 2
  → GET_INFOR 0x8F → dev_id must be a FUN60 Pro (2304, 2305, 2464, 2600, 2785)
  → SET_PROFILE 0x04 <layer 0-3>
  → GET_PROFILE 0x84 → confirm the new layer
```

Each report is 64 bytes, sent as a HID feature report with report ID 0.
Bytes 0..7 must sum to 0xFF, and the checksum goes in byte 7.
The board needs about 50 ms before its reply is ready.

Over the dongle, a read takes one more step. After the command, `fun60ctl` sends `FC` (no checksum). That makes the dongle fetch the keyboard's reply over the radio, and the next `GET_REPORT` returns it about 20 ms later:

```
wired:   cmd → 50 ms → GET_REPORT = reply
dongle:  cmd → 50 ms → FC → 20 ms → GET_REPORT = reply, or zeros → repeat FC
```

### Lighting

`GET_LED` (`0x87`) returns 7 params, and `SET_LED` (`0x07`) writes them back:

```
[mode] [speed] [brightness] [options] [R] [G] [B]
```

`SET_LED` fills bytes 1..7, so its checksum goes in byte 8: bytes 0..8 sum to 0xFF.

| State | Reply after `87` |
|---|---|
| custom layer 1 | `0D 04 04 00 00 C8 C8` |
| custom layer 2 | `0D 04 04 10 00 C8 C8` |
| built-in "Always on" | `01 02 04 06 FA FF FA` |
| built-in "Stars twinkle" | `06 02 04 07 FA FF FA` |

- Mode `0x0D` shows a custom layer, and the high nibble of options is the layer, 0-based. Other modes are built-in effects.
- In a built-in mode, the reply shows that mode's speed, options and color.
- `light <n>` reads the params, sets the mode to `0D` and the layer nibble to `n−1`, and writes the rest back unchanged. A custom layer keeps its own look, even when the other bytes came from a built-in mode. Over the dongle, the write gets no reply, so `fun60ctl` confirms it the same way as an actuation switch.

## Hardware results

Tested on a FUN60 Pro, `dev_id` 2785, firmware v407, on macOS.

| Test | Wired | 2.4 GHz dongle |
|---|---|---|
| `info` | works, ~60 ms | works, ~75–220 ms |
| `actuation` (read) | works, ~60 ms | works, ~75–300 ms |
| `actuation <n>` | works, ~160 ms, confirmed by read-back | works, ~210–360 ms, confirmed by read-back |
| `actuation next` | works | works, ~290–360 ms |
| `light` (read) | works, ~70 ms | works, ~90 ms |
| `light <n>`, `light next` | works, ~220 ms, confirmed by read-back | works, ~290 ms, confirmed by read-back |

- `SET_PROFILE` (`0x04`) is the actuation layer. Layer 2 was set to 0.55 mm in the vendor app. After `actuation 2`, keys triggered at 0.55 mm and the app showed layer 2 active.
- The backlight blinks once on a switch, the same as a switch from the app.
- The dongle shows up as PID `0x5026` ("MonsGeek 2.4G Wireless Keyboard"), not the `0x5038` in ry5088-flasher's hardware notes. It has its own vendor interface (page `0xFFFF`, usage 2).
- Writes pass through the dongle unchanged. The vendor app sends the same `04 <layer>` frame over the dongle as over the cable.
- Without `FC`, the dongle answers `GET_REPORT` with its own status frame, for example `00 64 64 00 01 01 01 7B`, or with the previous reply. The second byte (`0x64` = 100) is probably the battery level.
- The keyboard never relays a reply to a write over the dongle. So `fun60ctl` confirms it with a read (`GET_PROFILE` or `GET_LED`), repeated until the board reports the new state. A frame sent right after the write replaces it in the dongle before it goes out, so the read-back waits 50 ms first.
- A `GET_PROFILE` reply for layer 1 is byte for byte the request (`84 00 … 7B`), because the layer byte is `00`. It is still a real reply: on layer 2 it comes back as `84 01 …`.
- The radio link varies. Back-to-back reads usually take ~75 ms. With a poor link or right after the board has been idle, a reply can take up to a second or get lost. When no reply arrives within 1 s, `fun60ctl` says the keyboard didn't answer. A sleeping board doesn't answer until you press a key.
- The vendor app polls the dongle status every 2 s with an `F7` frame (no checksum). Before a switch it sends `FE 40` (no checksum). The keyboard announces a new layer with input report ID 5: `0F <layer 0-3>`. `fun60ctl` uses none of these.
- The `FC` step was found by capturing the vendor web app ([app.monsgeek.com](https://app.monsgeek.com)) in Chrome. A DevTools snippet wrapped `HIDDevice.sendFeatureReport` and `receiveFeatureReport` to log every frame while the app loaded a config.

### Bluetooth isn't supported

Over Bluetooth the board shows up as `3151:5027`. Its vendor collection is on page `0xFF55` and uses report ID 6 as input and output reports, not feature reports. `fun60ctl` doesn't speak that.

On macOS there's a second problem. The vendor collection shares one HID device with the keyboard. Opening it needs the Input Monitoring permission. And when Karabiner modifies the board, it seizes that device, so no other program can open it.

## Credits

The frame format, the `GET_INFOR` layout and the FUN60 Pro `dev_id` list come from [ry5088-flasher](https://github.com/dot-agi/ry5088-flasher), which documents the RY5088 vendor protocol.

## License

MIT, see [LICENSE](LICENSE).
