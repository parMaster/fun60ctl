---
worth: later
where: main.go:283
added: 2026-11-15
---
# The board doesn't show which actuation layer is active

A switch only blinks the backlight once, so afterwards you can't tell which layer you're on. The idea: after a switch, also set a lighting color per actuation layer, for example "red = 0.55 mm gaming layer".

**Unknown that settles the value:** whether setting the LED clobbers the user's custom lighting layer (there are 5). It might also write to flash on every switch. If it overwrites a custom layer, this isn't worth doing as-is.

What's known:

- `SET_LED` is `0x07`, with params `[mode][speed][brightness][options][R][G][B]`. It uses the **Bit8** checksum: bytes 0..8 sum to `0xFF`, and the checksum goes in byte 8. `GET_LED` is `0x87`. Source: ry5088-flasher's `docs/firmware.md`.
- Fn layers can't get this. The keyboard handles Fn internally, so the host never sees an Fn-layer change.
- Over the dongle, writes work, so an LED write should reach the keyboard too.

First test: read the LED params with `0x87`, change the color, then check in the vendor app whether a lighting layer changed.
