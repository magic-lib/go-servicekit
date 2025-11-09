mac:
	GOOS=darwin go build -ldflags="-s -w" -o ${GOPATH}/bin/sql2pb sql2pb.go
	$(if $(shell command -v upx || which upx), upx sql2pb)
