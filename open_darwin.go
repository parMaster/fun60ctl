package main

import hid "github.com/sstallion/go-hid"

// openShared stops hidapi from seizing the device on macOS, so the vendor
// app and key remappers keep working while fun60ctl has it open.
func openShared() { hid.SetOpenExclusive(false) }
