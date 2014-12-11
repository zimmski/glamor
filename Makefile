.PHONY: clean fmt install lint

clean:
	go clean github.com/zimmski/glamor
fmt:
	gofmt -l -w -tabs=true .
install:
	go install github.com/zimmski/glamor
lint:
	errcheck github.com/zimmski/glamor/...
	golint ./...
	go tool vet -all=true -v=true .

