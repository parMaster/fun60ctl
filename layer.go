package main

import (
	"fmt"
	"strconv"
)

// targetLayer turns the user's argument into a 0-based wire layer out of count.
// Users count layers from 1: actuation 1..4 matches the manual's Fn+F/G/H/J
// order, lighting 1..5 matches the vendor app's custom layers.
// "next" moves one past current (0-based) and wraps count -> 1.
func targetLayer(arg string, current, count byte) (byte, error) {
	if arg == "next" {
		return (current + 1) % count, nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > int(count) {
		return 0, fmt.Errorf("layer must be 1..%d or next, got %q", count, arg)
	}
	return byte(n - 1), nil
}
