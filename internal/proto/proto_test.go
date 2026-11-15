package proto

import (
	"errors"
	"testing"
)

func headerSum(f [ReportLen]byte) byte {
	var s byte
	for _, b := range f[:8] {
		s += b
	}
	return s
}

func TestFramesSumToFF(t *testing.T) {
	set, err := SetLayer(2)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][ReportLen]byte{
		"GetInfo":  GetInfo(),
		"GetLayer": GetLayer(),
		"SetLayer": set,
	}
	for name, f := range cases {
		if s := headerSum(f); s != 0xFF {
			t.Errorf("%s: header sums to 0x%02X, want 0xFF", name, s)
		}
		for i, b := range f[8:] {
			if b != 0 {
				t.Errorf("%s: byte %d = 0x%02X, want 0 padding", name, i+8, b)
			}
		}
	}
}

// Known-good vector: GET_INFOR (0x8F) carries checksum 0x70.
func TestGetInfoMatchesFlasherVector(t *testing.T) {
	f := GetInfo()
	if f[0] != 0x8F || f[7] != 0x70 {
		t.Fatalf("got op 0x%02X checksum 0x%02X, want 0x8F / 0x70", f[0], f[7])
	}
}

func TestFetchReplyHasNoChecksum(t *testing.T) {
	f := FetchReply()
	if f[0] != OpFetchReply {
		t.Fatalf("op = 0x%02X, want 0xFC", f[0])
	}
	for i, b := range f[1:] {
		if b != 0 {
			t.Fatalf("byte %d = 0x%02X, want 0", i+1, b)
		}
	}
}

// Captured on hardware: the write that selects custom lighting layer 1.
func TestSetLEDMatchesHardwareVector(t *testing.T) {
	f := SetLED(LED{0x0D, 0x04, 0x04, 0x00, 0x00, 0xC8, 0xC8})
	want := []byte{0x07, 0x0D, 0x04, 0x04, 0x00, 0x00, 0xC8, 0xC8, 0x53}
	if string(f[:9]) != string(want) {
		t.Fatalf("got % X, want % X", f[:9], want)
	}
	for i, b := range f[9:] {
		if b != 0 {
			t.Fatalf("byte %d = 0x%02X, want 0 padding", i+9, b)
		}
	}
}

func TestParseLED(t *testing.T) {
	cases := []struct {
		name   string
		resp   []byte
		mode   byte
		layer  byte
		custom bool
	}{
		{"custom 1", []byte{0x00, 0x87, 0x0D, 0x04, 0x04, 0x00, 0x00, 0xC8, 0xC8}, 0x0D, 0, true},
		{"custom 2", []byte{0x00, 0x87, 0x0D, 0x04, 0x04, 0x10, 0x00, 0xC8, 0xC8}, 0x0D, 1, true},
		{"always on", []byte{0x00, 0x87, 0x01, 0x02, 0x04, 0x06, 0xFA, 0xFF, 0xFA}, 0x01, 0, false},
	}
	for _, c := range cases {
		l, err := ParseLED(c.resp)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		layer, custom := l.Custom()
		if l.Mode() != c.mode || custom != c.custom || (custom && layer != c.layer) {
			t.Errorf("%s: mode 0x%02X layer %d custom %v", c.name, l.Mode(), layer, custom)
		}
	}
	if _, err := ParseLED([]byte{0x00, 0x84, 0x00}); !errors.Is(err, ErrBadReply) {
		t.Errorf("wrong echo: err = %v, want ErrBadReply", err)
	}
}

func TestWithCustomKeepsOtherBytes(t *testing.T) {
	alwaysOn := LED{0x01, 0x02, 0x04, 0x06, 0xFA, 0xFF, 0xFA}
	got := alwaysOn.WithCustom(2)
	want := LED{0x0D, 0x02, 0x04, 0x26, 0xFA, 0xFF, 0xFA}
	if got != want {
		t.Fatalf("got % X, want % X", got, want)
	}
}

func TestSetLayerFrame(t *testing.T) {
	f, err := SetLayer(3)
	if err != nil {
		t.Fatal(err)
	}
	if f[0] != OpSetProfile || f[1] != 3 || f[7] != 0xFF-0x04-0x03 {
		t.Fatalf("got % X", f[:8])
	}
	if _, err := SetLayer(4); err == nil {
		t.Fatal("SetLayer(4) should fail")
	}
}

func TestParseInfo(t *testing.T) {
	// dev_id 2304 = 0x0900 LE, version major 3 minor 6 -> v306
	body := []byte{0x8F, 0x00, 0x09, 0, 0, 0, 0, 6, 3, 0, 0}
	for name, resp := range map[string][]byte{
		"plain":          body,
		"with report id": append([]byte{0}, body...),
	} {
		got, err := ParseInfo(resp)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got.DevID != 2304 || got.Version != "v306" {
			t.Errorf("%s: got %+v", name, got)
		}
	}
}

func TestParseLayer(t *testing.T) {
	body := []byte{0x84, 2, 0, 0, 0, 0, 0, 0}
	for name, resp := range map[string][]byte{
		"plain":          body,
		"with report id": append([]byte{0}, body...),
	} {
		got, err := ParseLayer(resp)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got != 2 {
			t.Errorf("%s: got %d, want 2", name, got)
		}
	}
}

func TestParseRejectsWrongEcho(t *testing.T) {
	if _, err := ParseInfo([]byte{0x84, 0, 9, 0, 0, 0, 0, 6, 3}); !errors.Is(err, ErrBadReply) {
		t.Errorf("ParseInfo: got %v, want ErrBadReply", err)
	}
	if _, err := ParseLayer([]byte{0x8F, 1}); !errors.Is(err, ErrBadReply) {
		t.Errorf("ParseLayer: got %v, want ErrBadReply", err)
	}
	if _, err := ParseLayer([]byte{0x84, 7}); !errors.Is(err, ErrBadReply) {
		t.Errorf("ParseLayer out of range: got %v, want ErrBadReply", err)
	}
	if _, err := ParseLayer(nil); !errors.Is(err, ErrBadReply) {
		t.Errorf("ParseLayer(nil): got %v, want ErrBadReply", err)
	}
}
