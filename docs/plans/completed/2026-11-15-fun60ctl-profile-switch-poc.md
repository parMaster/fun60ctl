# fun60ctl: actuation layer switch POC

**Goal:** prove that a one-shot Go CLI can read and switch the active actuation layer on a stock-firmware FUN60 Pro, wired and over the 2.4 GHz dongle.

**Kind of change:** feature

## Intent

The FUN60 Pro manual says Fn+F/G/H/J switches between four actuation layers (sets of magnetic-switch settings). Recent stock firmware dropped that shortcut. Today the only way to switch is the vendor driver, which is slow over wireless. The stock vendor protocol has `SET_PROFILE` (`0x04`) and `GET_PROFILE` (`0x84`), which probably control the actuation layer. If they do, a small CLI that sends one HID feature report does the same switch in milliseconds. A hotkey tool (Karabiner, PowerToys) calls it later.

This POC answers four questions on real hardware:

1. Does stock firmware accept `SET_PROFILE` from a plain HID client?
2. Is `SET_PROFILE` the actuation layer, and not a lighting layer, an Fn layer or the whole board?
3. Does the switch change the magnetic settings (actuation, rapid trigger)?
4. Does the same frame work through the dongle (`3151:5038`)?

## Decisions

- **Go, fully isolated in `fun60ctl/` with its own `go.mod`** — the user knows Go. Nothing outside `fun60ctl/` changes, apart from this plan. That means no edits to the root README, CI or `.gitignore`. The folder can move to its own repo later as-is.
- **HID library: `github.com/sstallion/go-hid`** — it wraps hidapi, the library the Rust flasher already uses with these boards. It supports feature reports on macOS, Windows and Linux. It needs cgo, so cross-compiling needs `zig cc` or mingw. Pure-Go HID libraries were dropped because their Windows and macOS feature-report support is less proven.
- **One-shot binary, no daemon** — each call opens the device, sends one frame, reads the reply and exits.
- **Commands use the app's word "layer"**, and `actuation` names which kind. The board has three kinds of layer, so a bare `profile` or `layer` would be ambiguous.

  | Command | Effect |
  |---|---|
  | `fun60ctl info` | print VID:PID, path (wired/dongle), `dev_id`, firmware version |
  | `fun60ctl actuation` | print the active actuation layer |
  | `fun60ctl actuation <1-4>` | switch to that actuation layer |
  | `fun60ctl actuation next` | switch to the next one, wrapping 4 → 1 |

- **Layers are 1–4 to the user, 0–3 on the wire** — this matches the manual's F/G/H/J order. The CLI converts in one place.
- **Device discovery:**

  ```
  enumerate HID
    → VID 0x3151, PID 0x5030 (wired) or 0x5038 (dongle)
    → usage page 0xFFFF, usage 2
    → GET_INFOR (0x8F) → dev_id
    → dev_id in {2304, 2305, 2464, 2600, 2785}? → use it
  ```

  Wired wins when both are present. A `--dongle` flag forces the dongle path for testing. `info` prints any `dev_id` it gets, even unknown ones. `actuation <n>` refuses a `dev_id` outside the FUN60 Pro list.

- **Frame format is ported, not shared** — the Bit7 header and the `GET_INFOR` parsing come from `flasher/src/proto.rs` (`hdr_bit7`, `parse_infor`). They are re-written in Go with the same unit-test vectors.

## Constraints / out of scope

- No firmware flashing. The CLI never sends `0xC5` or `0x7F`, because `ENTER_BOOTLOADER` wipes the config region.
- No other writes than the actuation-layer switch.
- Out of scope: hotkey setup, GitHub Actions, a separate repo, the release process, and other settings (LED, polling).
- If the dongle needs a routing byte, the POC records that finding. Reverse-engineering the dongle protocol is a follow-up plan.

## Traps

- **The board has three kinds of layer, and only one is our target.**

  | Kind | Count | Likely command |
  |---|---|---|
  | Actuation (magnetic switch settings) | 4 | `SET_PROFILE 0x04`? (count matches 0–3) |
  | Custom lighting | 5 | LED / userpic family (`0x07`, `0x0C`) |
  | Fn layer | 2 | `SET_KBOPTION 0x09` "Fn-layer bank" |

  `docs/firmware.md` §2.3 says `0x04` "reloads keymap/config". It doesn't say whether that means the actuation layer or a whole-board profile. The match is only by count until hardware proves it.
- **The repo's firmware is a rewrite, not the stock firmware.** `firmware/common/vendor_proto.c` (`FEA_SET_PROFILE`) only stores the byte and sets a dirty flag. That proves nothing about what stock firmware reloads on a switch. Only hardware tests answer questions 2 and 3.
- **Feature reports carry a leading report ID.** The flasher sends a 65-byte buffer: report ID `0`, then the 64-byte frame (`flasher/src/device.rs`, `send`). go-hid's `SendFeatureReport` uses the same convention.
- **The reply may or may not start with the report ID.** The flasher's `read_infor` tries to parse the reply as-is, then again without the first byte. The Go port needs the same fallback for every reply.
- **The firmware needs time before the reply is ready.** The flasher sleeps 50 ms between send and `GET_REPORT` (`read_infor`).
- **Byte 0 of the reply echoes the opcode.** `parse_infor` checks `resp[0] == 0x8F`. The `GET_PROFILE` reply should start with `0x84`, with the layer in byte 1. Check the echo before trusting the value.
- **The checksum covers bytes 0..6 only** and is written to byte 7 (`hdr_bit7`). Byte 8 is the Bit8 form, used only for LED commands.
- **One PID covers many models.** `0x5030` is shared across the family (`docs/models.md`), so only the `dev_id` check identifies a FUN60 Pro.
- **Over the dongle, `GET_INFOR` may return the dongle's own identity** instead of the keyboard's. `docs/cloud-api.md` mentions a `DangleDevType` routing field in the vendor driver. If `info` over the dongle shows a strange `dev_id`, that's why.

## Definition of Done

- [x] Frame building is correct — proof: unit tests show the `GET_INFOR`, `GET_PROFILE` and `SET_PROFILE` frames each sum to `0xFF` over bytes 0..7, with the same vectors as `proto.rs` tests (`0x8F` → checksum `0x70`).
- [x] Reply parsing handles both reply shapes — proof: unit tests parse a `GET_INFOR` reply and a `GET_PROFILE` reply with and without a leading `0x00`. They reject a reply whose echo byte is wrong.
- [x] Layer argument handling — proof: unit tests show `1..4` maps to `0..3`, `next` wraps 4 → 1, and `0`, `5` and `x` are rejected.
- [x] `fun60ctl info` identifies the board wired — proof: on the user's FUN60 Pro, it prints PID `5030`, a `dev_id` from the Pro list and a firmware version.
- [x] `fun60ctl actuation` reads the active layer wired — proof: the printed layer matches the one the vendor driver shows.
- [x] `fun60ctl actuation <n>` switches wired — proof: after the switch, `fun60ctl actuation` and the vendor driver both show the new layer.
- [x] The switch changes magnetic settings — proof: set two actuation layers to very different actuation points in the driver (for example 0.3 mm vs 3.5 mm). After the CLI switches, typing feels clearly different. Record the result in the README, whichever way it goes.
- [x] *(lighting confirmed unchanged; Fn layer not checked)* The switch changes only the actuation layer — proof: after a switch, the lighting layer and the Fn layer stay the same, both on the board and in the driver. If `0x04` turns out to be a different kind of layer, the next DoD item becomes required.
- [x] *(not needed: `0x04` is the actuation layer)* *(only if `0x04` is the wrong layer)* The real actuation-layer command is known — proof: a capture of the bytes the web driver sends when you switch actuation layers, saved in `fun60ctl/README.md`. In Chrome, a devtools snippet can wrap `HIDDevice.prototype.sendFeatureReport` to log each report. The CLI then sends that command.
- [x] Dongle result is known — proof: `info`, `actuation` and `actuation <n>` run with `--dongle`. The README records which work and how long each call takes. If they fail, it records the error and the `dev_id` seen.
  - ➕ Writes pass through the dongle (PID `0x5026`): `actuation <n>` works send-only, confirmed on hardware both ways. Reads don't: the dongle answers with its own status frame. `info`, `actuation` (read) and `next` report that clearly. Reading over the dongle is a follow-up.
- [x] The README tells someone how to build and run it — proof: `fun60ctl/README.md` covers the build on macOS, cross-compiling for Windows with `zig cc`, the commands and the hardware results.

## Work order

1. Frames, parsing and unit tests (no hardware needed).
2. `info` and `actuation` (read-only) on hardware, wired.
3. `actuation <n>` and `next`, wired, then the magnetism and other-layers checks.
4. The same three commands over the dongle.

## Wrap-up

- [ ] full test suite passes: `cd fun60ctl && go test ./...`
- [ ] linter passes: `cd fun60ctl && golangci-lint run ./...`
- [ ] `git status` shows changes only under `fun60ctl/` and `docs/plans/`
- [ ] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/2026-11-15-fun60ctl-profile-switch-poc.md docs/plans/completed/`)
- [ ] single commit: all changes + plan move

## Post-Completion

- Windows hardware test with a cross-compiled `.exe`.
- Hotkeys: Karabiner maps ctrl+opt+1..4 to `fun60ctl actuation 1..4`. PowerToys gets the same on Windows.
- A lighting color per actuation layer, so the board shows which layer is active. Fn layers can't do this, because the host never sees Fn.
- GitHub Actions builds per OS, and moving `fun60ctl/` to its own repo.
