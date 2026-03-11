# wolgo

A simple Wake-on-LAN (WoL) CLI tool written in Go.

## Usage

```bash
wolgo <MAC_ADDRESS>
```

### Examples

```bash
wolgo 00:11:22:33:44:55
```

## Building

```bash
go build
```

## Development

Run linting:
```bash
golangci-lint run --fix
```