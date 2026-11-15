# Select custom lighting layers from the command line

**Goal:** `fun60ctl light` prints the active lighting, and `light <1-5>` / `light next` select one of the board's 5 custom lighting layers, wired and over the dongle.

**Kind of change:** feature

## Intent

Fn+Enter switches the board to its built-in lighting modes, and no key brings it back to the 5 custom lighting layers. Fn+/ cycles the custom layers only while one is already active. Today the only way back is the vendor app. Hardware tests on the cable showed the protocol:

| State | `GET_LED` reply (after `87`) |
|---|---|
| custom layer 1 | `0D 04 04 00 00 C8 C8` |
| custom layer 2 | `0D 04 04 10 00 C8 C8` |
| built-in "Always on" | `01 02 04 06 FA FF FA` |

The bytes are `[mode][speed][brightness][options][R][G][B]`. Mode `0D` is custom, and the high nibble of options is the custom layer minus 1. Writing `SET_LED` (`07`) with mode `0D` and options `(n−1)<<4` selected custom layers 1–5 on hardware, and it brought the board back from "Always on". The board echoed the write over the cable.

This adds a `light` command so a hotkey can return to a custom layer.

## Decisions

- **Command shape mirrors `actuation`.**

  ```
  fun60ctl light            # "custom 2", or "built-in mode 1"
  fun60ctl light <1-5>      # select custom layer n, confirmed by read-back
  fun60ctl light next       # custom n → n+1, 5 → 1; from a built-in mode → custom 1
  ```

- **Read, change two fields, write back.** `light <n>` reads `GET_LED`, sets mode to `0D` and the high nibble of options to `n−1`, and keeps every other byte. Then it reads back and checks mode and nibble. Dropped: writing a fixed frame, which would reset brightness and the other bytes to whatever we hard-code.
- **The `SET_LED` frame gets its own builder in `proto`, with the Bit8 checksum.** `proto.Frame` puts the checksum in byte 7 and copies only 7 payload bytes, while `SET_LED` carries 7 params after the opcode. `GET_LED` uses the normal Bit7 `Frame`, which the hardware accepted.
- **Over the dongle, the write follows the actuation switch pattern.** Send without polling, wait `replyDelay`, then repeat `GET_LED` until it shows the target or `replyTimeout` passes. Refactor `setLayerDongle` into one helper that both commands share, instead of copying it.
- **The dev_id check and `-force` apply to `light` too.** It's the same safety gate: only write to a known FUN60 Pro.
- **The custom layer count is a `proto` constant (5).** Make `targetLayer` take the count, so actuation (4) and lighting (5) share it, and the error message names the right range.

## Constraints / out of scope

- `actuation` must behave exactly as today.
- No editing of custom layer contents, colors, speed or brightness. No selecting built-in modes.
- No coupling between actuation and lighting layers. The backlog item keeps that idea.
- No hotkey example for `light` in the README beyond one line saying a hotkey can call it.

## Traps

- `run` accepts only `info` and `actuation` in its `validCmd` check, and the usage text lists only those (`main.go`, `run`, `usageMessage`). Both need `light`.
- `targetLayer` hard-codes `proto.LayerCount` (4) in its range check and its error text (`layer.go`). `TestTargetLayer` calls it with two arguments and pins the actuation behavior, including that "5" fails. Changing the signature means updating those calls, but the actuation cases must keep the same results.
- The wired `SET_LED` echo carries the opcode `07`, so `transact` works for the write on the cable. Over the dongle, nobody has tested whether `07` gets a reply. Assume it doesn't, like `04`.
- In a built-in mode, `GET_LED` returns that mode's speed, options and color (`02`, `06`, `FA FF FA`), not the custom layer's. Writing mode `0D` with those bytes is untested. Brightness was `04` in both states, so it's likely global. The DoD checks that a custom layer looks right after `light <n>` from a built-in mode.
- The low nibble of options was `0` in custom and `6` in "Always on". Its meaning is unknown, so keep it as read. If a custom layer then looks wrong, clear it.
- `GET_LED`'s reply pads with zeros, and the parser needs at least 8 bytes after the echo (`proto.echoed` takes the minimum length).

## Definition of Done

- [x] `light` prints `custom <n>` on a custom layer and `built-in mode <m>` otherwise — proof: `TestParseLED`, `TestDescribeLED`, and hardware reads of `custom 5` and `built-in mode 6` ("Stars twinkle").
- [x] `light <1-5>` selects that custom layer over the cable, confirmed by read-back — proof: `light 3` printed `custom 3` in 222 ms.
- [x] Selecting a custom layer brings the board back from a built-in mode, with brightness unchanged and the layer looking as designed — proof: from "Stars twinkle", `light 5` wrote `0D 02 04 47 FA FF FA`. The user confirmed the layer looked the same, and Fn+/ still cycled.
- [x] `light next` cycles 1→2→…→5→1, and from a built-in mode it goes to custom 1 — proof: `TestLightTarget`, plus `next` 3→4→5→1 on the cable.
- [x] The `SET_LED` frame has the Bit8 checksum — proof: `TestSetLEDMatchesHardwareVector`.
- [x] `light <n>` works over the dongle, confirmed by read-back — proof: 5× `light next` (5→1→2→3→4→5), ~290 ms each.
- [x] A bad argument fails with a clear range — proof: `light 6` prints `layer must be 1..5 or next, got "6"`, and `TestTargetLayer` passes.
- [x] `actuation` is unchanged — proof: the tests pass, and wired read, `next` and switch worked in 66–221 ms.
- [x] The README documents `light` and the `SET_LED` / `GET_LED` bytes — proof: the Commands section and a new Lighting section under How it works.
- [x] `docs/backlog/lighting-per-actuation-layer.md` is rewritten: its unknown is answered (custom layers can be selected without overwriting them), `where` is updated or dropped, and it keeps only the idea of linking actuation to lighting — proof: the item now asks whether lighting should follow actuation at all.
- [x] ➕ `setLayerDongle` became `writeConfirmed`, shared by both commands for cable and dongle. The dev_id check moved into `checkModel` — proof: `TestDongleSwitchRetriesReadBack` passes against `writeConfirmed`.
- [x] ➕ `-v` shows 9 bytes per sent frame, so the `SET_LED` checksum is visible — proof: `send 07 0D 02 04 47 FA FF FA AB` in the hardware log.
- [x] ➕ Carrying a built-in mode's speed, low options nibble and color into the custom write doesn't change how a custom layer looks — proof: the user's check above.

## Wrap-up

- [x] full test suite passes: `go test ./...`
- [x] linter passes: `golangci-lint run ./...`
- [x] README.md updated
- [x] delete `tmp/ledprobe` before the final lint run
- [x] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/2026-11-15-light-command.md docs/plans/completed/`)
- [x] single commit: all changes, plan move, and the backlog item rewrite

## Post-Completion

- Map a hotkey to `light 1` in Karabiner if you want a one-key way back from the built-in modes.
- Tag a release when you want `light` in the binaries.
