package main

import (
	"fmt"
	"strconv"

	"github.com/parMaster/fun60ctl/internal/proto"
)

// targetLayer turns the user's argument into a 0-based wire layer.
// Users count layers 1..4 to match the manual's Fn+F/G/H/J order.
// "next" moves one past current (0-based) and wraps 4 -> 1.
func targetLayer(arg string, current byte) (byte, error) {
	if arg == "next" {
		return (current + 1) % proto.LayerCount, nil
	}
	n, err := strconv.Atoi(arg)
	if err != nil || n < 1 || n > proto.LayerCount {
		return 0, fmt.Errorf("layer must be 1..%d or next, got %q", proto.LayerCount, arg)
	}
	return byte(n - 1), nil
}
