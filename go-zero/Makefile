ZeroTemplateDir := ${GOPATH}/code/magic-lib/go-servicekit/go-zero
ZGrpcProtoHub := ./goctl-zrpc/protohub
ZGrpcRpcCreateFolder := ./demo/rpc
ZGrpcZRpcCreateFolder := ./demo/zrpc
ModelGoDir := "./demo/model/db"
ModelMysqlDsn := "root:mjhttyryt565-jyjh5824t-p55w@tcp(192.168.2.84:10365)/account-server"
GoCtlPath := "${GOPATH}/bin"

############### api-init-start ###############
APIGoProjectName := "account"
APIGoProjectDir := "./demo"

############### api-init-end  ###############

gen-update-package:
	go install github.com/zeromicro/go-zero/tools/goctl@latest
	go install github.com/zeromicro/goctl-swagger@latest
	go get github.com/magic-lib/go-servicekit@master
	go get github.com/magic-lib/go-plat-utils@master
	go install github.com/Mikaelemmmm/sql2pb@latest
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

#go_package go语言中的包路径和包名
#package 是proto本身的包名
#		-user "root" \
#		-password "mjhttyryt565-jyjh5824t-p55w" \
#		-host "192.168.2.84" \
#		-port "10365" \
#		-schema "account-server" \


gen-zero-grpc:
	@sql2pb \
		-db "mysql" \
		-dsn $(ModelMysqlDsn) \
		-schema "account-server" \
		-table "*" \
		-service_name "AccountServer" \
		-go_package "./commpb" \
		-package "commpb" -field_style "sql_pb" > $(ZGrpcProtoHub)/account-server.proto;
	@echo "正在生成zero-grpc代码"
	@for filename in $(ZGrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  goctl rpc protoc $${filename} --go_out=$(ZGrpcRpcCreateFolder) \
      		--go-grpc_out=$(ZGrpcRpcCreateFolder) \
      		--proto_path=$(ZGrpcProtoHub) \
      		--proto_path=$(ZeroTemplateDir)/goctl-zrpc/protoc-29.2/include \
      		--zrpc_out=$(ZGrpcZRpcCreateFolder) \
      		--style go_zero; \
	done
	@echo "生成zero-grpc代码完成"

gen-grpc:
	@for filename in $(ZGrpcProtoHub)/*.proto; do \
	  echo "正在处理文件: $${filename}"; \
	  mkdir -p $(ZGrpcRpcCreateFolder); \
	  protoc $${filename} --go_out=$(ZGrpcRpcCreateFolder) \
      		--go-grpc_out=$(ZGrpcRpcCreateFolder) \
      		--proto_path=$(ZGrpcProtoHub) \
      		--proto_path=$(ZeroTemplateDir)/goctl-zrpc/protoc-29.2/include; \
	done
	@echo "生成grpc代码完成"