.PHONY: generate build clean

generate:
	~/go/bin/oapi-codegen --config oapi-codegen.cfg.yaml openapi.yml

build: generate
	GOOS=linux GOARCH=arm64 go build -o bootstrap cmd/api/main.go

clean:
	rm -f bootstrap
