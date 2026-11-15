---
worth: yes
where: main.go:156
added: 2026-11-15
---
# Reads don't work over the 2.4 GHz dongle, so `info`, `actuation` and `next` need the cable

Over the dongle (PID `0x5026`), writes reach the keyboard. `actuation <n>` works send-only, tested both ways on hardware. Reads fail, because every `GET_REPORT` returns the dongle's own status frame, never the keyboard's reply. So `info`, `actuation` (read) and `actuation next` return `errDongleRead`.

## What we know

- **Status frame:** `00 64 64 00 01 01 01 7B` (after the report ID). `0x64` = 100, probably the battery level. The first byte flips `00` → `01` some time after a command, so it may mean "reply waiting". It's not the layer: it went to `01` after a switch to layer 1 (wire value `0`).
- **The vendor web app** (captured in Chrome with WebHID):
  - It polls every 2 s with `F7 00 …` (no checksum) and reads the status frame back.
  - Before a switch it sends `FE 40` (no checksum). Its meaning is unknown.
  - The switch itself is the same `04 <layer> 00 00 00 00 00 <cs>` frame as wired.
  - It never reads a feature reply for the switch. The keyboard announces the new layer with **input report ID 5: `0F <layer 0-3>`**.

## Next step

Capture the app while it loads a full config over the dongle. That has to read data, so it shows how replies are fetched. Maybe `FE 40` arms a relay, or maybe the reply arrives as an input report.

Capture snippet, pasted into the DevTools console of the web driver. It hides the `F7` poll, then `copy(__hid.join('\n'))` copies the log:

```js
(() => {
  const hex = d => Array.from(new Uint8Array(d.buffer ?? d, d.byteOffset ?? 0, d.byteLength))
    .map(b => b.toString(16).padStart(2, '0')).join(' ').replace(/( 00)+$/, ' …');
  const log = window.__hid = [];
  const add = s => { log.push(s); console.log(s); };
  const P = HIDDevice.prototype;
  let skipReply = false;
  for (const name of ['sendFeatureReport', 'sendReport']) {
    const orig = P[name];
    P[name] = function (id, data) {
      const h = hex(data);
      skipReply = h.startsWith('f7');
      if (!skipReply) add(`[hid] ${name} id=${id} ${h}`);
      return orig.call(this, id, data);
    };
  }
  const recv = P.receiveFeatureReport;
  P.receiveFeatureReport = async function (id) {
    const r = await recv.call(this, id);
    if (!skipReply) add(`[hid] recvFeature id=${id} ${hex(r)}`);
    skipReply = false;
    return r;
  };
  navigator.hid.getDevices().then(ds => ds.forEach(d => d.addEventListener('inputreport', e =>
    add(`[hid] input id=${e.reportId} ${hex(e.data)}`))));
  add('[hid] logging on (pings hidden)');
})();
```

## Watch out

If replies come as input reports, they may arrive on interface 1. That interface also carries the keyboard usages (page `0x0001`, usage 6), so reading it on macOS may need the Input Monitoring permission. Karabiner's `shell_command` would then need that permission too.
