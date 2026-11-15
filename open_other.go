//go:build !darwin

package main

// openShared does nothing here: hidapi opens devices in shared mode outside
// macOS, and go-hid has no exclusive-mode switch for other platforms.
func openShared() {}
