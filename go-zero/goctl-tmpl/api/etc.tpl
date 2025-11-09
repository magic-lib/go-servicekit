RestConf: # http服务配置
  Name: {{.serviceName}}-http
  Host: {{.host}}
  Port: {{.port}}
  Timeout: 60000
  MaxBytes: 10485760 # 设置为10MB，可根据需要调整
RpcServerConf: # rpc服务端配置
  Name: {{.serviceName}}-rpc
  ListenOn: 0.0.0.0:10702 # gRPC 服务监听地址
  Mode: dev
Prefix: /{{.serviceName}}/api/v1

TraceConfig:
  Namespace: "namespace"
  ServiceName: "{{.serviceName}}"
  Endpoint: "" #http://192.168.2.84:14268/api/traces
  SamplerPercent: 50

MysqlConf: # 数据库配置
  mysql:
    protocol: tcp
    username: root
    pwEncoded: 5904da17d89ac224ae47283d9dd4b5c81758e3d50e4f76138c0c13a91464ec8e
    address: 192.168.2.84:10365
    database: zamloan2-collection
    charset: utf8mb4
  MaxOpenConns: 50        # 最大连接数
  MaxIdleConns: 10        # 最大空闲连接数
  ConnMaxLifetime: 500s   # 单个连接的最大生命周期（秒）
  ConnMaxIdleTime: 200s   # 空闲连接最大保留时间（秒）

Log: # 日志配置
  ServiceName: {{.serviceName}}-log
  Mode: file    # 支持 console（控制台）或 file（文件）
  Path: logs
  Level: info   # 日志级别，支持 debug、info、warn、error
  Compress: true
  KeepDays: 7   # 日志保留天数
  MaxBackups: 3
  MaxSize: 100

api:
  manager-server:
    domain: http://localhost:8080
    auth:
      X-Internal-Token: 1a47858d12d67396551c311d4b33bdd10a90c77bea12846ad0ee46c75097c5c82a6baf74282ca2ff958105db73cd2874
    urls:
      ListRoleByRoleType: /manager/api/v1/listRole
custom:
  normal:
    collectionManagerUserNameList: [ "admin" ]
    collectionManagerRoleName: "collection-manager"

