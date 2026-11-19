package cmd

import (
	"fmt"
	"net"
	"strings"
)

// wolPort is the UDP port Wake-on-LAN magic packets are sent to.
const wolPort = 9

// parseMAC parses a MAC address string and returns its byte representation.
// Supports formats: "00:11:22:33:44:55", "00-11-22-33-44-55", "001122334455".
func parseMAC(macStr string) ([]byte, error) {
	cleaned := strings.ReplaceAll(macStr, ":", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) != 12 {
		return nil, fmt.Errorf("invalid MAC address format: %s", macStr)
	}

	mac := make([]byte, 6)
	for i := range 6 {
		byteStr := cleaned[i*2 : i*2+2]
		b, err := parseHexByte(byteStr)
		if err != nil {
			return nil, fmt.Errorf("invalid MAC address: %s", macStr)
		}
		mac[i] = b
	}

	return mac, nil
}

// parseHexByte parses a 2-character hex string into a byte.
func parseHexByte(s string) (byte, error) {
	var result byte
	for i := range 2 {
		c := s[i]
		var val byte
		switch {
		case c >= '0' && c <= '9':
			val = c - '0'
		case c >= 'a' && c <= 'f':
			val = c - 'a' + 10
		case c >= 'A' && c <= 'F':
			val = c - 'A' + 10
		default:
			return 0, fmt.Errorf("invalid hex character: %c", c)
		}
		result = result<<4 | val
	}
	return result, nil
}

// createMagicPacket creates a Wake-on-LAN magic packet for the given MAC address.
// Format: 6 bytes of 0xFF followed by 16 repetitions of the MAC address.
func createMagicPacket(mac []byte) []byte {
	packet := make([]byte, 102)
	for i := range 6 {
		packet[i] = 0xFF
	}
	for i := range 16 {
		copy(packet[6+i*6:], mac)
	}
	return packet
}

// subnetBroadcast returns the directed broadcast address of ipn, or nil for
// non-IPv4 networks.
func subnetBroadcast(ipn *net.IPNet) net.IP {
	ip4 := ipn.IP.To4()
	if ip4 == nil || len(ipn.Mask) != 4 {
		return nil
	}
	b := make(net.IP, 4)
	for i := range 4 {
		b[i] = ip4[i] | ^ipn.Mask[i]
	}
	return b
}

// broadcastTargets returns the UDP addresses a WOL packet should be sent to:
// the subnet-directed broadcast address of every enabled, non-loopback IPv4
// interface, plus the global broadcast 255.255.255.255 as a fallback.
//
// Directed broadcasts are required for wake-ups routed through other
// segments (routers drop global broadcast), and each subnet broadcast is
// routed out of its own interface, covering multi-homed hosts.
func broadcastTargets() ([]*net.UDPAddr, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("enumerating network interfaces: %w", err)
	}

	targets := []*net.UDPAddr{{IP: net.IPv4bcast, Port: wolPort}}
	seen := map[string]bool{net.IPv4bcast.String(): true}

	for i := range ifaces {
		iface := ifaces[i]
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, fmt.Errorf("listing addresses of interface %s: %w", iface.Name, err)
		}
		for _, addr := range addrs {
			ipn, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			bcast := subnetBroadcast(ipn)
			if bcast == nil {
				continue
			}
			key := bcast.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			targets = append(targets, &net.UDPAddr{IP: bcast, Port: wolPort})
		}
	}
	return targets, nil
}

// sendWOL sends a Wake-on-LAN magic packet to the specified MAC address.
//
// The packet is broadcast via a single unbound UDP socket to every subnet
// broadcast address (the kernel routes each to its interface) and to the
// global broadcast. It reports success if at least one send succeeded and
// failure only if every send failed.
func sendWOL(mac []byte) error {
	packet := createMagicPacket(mac)

	targets, err := broadcastTargets()
	if err != nil {
		return err
	}

	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	var firstErr error
	sent := 0
	for _, target := range targets {
		if _, err := conn.WriteTo(packet, target); err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("sending to %s: %w", target, err)
			}
			continue
		}
		sent++
	}

	if sent == 0 {
		if firstErr != nil {
			return fmt.Errorf("failed to send packet: %w", firstErr)
		}
		return fmt.Errorf("failed to send packet: no reachable broadcast target")
	}
	return nil
}
