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
	if err := setLayerDongle(dongleBoard(f), frame, 2); err != nil {
		t.Fatal(err)
	}
	want := []byte{proto.OpSetProfile, // no FC after the switch: it gets no reply
		proto.OpGetProfile, proto.OpFetchReply,
		proto.OpGetProfile, proto.OpFetchReply}
	if string(f.sent) != string(want) {
		t.Fatalf("sent % X, want % X", f.sent, want)
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
