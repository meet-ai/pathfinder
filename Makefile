.PHONY: build clean run-tw

BINARY_DIR := bin
MSG ?= test run

build:
	go build -o $(BINARY_DIR)/pathfinder ./cmd/pathfinder

clean:
	rm -rf $(BINARY_DIR)

# Run pathfinder via Termwright (requires: cargo install termwright)
run-tw:
	termwright run -- $(BINARY_DIR)/pathfinder -m "$(MSG)"

# TUI 验收：抓屏文本并校验关键文案（需 termwright + socat + jq）
tui-check: build
	./scripts/tui-termwright-text.sh
