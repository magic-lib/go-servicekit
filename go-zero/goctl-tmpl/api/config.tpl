package config

import {{.authImport}}

import (
	"github.com/magic-lib/go-plat-startupcfg/startupcfg"
	"github.com/zeromicro/go-zero/zrpc"
	"github.com/magic-lib/go-servicekit/tracer"
)

type Config struct {
	RestConf      rest.RestConf
	RpcServerConf zrpc.RpcServerConf // gRPC 服务端配置
    TraceConfig   tracer.TraceConfig
    MysqlConf     MysqlDatabase
    Log           logx.LogConf
    Prefix        string
	ServiceConfig *startupcfg.ConfigAPI
	{{.auth}}
	{{.jwtTrans}}
}

type MysqlDatabase struct { // 数据库配置
	Mysql           *startupcfg.MysqlConfig `json:"mysql" yaml:"mysql"`
	MaxOpenConn     int                     `json:"maxOpenConn" yaml:"max_open_conn"`
	MaxIdleConn     int                     `json:"maxIdleConn" yaml:"max_idle_conn"`
	ConnMaxLifetime string                  `json:"connMaxLifetime" yaml:"conn_max_lifetime"`
	ConnMaxIdleTime string                  `json:"connMaxIdleTime" yaml:"conn_max_idle_time"`
}

// InitStartConfig 初始化配置文件
func InitStartConfig(configFile string) (*Config, error) {
	//设置配置文件默认加解密
	_ = startupcfg.SetDefaultEncryptedHandler("")
	// 加载配置
	c := &Config{}
	startUpCfg, err := startupcfg.DecryptSecretByYamlFile(configFile, "", c)
	if err != nil {
		return c, err
	}
	c.ServiceConfig = startUpCfg
	return c, nil
}
