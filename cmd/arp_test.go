package cmd

import (
	"net"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"00:11:22:33:44:55": "001122334455",
		"0:1c:42:2e:60:4a":  "001c422e604a",
		"00-11-22-33-44-55": "001122334455",
		"001122334455":      "001122334455",
		"bad-mac":           "",
		// Malformed inputs must be rejected, not rewritten into a
		// different valid MAC.
		"00:11:22:33:44:5 5": "", // 7 groups
		"0 0:11:33:44:55:66": "", // 7 groups
		"00:1122:33:44:55":   "", // 5 groups
		"0011.2233.4455":     "", // unsupported dotted form
		"":                   "",
	}

	for input, want := range tests {
		if got := normalizeMAC(input); got != want {
			t.Fatalf("normalizeMAC(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizeMACToleratesEmptyGroupBoundaries(t *testing.T) {
	t.Parallel()

	// FieldsFunc collapses repeated separators and drops trailing ones;
	// a trailing colon therefore still yields 6 valid groups.
	if got := normalizeMAC("00:11:22:33:44:55:"); got != "001122334455" {
		t.Fatalf("normalizeMAC(trailing colon) = %q, want %q", got, "001122334455")
	}
}

func TestParseMacOSARPMatchesSingleDigitOctets(t *testing.T) {
	t.Parallel()

	data := "? (192.168.1.10) at 0:1c:42:2e:60:4a on en0 ifscope [ethernet]\n" +
		"? (192.168.1.20) at 00:11:22:33:44:55 on en0 ifscope [ethernet]"

	ips, err := parseMacOSARP(data, "001c422e604a")
	if err != nil {
		t.Fatalf("parseMacOSARP() error = %v", err)
	}
	if len(ips) != 1 {
		t.Fatalf("len(ips) = %d, want 1", len(ips))
	}
	if !ips[0].Equal(net.ParseIP("192.168.1.10")) {
		t.Fatalf("ips[0] = %v, want 192.168.1.10", ips[0])
	}
}

func TestParseMacOSARPHandlesIPv6ZoneSuffix(t *testing.T) {
	t.Parallel()

	// macOS arp -a emits IPv6 link-local entries with a %zone suffix that
	// net.ParseIP rejects; the entry must not be silently dropped.
	data := "? (fe80::1%en0) at 0:1c:42:2e:60:4a on en0 ifscope [ethernet]\n" +
		"? (192.168.1.10) at 0:1c:42:2e:60:4a on en0 ifscope [ethernet]"

	ips, err := parseMacOSARP(data, "001c422e604a")
	if err != nil {
		t.Fatalf("parseMacOSARP() error = %v", err)
	}
	if len(ips) != 2 {
		t.Fatalf("len(ips) = %d, want 2", len(ips))
	}
	if got, want := ips[0].String(), "fe80::1"; got != want {
		t.Fatalf("ips[0] = %q, want %q", got, want)
	}
}

func TestParseWindowsARPIgnoresHeaders(t *testing.T) {
	t.Parallel()

	data := "Interface: 192.168.1.2 --- 0x3\r\n" +
		"  Internet Address      Physical Address      Type\r\n" +
		"  192.168.1.1           00-11-22-33-44-55     dynamic\r\n"

	ips, err := parseWindowsARP(data, "001122334455")
	if err != nil {
		t.Fatalf("parseWindowsARP() error = %v", err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.ParseIP("192.168.1.1")) {
		t.Fatalf("ips = %v, want [192.168.1.1]", ips)
	}
}

func TestSubnetBroadcast(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cidr string
		want string
	}{
		{"172.19.185.220/20", "172.19.191.255"},
		{"192.168.1.10/24", "192.168.1.255"},
		{"10.0.0.5/8", "10.255.255.255"},
	}
	for _, tc := range tests {
		_, ipn, err := net.ParseCIDR(tc.cidr)
		if err != nil {
			t.Fatalf("ParseCIDR(%q) error = %v", tc.cidr, err)
		}
		got := subnetBroadcast(ipn)
		if got == nil || got.String() != tc.want {
			t.Fatalf("subnetBroadcast(%q) = %v, want %s", tc.cidr, got, tc.want)
		}
	}

	_, ipn, err := net.ParseCIDR("fe80::1/64")
	if err != nil {
		t.Fatalf("ParseCIDR error = %v", err)
	}
	if got := subnetBroadcast(ipn); got != nil {
		t.Fatalf("subnetBroadcast(ipv6) = %v, want nil", got)
	}
}

func TestBroadcastTargetsAreValidIPv4(t *testing.T) {
	t.Parallel()

	targets, err := broadcastTargets()
	if err != nil {
		t.Fatalf("broadcastTargets() error = %v", err)
	}
	if len(targets) == 0 {
		t.Fatal("broadcastTargets() returned no targets")
	}
	for _, target := range targets {
		if target.Port != wolPort {
			t.Fatalf("target %s port = %d, want %d", target, target.Port, wolPort)
		}
		if target.IP.To4() == nil {
			t.Fatalf("target %s is not IPv4", target)
		}
	}
}
