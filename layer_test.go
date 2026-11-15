package main

import "testing"

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
		got, err := targetLayer(c.arg, c.current)
		if err != nil || got != c.want {
			t.Errorf("targetLayer(%q, %d) = %d, %v; want %d", c.arg, c.current, got, err, c.want)
		}
	}
	for _, bad := range []string{"0", "5", "x", "", "-1"} {
		if _, err := targetLayer(bad, 0); err == nil {
			t.Errorf("targetLayer(%q) should fail", bad)
		}
	}
}
