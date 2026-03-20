.PHONY: generate build clean build-ApiFunction

generate:
	~/go/bin/oapi-codegen --config oapi-codegen.cfg.yaml openapi.yml

build:
	GOOS=linux GOARCH=arm64 go build -o bootstrap cmd/api/main.go

build-ApiFunction:
	GOOS=linux GOARCH=arm64 go build -o $(ARTIFACTS_DIR)/bootstrap cmd/api/main.go

clean:
	rm -f bootstrap
