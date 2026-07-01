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
	}

	for input, want := range tests {
		if got := normalizeMAC(input); got != want {
			t.Fatalf("normalizeMAC(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestParseMacOSARPMatchesSingleDigitOctets(t *testing.T) {
	t.Parallel()

	data := `? (192.168.1.10) at 0:1c:42:2e:60:4a on en0 ifscope [ethernet]
? (192.168.1.20) at 00:11:22:33:44:55 on en0 ifscope [ethernet]`

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
