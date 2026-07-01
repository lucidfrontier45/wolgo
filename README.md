# wolgo

Wake-on-LAN and network utilities CLI tool.

Send magic packets, resolve MAC addresses via ARP cache, and manage an
alias registry so you never type a MAC by hand again.

## Install

```bash
go install github.com/lucidfrontier45/wolgo@latest
```

Or build from source:

```bash
git clone https://github.com/lucidfrontier45/wolgo
cd wolgo
go build
```

## Usage

```
wolgo [MAC_ADDRESS | alias] [--all]
wolgo register <alias> <MAC_ADDRESS>
wolgo list
wolgo remove <alias>
wolgo find-ip [MAC_ADDRESS | alias] [--all]
```

### Send WOL packet

By MAC:

```bash
wolgo 00:11:22:33:44:55
```

By alias:

```bash
wolgo office-pc
```

To every registered target:

```bash
wolgo --all
```

### Register an alias

```bash
wolgo register office-pc 00:11:22:33:44:55
```

Saved to `~/.wolgo/targets.json`. Overwrites existing alias silently.

Alias rules: no spaces, no tabs, must not look like a MAC address.

### List registered aliases

```bash
wolgo list
```

Output: one `alias -> mac` per line, sorted alphabetically.

### Remove an alias

```bash
wolgo remove office-pc
```

### Find IP from MAC address

Scans system ARP cache. Supports Linux, macOS, Windows.

By MAC:

```bash
wolgo find-ip 00:11:22:33:44:55
```

By alias:

```bash
wolgo find-ip office-pc
```

List all registered targets (no ARP scan):

```bash
wolgo find-ip --all
```

### Precedence

If an argument is a valid MAC address, it is always treated as a MAC, even
if an alias with the same string exists. This prevents a MAC-formatted alias
from shadowing direct MAC input.

## Development

```bash
golangci-lint run --fix   # lint + format
go test ./...             # test
go build ./...            # build
```
