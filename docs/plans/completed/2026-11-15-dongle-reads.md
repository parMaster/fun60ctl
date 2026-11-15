# Reads over the 2.4 GHz dongle

**Goal:** `info`, `actuation`, `actuation next` and a confirmed switch work over the dongle the same way they work over the cable.

**Kind of change:** feature

## Intent

Over the dongle, `fun60ctl` can only send a switch. Every read fails with `errDongleRead`, because the dongle answers `GET_REPORT` with its own status frame. A capture of the vendor app (app.monsgeek.com) showed the missing step. After a command, the app sends `FC` (no checksum), and the next `GET_REPORT` returns the keyboard's reply. A probe confirmed it from Go: `GET_PROFILE` and `GET_INFO` both returned the same replies as over the cable, opcode echo included.

This change makes `transact` fetch replies through `FC` on the dongle. Then the dongle path runs the same commands as the wired path: the dev_id check, `next`, and the read-back after a switch.

```
before (dongle):  transact → errDongleRead
after  (dongle):  send cmd → loop { wait ~50 ms → send FC → GET_REPORT → echo? }
wired (unchanged): send cmd → loop { wait 50 ms → GET_REPORT → echo? }
```

## Decisions

- **One `transact` with a dongle branch.** The dongle branch sends `FC` before each read in the poll loop. Everything above `transact` stays shared, so `setLayerDongle` and `errDongleRead` go away. Dropped: a separate dongle-only command set, which duplicates `cmdSetLayer`.
- **Re-send `FC` on every poll.** An early `FC` returns zeros, and later reads stay zero until the next `FC` (probe: zeros at 23, 44 and 66 ms after one early `FC`). The loop keeps the existing `replyTimeout` of 1 s.
- **The switch goes through `transact` like on the cable, and the read-back decides success.** The vendor app never reads a reply to `04`. If no echo of `04` arrives, `transact` already returns the last bytes without an error. The `GET_PROFILE` read-back is what confirms the layer. Check on hardware that a missing `04` echo doesn't add the full 1 s timeout to every switch. If it does, send the switch without polling over the dongle, and keep the read-back.
- **A small interface over the HID device.** `transact` takes something with `SendFeatureReport` and `GetFeatureReport`, so a test can fake the dongle with scripted replies. `*hid.Device` satisfies it unchanged.
- **No reply means a clear error.** If the timeout passes without an echo over the dongle, say that the keyboard didn't answer, and suggest checking that it's in 2.4 GHz mode. This also catches the case where the dongle has no keyboard linked: the dev_id check now runs first, so the switch fails instead of printing "sent".

## Constraints / out of scope

- The wired path must behave exactly as today, with the same frames, timing and output.
- No change to the `proto` frames or parsers. The dongle replies already match `echoed`'s `00 <op>` shape.
- `FE 40` (sent by the app before a switch) and input report 5 (`0F <layer>`) stay unexplained, and they aren't used.
- No Bluetooth support.

## Traps

- **A stale reply can echo the right opcode.** Without `FC`, `GET_REPORT` returns the previous reply. The probe saw an `84` reply still sitting there before the next command. A read-back of `GET_PROFILE` after an earlier `GET_PROFILE` could accept the old layer. The dongle branch must never read before sending `FC` for the current command. Nobody has checked whether an early `FC` can return the previous reply instead of zeros. The hardware DoD item covers that case.
- **`FC` is a raw frame.** `FC` is byte 0, and the rest is zeros, with no checksum. `proto.Frame` would add a checksum byte, so don't build it with `Frame`.
- **`transact` returns `last` on timeout without an error** (`main.go`, `transact`). The parsers turn that into `ErrBadReply`. Over the dongle, that message would show the dongle's zeros. Give the clearer error from Decisions instead.
- **`cmdSetLayer` branches to `setLayerDongle` first** (`main.go`, `cmdSetLayer`), and `-force` skips the dev_id check only in the wired branch. After the change, `-force` should skip it over the dongle too.
- **The README's hardware table and dongle bullets describe the old behavior.** They say reads are "not supported" and describe the status frame as the only reply.

## Definition of Done

- [x] Over the dongle, `fun60ctl info` prints `3151:5026 dongle  dev_id 2785 (FUN60 Pro)  firmware v407` — proof: run it on hardware.
- [x] Over the dongle, `fun60ctl actuation` prints the layer that the vendor app shows — proof: run it on hardware. The app showed layer 2, and the read gave `84 01`.
- [x] Over the dongle, `actuation <n>` and `actuation next` print the new layer, confirmed by read-back — proof: 10 switches in a row (2 3 4 1 next×4 3 1) all confirmed.
- [x] A switch over the dongle takes well under 1 s — proof: 210–357 ms with a good radio link.
- [x] With the dongle plugged in but the board on Bluetooth or the cable, a dongle command fails with the "keyboard didn't answer" error and switches nothing — proof: `-dongle actuation 2` with the board on the cable failed after 1.08 s, and the layer stayed 1.
- [x] The dongle reply loop ignores zeros and stale replies — proof: `TestDongleReadSkipsStaleAndEarlyReplies` in `main_test.go`.
- [x] ➕ The keyboard never relays a reply to `04` over the dongle, so a switch through `transact` always timed out. Over the dongle the switch is now send-only, then confirmed by a read-back — proof: hardware run above, plus `TestDongleSwitchRetriesReadBack`.
- [x] ➕ A frame sent right after `04` replaces it in the dongle, so the switch never landed. The read-back now waits 50 ms after the switch, and it repeats until it sees the target layer — proof: before the wait, the board stayed on the old layer for 1 s; with it, all 10 switches landed.
- [x] ➕ The reply lands about 20 ms after `FC`, so each poll waits `fetchDelay` before `GET_REPORT` — proof: without it, `info` needed 8 `FC`s.
- [x] ➕ `FetchReply` lives in `proto` next to the other frames — proof: `TestFetchReplyHasNoChecksum`.
- [x] ➕ The radio link varies with position and use. With a poor link, replies take up to 1 s or get lost — proof: `tmp/dongleprobe` got 16 of 20 at one position and 40 of 40 after moving the laptop.
- [x] The wired path is unchanged — proof: `info` 71 ms with the same frames and no `FC`; read, `next` and switch 78–240 ms. `TestWiredReadSendsNoFetch` covers it too.
- [x] The README describes reads over the dongle — proof: the hardware table shows the dongle results, and the dongle bullets explain the `FC` fetch.
- [x] The vendor app URL and the capture method survive the backlog item's deletion — proof: the README mentions app.monsgeek.com where it explains how the protocol was captured.
- [x] `docs/backlog/dongle-reads.md` is deleted with `git rm` in the final commit — proof: `git show --stat HEAD`.

## Wrap-up

- [x] full test suite passes: `go test ./...`
- [x] linter passes: `golangci-lint run ./...`
- [x] README.md updated
- [x] delete `tmp/dongleprobe` before the final lint run, because `./...` picks it up locally
- [x] move this plan to `docs/plans/completed/` (`mkdir -p docs/plans/completed && mv docs/plans/2026-11-15-dongle-reads.md docs/plans/completed/`)
- [x] single commit: all changes, plan move, and the backlog item deletion

## Post-Completion

- Tag a release (`v0.2.0`) when you want the dongle reads in the binaries and in `go install @latest`.
