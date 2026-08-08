.PHONY: build test clean

COMMANDS := $(notdir $(wildcard cmd/*))

build:
	@mkdir -p .build
	@for command in $(COMMANDS); do \
		echo "building $$command"; \
		CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ".build/$$command/bootstrap" "./cmd/$$command" || exit 1; \
	done

test:
	go test ./...

clean:
	rm -rf .build
