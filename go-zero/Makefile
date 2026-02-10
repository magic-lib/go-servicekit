ZeroTemplateDir := ${GOPATH}/code/magic-lib/go-servicekit/go-zero
ModelGoDir := "./demo/model/db"
ModelMysqlDsn := "root:mjhttyryt565-jyjh5824t-p55w@tcp(192.168.2.84:10365)/allinone-account"
GoCtlPath := "${GOPATH}/bin"

############### api-init-start ###############
APIGoProjectName := "account"
APIGoProjectDir := "./demo"

############### api-init-end  ###############

gen-update-package:
	go install github.com/zeromicro/go-zero/tools/goctl@latest
	go install github.com/zeromicro/goctl-swagger@latest
	#go get github.com/magic-lib/go-servicekit@master
	go get github.com/magic-lib/go-plat-utils@master
	go install github.com/Mikaelemmmm/sql2pb@latest
	#https://github.com/protocolbuffers/protobuf/releases
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go mod tidy

gen-zero-init:
	@goctl api new $(APIGoProjectDir)/$(APIGoProjectName) --home $(ZeroTemplateDir)/goctl-tmpl --style gozero
	@sudo cp -rf ./static-file/* $(APIGoProjectDir)/$(APIGoProjectName)
	@sudo cp -rf ./static-file/Dockerfile $(APIGoProjectDir)/$(APIGoProjectName)
	@sudo cp -rf ./static-file/.gitlab-ci.yml $(APIGoProjectDir)/$(APIGoProjectName)
	$(MAKE) gen-model;

gen-model:
	#@goctl template init --home $(modelGoDir)/tmpl  ##生成模版到文件夹中
	@rm -rf $(ModelGoDir)/*_gen.go
	@$(GoCtlPath)/goctl model mysql datasource \
    		--dir $(ModelGoDir) \
    		--table "*" \
    		--home $(ZeroTemplateDir)/goctl-tmpl \
    		--url $(ModelMysqlDsn) --style go_zero;


GrpcProtoHub := ./goctl-zrpc/protohub
GrpcProtoDatabaseName := allinone-credit
GrpcProtoServiceName := CreditServer
GrpcProtoGoPackage := ./creditpb
GrpcProtoPackage := creditpb
GrpcProtoCreateFileName := credit_server
GrpcRpcGoCreateFolder := ./demo/rpc

gen-mysql-proto:
	@mkdir -p $(GrpcProtoHub);
	@sql2pb \
		-db "mysql" \
		-dsn $(ModelMysqlDsn) \
		-schema $(GrpcProtoDatabaseName) \
		-table "*" \
		-service_name $(GrpcProtoServiceName) \
		-go_package $(GrpcProtoGoPackage) \
		-package $(GrpcProtoPackage) -field_style "sql_pb" > $(GrpcProtoHub)/$(GrpcProtoCreateFileName).proto;

gen-grpc:
	@for filename in $(GrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  mkdir -p $(GrpcRpcGoCreateFolder); \
	  protoc $${filename} --go_out=$(GrpcRpcGoCreateFolder) \
      		--go-grpc_out=$(GrpcRpcGoCreateFolder) \
      		--proto_path=$(GrpcProtoHub) \
      		--proto_path=$(ZeroTemplateDir)/goctl-zrpc/protoc-33.4/include; \
	done
	@cp -f $(ZeroTemplateDir)/goctl-tmpl/rpc/.gitlab-ci.yml $(GrpcRpcGoCreateFolder);
	@echo "生成grpc代码完成"

ZGrpcZRpcCreateFolder := ./demo/zrpc

gen-zero-grpc:
	@echo "正在生成zero-grpc代码"
	@for filename in $(GrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  goctl rpc protoc $${filename} --go_out=$(GrpcRpcGoCreateFolder) \
			--go-grpc_out=$(GrpcRpcGoCreateFolder) \
			--proto_path=$(GrpcProtoHub) \
			--proto_path=$(ZeroTemplateDir)/goctl-zrpc/protoc-33.4/include \
			--zrpc_out=$(ZGrpcZRpcCreateFolder) \
			--style go_zero; \
	done
	@cp -f $(ZeroTemplateDir)/goctl-tmpl/rpc/.gitlab-ci.yml $(GrpcRpcGoCreateFolder);
	@echo "生成zero-grpc代码完成"

gen-zero-grpc-all:
	@MAKE gen-mysql-proto;
	@MAKE gen-zero-grpc;


krotos-go-install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

krotos-init:
	@cd demo && kratos new helloworld3