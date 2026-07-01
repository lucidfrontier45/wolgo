package cmd

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// FindIPsFromMAC scans the system ARP cache for all IP addresses associated
// with the given MAC address. It supports Linux, macOS, and Windows.
func FindIPsFromMAC(macStr string) ([]net.IP, error) {
	targetMAC := normalizeMAC(macStr)
	if targetMAC == "" {
		return nil, fmt.Errorf("invalid MAC address: %s", macStr)
	}

	switch runtime.GOOS {
	case "linux":
		return findIPsFromMACLinux(targetMAC)
	case "darwin":
		return findIPsFromMACMacOS(targetMAC)
	case "windows":
		return findIPsFromMACWindows(targetMAC)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// macStripReplacer strips common MAC address separators.
var macStripReplacer = strings.NewReplacer(":", "", "-", "", " ", "")

// normalizeMAC strips separators, lowercases, and zero-pads 1-digit octets.
func normalizeMAC(mac string) string {
	mac = strings.TrimSpace(mac)
	if !strings.ContainsAny(mac, ":- ") {
		compact := strings.ToLower(mac)
		if len(compact) != 12 {
			return ""
		}
		for _, r := range compact {
			if !strings.ContainsRune("0123456789abcdef", r) {
				return ""
			}
		}
		return compact
	}

	parts := strings.FieldsFunc(mac, func(r rune) bool {
		return r == ':' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.Grow(12)

	for _, part := range parts {
		if len(part) == 0 || len(part) > 2 {
			return ""
		}
		if len(part) == 1 {
			builder.WriteByte('0')
		}
		for _, r := range part {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return ""
			}
		}
		builder.WriteString(strings.ToLower(part))
	}

	if builder.Len() == 12 {
		return builder.String()
	}

	compact := macStripReplacer.Replace(mac)
	compact = strings.ToLower(compact)
	if len(compact) != 12 {
		return ""
	}
	for _, r := range compact {
		if !strings.ContainsRune("0123456789abcdef", r) {
			return ""
		}
	}
	return compact
}

// findIPsFromMACLinux parses /proc/net/arp on Linux.
func findIPsFromMACLinux(targetMAC string) ([]net.IP, error) {
	data, err := readFile("/proc/net/arp")
	if err != nil {
		return nil, fmt.Errorf("reading /proc/net/arp: %w", err)
	}

	return parseLinuxARP(data, targetMAC)
}

// parseLinuxARP parses the content of /proc/net/arp.
func parseLinuxARP(data, targetMAC string) ([]net.IP, error) {
	lines := strings.Split(data, "\n")
	var ips []net.IP

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Skip header line
		if fields[0] == "IP" || fields[0] == "ip" {
			continue
		}

		hwAddr := fields[3]
		if normalizeMAC(hwAddr) == targetMAC {
			ip := net.ParseIP(fields[0])
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}

// findIPsFromMACMacOS parses output of `arp -a` on macOS.
func findIPsFromMACMacOS(targetMAC string) ([]net.IP, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("executing arp -a: %w", err)
	}

	return parseMacOSARP(string(output), targetMAC)
}

// parseMacOSARP parses the output of `arp -a` on macOS.
// Format: ? (192.168.1.1) at 00:11:22:33:44:55 on en0 ifscope [ethernet]
func parseMacOSARP(data, targetMAC string) ([]net.IP, error) {
	lines := strings.Split(data, "\n")
	var ips []net.IP

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract IP from parentheses: ? (192.168.1.1) at ...
		ipStart := strings.Index(line, "(")
		ipEnd := strings.Index(line, ")")
		if ipStart == -1 || ipEnd == -1 || ipEnd <= ipStart {
			continue
		}
		ipStr := line[ipStart+1 : ipEnd]

		// Extract MAC after "at " and before " on " or " "
		atIdx := strings.Index(line, " at ")
		if atIdx == -1 {
			continue
		}
		macPart := line[atIdx+4:]

		// MAC ends at first space or " on "
		macEnd := strings.Index(macPart, " ")
		var macStr string
		if macEnd != -1 {
			macStr = macPart[:macEnd]
		} else {
			macStr = macPart
		}

		if normalizeMAC(macStr) == targetMAC {
			ip := net.ParseIP(ipStr)
			if ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips, nil
}

// findIPsFromMACWindows parses output of `arp -a` on Windows.
func findIPsFromMACWindows(targetMAC string) ([]net.IP, error) {
	cmd := exec.Command("arp", "-a")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("executing arp -a: %w", err)
	}

	return parseWindowsARP(string(output), targetMAC)
}

// parseWindowsARP parses the output of `arp -a` on Windows.
// Format:
//
//	Internet Address    Physical Address    Type
//	192.168.1.1         00-11-22-33-44-55   dynamic
func parseWindowsARP(data, targetMAC string) ([]net.IP, error) {
	lines := strings.Split(data, "\n")
	var ips []net.IP

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		// Skip non-data lines: headers, interface lines
		ip := net.ParseIP(fields[0])
		if ip == nil {
			continue
		}

		if normalizeMAC(fields[1]) == targetMAC {
			ips = append(ips, ip)
		}
	}

	return ips, nil
}

// readFile is a thin wrapper around os.ReadFile.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
