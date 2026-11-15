package main

import (
	"errors"
	"testing"

	hid "github.com/sstallion/go-hid"

	"github.com/parMaster/fun60ctl/internal/proto"
)

// fakeDongle mimics the 2.4 GHz dongle: GET_REPORT returns whatever sits in
// its buffer, and each FC replaces the buffer with the next scripted reply.
type fakeDongle struct {
	buf     []byte   // what GET_REPORT returns now
	replies [][]byte // what successive FCs load into buf
	sent    []byte   // opcodes of every frame sent, in order
}

func (f *fakeDongle) SendFeatureReport(p []byte) (int, error) {
	op := p[1] // p[0] is report ID 0
	f.sent = append(f.sent, op)
	if op == proto.OpFetchReply {
		f.buf = make([]byte, 1+proto.ReportLen) // zeros unless a reply is scripted
		if len(f.replies) > 0 {
			copy(f.buf, f.replies[0])
			f.replies = f.replies[1:]
		}
	}
	return len(p), nil
}

func (f *fakeDongle) GetFeatureReport(p []byte) (int, error) {
	return copy(p, f.buf), nil
}

func (f *fakeDongle) Close() error { return nil }

func dongleBoard(f *fakeDongle) board {
	return board{dev: f, info: hid.DeviceInfo{ProductID: dongleIDs[0]}}
}

func TestDongleReadSkipsStaleAndEarlyReplies(t *testing.T) {
	f := &fakeDongle{
		buf:     []byte{0x00, proto.OpGetProfile, 3}, // stale reply to an earlier GET_PROFILE
		replies: [][]byte{nil, {0x00, proto.OpGetProfile, 1}},
	}
	got, err := dongleBoard(f).readLayer()
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 {
		t.Fatalf("layer = %d, want 1 (3 is the stale reply)", got)
	}
	want := []byte{proto.OpGetProfile, proto.OpFetchReply, proto.OpFetchReply}
	if string(f.sent) != string(want) {
		t.Fatalf("sent % X, want % X", f.sent, want)
	}
}

func TestDongleSwitchRetriesReadBack(t *testing.T) {
	f := &fakeDongle{replies: [][]byte{
		{0x00, proto.OpGetProfile, 0}, // the board still reports the old layer
		{0x00, proto.OpGetProfile, 2},
	}}
	frame, err := proto.SetLayer(2)
	if err != nil {
		t.Fatal(err)
	}
	b := dongleBoard(f)
	ok, err := writeConfirmed(b, frame, func() (bool, error) {
		l, err := b.readLayer()
		return l == 2, err
	})
	if err != nil || !ok {
		t.Fatalf("ok = %v, err = %v", ok, err)
	}
	want := []byte{proto.OpSetProfile, // no FC after the switch: it gets no reply
		proto.OpGetProfile, proto.OpFetchReply,
		proto.OpGetProfile, proto.OpFetchReply}
	if string(f.sent) != string(want) {
		t.Fatalf("sent % X, want % X", f.sent, want)
	}
}

func TestLightTarget(t *testing.T) {
	custom3 := proto.LED{proto.ModeCustom, 4, 4, 0x20, 0, 0xC8, 0xC8}
	custom5 := proto.LED{proto.ModeCustom, 4, 4, 0x40, 0, 0xC8, 0xC8}
	alwaysOn := proto.LED{0x01, 2, 4, 0x06, 0xFA, 0xFF, 0xFA}
	for _, c := range []struct {
		name string
		arg  string
		cur  proto.LED
		want byte
	}{
		{"next from custom 3", "next", custom3, 3},
		{"next wraps from custom 5", "next", custom5, 0},
		{"next from a built-in mode", "next", alwaysOn, 0},
		{"explicit layer from a built-in mode", "4", alwaysOn, 3},
	} {
		got, err := lightTarget(c.arg, c.cur)
		if err != nil || got != c.want {
			t.Errorf("%s: got %d, %v; want %d", c.name, got, err, c.want)
		}
	}
}

func TestDescribeLED(t *testing.T) {
	if got := describeLED(proto.LED{proto.ModeCustom, 4, 4, 0x10}); got != "custom 2" {
		t.Errorf("got %q, want custom 2", got)
	}
	if got := describeLED(proto.LED{0x01, 2, 4, 0x06}); got != "built-in mode 1" {
		t.Errorf("got %q, want built-in mode 1", got)
	}
}

func TestDongleNoAnswer(t *testing.T) {
	_, err := dongleBoard(&fakeDongle{}).readLayer()
	if !errors.Is(err, errNoAnswer) {
		t.Fatalf("err = %v, want errNoAnswer", err)
	}
}

func TestWiredReadSendsNoFetch(t *testing.T) {
	f := &fakeDongle{buf: []byte{0x00, proto.OpGetProfile, 2}}
	b := board{dev: f, info: hid.DeviceInfo{ProductID: pidWired}}
	got, err := b.readLayer()
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("layer = %d, want 2", got)
	}
	if string(f.sent) != string([]byte{proto.OpGetProfile}) {
		t.Fatalf("sent % X, want only 84", f.sent)
	}
}
