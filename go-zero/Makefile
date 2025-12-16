ZeroPackageDir := ${GOPATH}/code/magic-lib/go-servicekit/go-zero
ZGrpcProtoHub := ./rpc/protohub
ZGrpcRpcCreateFolder := ./rpc
ZGrpcZRpcCreateFolder := ./zrpc
ModelGoDir := "./internal/model/db"
ModelMysqlDsn := ""

gen-update-package:
	go get github.com/magic-lib/go-servicekit@master
	go get github.com/magic-lib/go-plat-utils@master
	go mod tidy

gen-model:
	#@goctl template init --home $(modelGoDir)/tmpl
	@rm -rf $(ModelGoDir)/*_gen.go
	@goctl model mysql datasource \
    		--dir $(ModelGoDir) \
    		--table "*" \
    		--home $(ZeroPackageDir)/goctl-tmpl \
    		--url $(ModelMysqlDsn);

gen-zero-grpc:
	@for filename in $(ZGrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  goctl rpc protoc $${filename} --go_out=$(ZGrpcRpcCreateFolder) \
      		--go-grpc_out=$(ZGrpcRpcCreateFolder) \
      		--proto_path=$(ZGrpcProtoHub) \
      		--proto_path=$(ZeroPackageDir)/goctl-zrpc/protoc-29.2/include \
      		--zrpc_out=$(ZGrpcZRpcCreateFolder) \
      		--style go_zero; \
	done
	@echo "生成zero-grpc代码完成"

gen-grpc:
	@for filename in $(ZGrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  protoc --go_out=$(ZGrpcRpcCreateFolder) \
      		--go-grpc_out=$(ZGrpcRpcCreateFolder) \
      		--proto_path=$(ZGrpcProtoHub) \
      		--proto_path=$(ZeroPackageDir)/goctl-zrpc/protoc-29.2/include \
      		$${filename}; \
	done
	@echo "生成grpc代码完成"