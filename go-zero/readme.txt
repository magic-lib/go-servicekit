gen-update-package:
	go get github.com/magic-lib/go-plat-utils@master
	go mod tidy

gen-model:
	#@goctl template init --home ./internal/model/db/tmpl
	@rm -rf ./internal/model/db/*_gen.go
	@goctl model mysql datasource \
    		--dir ./internal/model/db/ \
    		--table "*" \
    		--home ./goctl-tmpl \
    		--url "root:xxxxxx@tcp(xxxxxxx:xx)/xxxxxx";


protohub := ./rpc/protohub
grpcCreateFolder := ./rpc
zGrpcCreateFolder := ./zrpc

gen-grpc:
	@for filename in $(protohub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  goctl rpc protoc $${filename} --go_out=$(grpcCreateFolder) \
      		--go-grpc_out=$(grpcCreateFolder) \
      		--proto_path=$(protohub) \
      		--proto_path=${GOPATH}/pkg/mod/protoc-29.2-linux-x86_64/include \
      		--zrpc_out=$(zGrpcCreateFolder) \
      		--style go_zero; \
	done