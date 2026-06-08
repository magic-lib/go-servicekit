mac:
	GOOS=darwin go build -ldflags="-s -w" -o ${GOPATH}/bin/embed generate_assets.go
	$(if $(shell command -v upx || which upx), upx embed-darwin)
