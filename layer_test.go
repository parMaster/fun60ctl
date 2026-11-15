package main

import (
	"testing"

	"github.com/parMaster/fun60ctl/internal/proto"
)

func TestTargetLayer(t *testing.T) {
	ok := []struct {
		arg     string
		current byte
		want    byte
	}{
		{"1", 2, 0},
		{"2", 0, 1},
		{"3", 0, 2},
		{"4", 0, 3},
		{"next", 0, 1},
		{"next", 2, 3},
		{"next", 3, 0}, // layer 4 wraps to layer 1
	}
	for _, c := range ok {
		got, err := targetLayer(c.arg, c.current, proto.LayerCount)
		if err != nil || got != c.want {
			t.Errorf("targetLayer(%q, %d) = %d, %v; want %d", c.arg, c.current, got, err, c.want)
		}
	}
	for _, bad := range []string{"0", "5", "x", "", "-1"} {
		if _, err := targetLayer(bad, 0, proto.LayerCount); err == nil {
			t.Errorf("targetLayer(%q) should fail", bad)
		}
	}
}

func TestTargetLightLayer(t *testing.T) {
	for _, c := range []struct {
		arg     string
		current byte
		want    byte
	}{
		{"5", 0, 4},
		{"next", 3, 4},
		{"next", 4, 0}, // custom 5 wraps to custom 1
	} {
		got, err := targetLayer(c.arg, c.current, proto.CustomLightCount)
		if err != nil || got != c.want {
			t.Errorf("targetLayer(%q, %d, 5) = %d, %v; want %d", c.arg, c.current, got, err, c.want)
		}
	}
	if _, err := targetLayer("6", 0, proto.CustomLightCount); err == nil || err.Error() != `layer must be 1..5 or next, got "6"` {
		t.Errorf("targetLayer(6) err = %v", err)
	}
}
