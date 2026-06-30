# TODO: 2026-07-01-find-ip-subcommand

## Phase 1: Add Cobra dependency
- [x] Run `go get github.com/spf13/cobra`

## Phase 2: Create arp.go — cross-platform ARP scanner
- [x] Write arp.go with Linux (/proc/net/arp), macOS (arp -a), Windows (arp -a) support

## Phase 3: Create cmd/findip.go — find-ip subcommand handler
- [x] Write cmd/findip.go using Cobra

## Phase 4: Create cmd/root.go — root command with backward compat
- [x] Write cmd/root.go with MAC-as-arg fallback to WOL

## Phase 5: Rewrite main.go — entrypoint stub
- [x] Rewrite main.go to call cmd.Execute()

## Phase 6: go mod tidy + lint
- [x] Run `go mod tidy`
- [x] Run `golangci-lint run --fix`
- [x] Run `go build ./...` to verify
