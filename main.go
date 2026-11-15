// Command fun60ctl reads and switches the actuation layer on a MonsGeek
// FUN60 Pro running stock firmware, over the vendor HID interface.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	hid "github.com/sstallion/go-hid"

	"github.com/parMaster/fun60ctl/internal/proto"
)

const (
	vid          = 0x3151
	pidWired     = 0x5030
	vendorPage   = 0xFFFF
	vendorUsage  = 2
	replyDelay   = 50 * time.Millisecond
	replyTimeout = time.Second
	usageMessage = `usage: fun60ctl [flags] <command>

commands:
  info               show the connected board (VID:PID, dev_id, firmware)
  actuation          show the active actuation layer (1-4)
  actuation <1-4>    switch to that actuation layer
  actuation next     switch to the next layer, wrapping 4 -> 1

flags:
`
)

// Dongle PIDs. 0x5026 is the "MonsGeek 2.4G Wireless Keyboard" receiver seen
// on a FUN60 Pro; 0x5038 is the dongle PID listed in ry5088-flasher's hardware notes.
var dongleIDs = []uint16{0x5026, 0x5038}

// FUN60 Pro dev_ids, from the vendor board catalog.
var fun60ProIDs = []uint16{2304, 2305, 2464, 2600, 2785}

var (
	flagDongle  = flag.Bool("dongle", false, "use the 2.4 GHz dongle even when a cable is connected")
	flagForce   = flag.Bool("force", false, "switch layers even if the dev_id is not a known FUN60 Pro")
	flagVerbose = flag.Bool("v", false, "print HID interfaces, raw reports and timings")
)

func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usageMessage)
		flag.PrintDefaults()
	}
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "fun60ctl:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	validCmd := (len(args) == 1 && (args[0] == "info" || args[0] == "actuation")) ||
		(len(args) == 2 && args[0] == "actuation")
	if !validCmd {
		flag.Usage()
		os.Exit(2)
	}
	if len(args) == 2 {
		if _, err := targetLayer(args[1], 0); err != nil {
			return err
		}
	}
	start := time.Now()
	if err := hid.Init(); err != nil {
		return fmt.Errorf("init hidapi: %w", err)
	}
	defer func() { _ = hid.Exit() }()
	// Don't seize the interface, so the vendor app and key remappers keep working.
	hid.SetOpenExclusive(false)

	b, err := openBoard()
	if err != nil {
		return err
	}
	defer func() { _ = b.dev.Close() }()
	defer func() { debugf("total %v", time.Since(start).Round(time.Millisecond)) }()

	switch {
	case args[0] == "info":
		return cmdInfo(b)
	case len(args) == 1:
		return cmdGetLayer(b)
	default:
		return cmdSetLayer(b, args[1])
	}
}

type board struct {
	dev  *hid.Device
	info hid.DeviceInfo
}

func (b board) link() string {
	if b.isDongle() {
		return "dongle"
	}
	return "wired"
}

// openBoard finds the vendor config interface. Wired wins unless -dongle is set.
func openBoard() (board, error) {
	var found []hid.DeviceInfo
	err := hid.Enumerate(vid, hid.ProductIDAny, func(info *hid.DeviceInfo) error {
		debugf("found %04x:%04x page=0x%04X usage=%d if=%d %q",
			info.VendorID, info.ProductID, info.UsagePage, info.Usage, info.InterfaceNbr, info.Path)
		if (info.ProductID == pidWired || slices.Contains(dongleIDs, info.ProductID)) &&
			info.UsagePage == vendorPage && info.Usage == vendorUsage {
			found = append(found, *info)
		}
		return nil
	})
	if err != nil {
		return board{}, fmt.Errorf("enumerate HID: %w", err)
	}

	want := append([]uint16{pidWired}, dongleIDs...)
	if *flagDongle {
		want = dongleIDs
	}
	for _, pid := range want {
		for _, info := range found {
			if info.ProductID != pid {
				continue
			}
			dev, err := hid.OpenPath(info.Path)
			if err != nil {
				debugf("open %s: %v", info.Path, err)
				continue
			}
			debugf("using %04x:%04x (%s)", info.VendorID, info.ProductID, info.Path)
			return board{dev: dev, info: info}, nil
		}
	}
	if *flagDongle {
		return board{}, errors.New("no dongle vendor interface found (3151:5026 or 3151:5038, page 0xFFFF)")
	}
	return board{}, errors.New("no keyboard found (3151:5030 wired, 3151:5026 or 3151:5038 dongle); run with -v to list interfaces")
}

func (b board) isDongle() bool { return slices.Contains(dongleIDs, b.info.ProductID) }

// errDongleRead explains why reads fail over the dongle: it answers every
// GET_REPORT with its own status frame, and the way to fetch the keyboard's
// reply is still unknown. Writes do reach the keyboard.
var errDongleRead = errors.New("reading from the keyboard over the 2.4 GHz dongle isn't supported yet; connect the cable for this command")

// send writes one frame without waiting for a reply.
func (b board) send(frame [proto.ReportLen]byte) error {
	out := make([]byte, 1+proto.ReportLen) // byte 0 is report ID 0
	copy(out[1:], frame[:])
	debugf("send % X", frame[:8])
	if _, err := b.dev.SendFeatureReport(out); err != nil {
		return fmt.Errorf("send 0x%02X: %w", frame[0], err)
	}
	return nil
}

// transact sends one frame and polls until a reply echoes its opcode.
// Over the dongle the first reads can return other frames, because the
// keyboard's answer comes back over radio later.
func (b board) transact(frame [proto.ReportLen]byte) ([]byte, error) {
	if b.isDongle() {
		return nil, errDongleRead
	}
	t0 := time.Now()
	if err := b.send(frame); err != nil {
		return nil, err
	}
	var last []byte
	for time.Since(t0) < replyTimeout {
		time.Sleep(replyDelay)
		in := make([]byte, 1+proto.ReportLen)
		n, err := b.dev.GetFeatureReport(in)
		if err != nil {
			return nil, fmt.Errorf("read reply to 0x%02X: %w", frame[0], err)
		}
		last = in[:n]
		debugf("recv % X (%v)", last[:min(n, 10)], time.Since(t0).Round(time.Millisecond))
		if proto.Echoes(last, frame[0]) {
			return last, nil
		}
	}
	return last, nil // the parser reports the mismatch with the last bytes seen
}

func (b board) readInfo() (proto.Info, error) {
	r, err := b.transact(proto.GetInfo())
	if err != nil {
		return proto.Info{}, err
	}
	return proto.ParseInfo(r)
}

func (b board) readLayer() (byte, error) {
	r, err := b.transact(proto.GetLayer())
	if err != nil {
		return 0, err
	}
	return proto.ParseLayer(r)
}

func cmdInfo(b board) error {
	info, err := b.readInfo()
	if err != nil {
		return err
	}
	model := "unknown model"
	if slices.Contains(fun60ProIDs, info.DevID) {
		model = "FUN60 Pro"
	}
	fmt.Printf("%04x:%04x %s  dev_id %d (%s)  firmware %s\n",
		b.info.VendorID, b.info.ProductID, b.link(), info.DevID, model, info.Version)
	return nil
}

func cmdGetLayer(b board) error {
	l, err := b.readLayer()
	if err != nil {
		return err
	}
	fmt.Println(l + 1)
	return nil
}

func cmdSetLayer(b board, arg string) error {
	if b.isDongle() {
		return setLayerDongle(b, arg)
	}
	if !*flagForce {
		info, err := b.readInfo()
		if err != nil {
			return err
		}
		if !slices.Contains(fun60ProIDs, info.DevID) {
			return fmt.Errorf("dev_id %d is not a known FUN60 Pro; use -force to try anyway", info.DevID)
		}
	}
	// Only "next" depends on the current layer.
	var current byte
	if arg == "next" {
		l, err := b.readLayer()
		if err != nil {
			return err
		}
		current = l
	}
	target, err := targetLayer(arg, current)
	if err != nil {
		return err
	}
	frame, err := proto.SetLayer(target)
	if err != nil {
		return err
	}
	if _, err := b.transact(frame); err != nil {
		return err
	}
	got, err := b.readLayer()
	if err != nil {
		return fmt.Errorf("switched, but read-back failed: %w", err)
	}
	if got != target {
		return fmt.Errorf("asked for layer %d, board reports %d", target+1, got+1)
	}
	fmt.Println(got + 1)
	return nil
}

// setLayerDongle sends the switch without confirmation. The keyboard applies
// it (tested: it blinks and the vendor app shows the new layer), but the
// dongle gives no readable reply, so "next" and the dev_id check can't run.
func setLayerDongle(b board, arg string) error {
	if arg == "next" {
		return fmt.Errorf("\"next\" needs the current layer: %w", errDongleRead)
	}
	target, err := targetLayer(arg, 0)
	if err != nil {
		return err
	}
	frame, err := proto.SetLayer(target)
	if err != nil {
		return err
	}
	if err := b.send(frame); err != nil {
		return err
	}
	fmt.Printf("%d (sent via dongle, not confirmed)\n", target+1)
	return nil
}

func debugf(format string, a ...any) {
	if *flagVerbose {
		fmt.Fprintf(os.Stderr, format+"\n", a...)
	}
}
