---
worth: later
where: main.go:314
added: 2026-11-15
---
# The board doesn't show which actuation layer is active

An actuation switch only blinks the backlight once, so afterwards you can't tell which layer you're on. The idea: `actuation <n>` could also select a custom lighting layer, for example custom 2 for the 0.55 mm gaming layer.

**Unknown that settles the value:** whether the lighting should follow the actuation layer day to day. The custom lighting layers are also a look you pick on its own, and `light <1-5>` already selects one with its own hotkey. If the two choices stay independent in practice, linking them only gets in the way.

The technical question is settled. `light <n>` selects a custom layer through `SET_LED` mode `0x0D`, and that doesn't overwrite the layer's contents (see the README's Lighting section). So a link would be an opt-in flag that maps actuation layer n to a custom lighting layer, in `cmdSetLayer`.
