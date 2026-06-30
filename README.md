# wolgo

Wake-on-LAN and network utilities CLI tool.

## Usage

```bash
wolgo [MAC_ADDRESS | command]
```

### Send WOL packet

```bash
wolgo 00:11:22:33:44:55
```

### Find IP from MAC address

Scans the system ARP cache for matching IP addresses. Supports Linux, macOS, Windows.

```bash
wolgo find-ip 00:11:22:33:44:55
```

## Building

```bash
go build
```

## Development

```bash
golangci-lint run --fix
```
