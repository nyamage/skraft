.PHONY: build test lint install release

build:
	go build -o /tmp/skraft .

test:
	go test ./...

lint:
	go vet ./...

install:
	go install .

release:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release VERSION=v0.1.0"; exit 1; fi
	git tag $(VERSION)
	git push origin $(VERSION)
