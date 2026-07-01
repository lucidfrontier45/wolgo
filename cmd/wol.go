package cmd

import (
	"fmt"
	"net"
	"strings"
)

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

// sendWOL sends a Wake-on-LAN magic packet to the specified MAC address.
func sendWOL(mac []byte) error {
	packet := createMagicPacket(mac)

	conn, err := net.Dial("udp", "255.255.255.255:9")
	if err != nil {
		return fmt.Errorf("failed to create connection: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	udpConn := conn.(*net.UDPConn)
	if _, err := udpConn.Write(packet); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}
