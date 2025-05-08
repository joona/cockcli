.PHONY: check
check:
	go test ./...

.PHONY: install
install: check
	go install github.com/joona/cockcli/cmd/cockcli
