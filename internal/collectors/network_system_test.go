package collectors

import (
	"strings"
	"testing"
)

func TestDefaultIface(t *testing.T) {
	route := "Iface\tDestination\tGateway\n" +
		"lo\t00000000\t00000000\n" +
		"docker0\t00000000\t00000000\n" +
		"wlo1\t00000000\t0102A8C0\n" +
		"wlo1\t0002A8C0\t00000000\n"
	iface, err := defaultIface([]byte(route))
	if err != nil {
		t.Fatal(err)
	}
	if iface != "wlo1" {
		t.Fatalf("iface = %q", iface)
	}
	if _, err := defaultIface([]byte("Iface\tDestination\nlo\t00000000\n")); err == nil {
		t.Fatal("expected error when only lo has default route")
	}
	if _, err := defaultIface([]byte("garbage")); err == nil {
		t.Fatal("expected error for missing route table")
	}
}

func TestNetworkCollectTwoTicks(t *testing.T) {
	route := "Iface\tDestination\tGateway\neth0\t00000000\t0100000A\n"
	withReadFile(map[string]string{
		"/proc/net/route":                         route,
		"/sys/class/net/eth0/statistics/rx_bytes": "1000",
		"/sys/class/net/eth0/statistics/tx_bytes": "2000",
	}, t)
	c := &NetworkCollector{}
	first, err := c.Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Summary, "eth0") {
		t.Fatalf("first = %q", first.Summary)
	}
	if !strings.Contains(first.Summary, "0 B/s") {
		t.Fatalf("first tick should show zero rates: %q", first.Summary)
	}
}

func TestParseOSRelease(t *testing.T) {
	got := parseOSRelease([]byte("NAME=Omarchy\nPRETTY_NAME=\"Omarchy 4.0\"\n"))
	if got != "Omarchy 4.0" {
		t.Fatalf("got %q", got)
	}
	if got := parseOSRelease([]byte("NAME=Arch\n")); got != "Arch" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestParseUptime(t *testing.T) {
	if got := parseUptime([]byte("9000.00 8000.00")); got != "2h30m" {
		t.Fatalf("got %q", got)
	}
	if got := parseUptime([]byte("90000.00 1.00")); got != "1d 1h" {
		t.Fatalf("got %q", got)
	}
	if got := parseUptime([]byte("300.00 1.00")); got != "5m" {
		t.Fatalf("got %q", got)
	}
	if got := parseUptime([]byte("junk")); got != "" {
		t.Fatalf("malformed got %q", got)
	}
}

func TestSystemCollectReal(t *testing.T) {
	snap, err := (&SystemCollector{}).Collect(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Available || snap.Summary == "" {
		t.Fatalf("snap = %+v", snap)
	}
}
