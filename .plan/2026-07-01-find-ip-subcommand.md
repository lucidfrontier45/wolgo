# Find IP from MAC (find-ip subcommand)

## Goal
Add `wolgo find-ip <MAC>` subcommand that reads the system ARP cache and prints all IP addresses associated with the given MAC address. Uses Cobra for CLI argument parsing.

## Background
- Current `wolgo` is a flat Go module (`main.go`) with a single WOL-magic-packet function
- No CLI framework — manual `os.Args` parsing
- Need: cross-platform ARP cache reading (Linux: `/proc/net/arp`, macOS/Windows: `arp -a`)
- Must work non-root

## Approach

### 1. Add Cobra dependency
```
go get github.com/spf13/cobra
```

### 2. Restructure to `cmd/` layout

| File | Action | Purpose |
|---|---|---|
| `main.go` | Rewrite | Stub that calls `cmd.Execute()` |
| `cmd/root.go` | New | Root Cobra command; handles `wolgo <MAC>` backward compat |
| `cmd/findip.go` | New | `wolgo find-ip <MAC>` handler |
| `arp.go` | New | Cross-platform ARP scanner logic |
| `go.mod` | Updated | Cobra + pflag deps |

### 3. ARP scanner (`arp.go`)
- Runtime platform detection (`runtime.GOOS`)
- **Linux**: parse `/proc/net/arp` (tab-separated: IP, HW type, flags, HW addr, mask, device)
- **macOS**: exec `arp -a`, parse `? (192.168.1.10) at 00:11:22:33:44:55 on en0 ...`
- **Windows**: exec `arp -a`, parse `192.168.1.10   00-11-22-33-44-55    dynamic`
- Normalize MAC: strip separators, lowercase — match against normalized input
- Return `[]net.IP`

### 4. CLI routing (`cmd/root.go`, `cmd/findip.go`)
- `wolgo <MAC>` — root command Run, route to existing WOL logic
- `wolgo find-ip <MAC>` — Cobra subcommand, calls ARP scanner, prints IPs line-separated
- `wolgo` with no args → usage text

### 5. Backward compat
- `wolgo 00:11:22:33:44:55` still sends WOL magic packet (root command detects MAC-like arg)
- No explicit `wol` subcommand

## Trade-offs
- **`cmd/` layout** over flat + manual routing: Cobra gives arg validation, autogen help, scales to more subcommands. Minor restructure cost.
- **Root-level `arp.go`** over `pkg/arp/`: keeps code close to usage, avoids package nesting for one file.
- **Runtime platform check** over build tags: single binary works everywhere.
- **Silent on no match**: exit 0, no output. Clean for piped usage.

## Files changed/created
- `main.go` (rewrite)
- `cmd/root.go` (new)
- `cmd/findip.go` (new)
- `arp.go` (new)
- `go.mod` / `go.sum` (updated)

## Next step
Implement: `arp.go` → `cmd/findip.go` → `cmd/root.go` → `main.go` rewrite → `go mod tidy` → `golangci-lint run --fix`
