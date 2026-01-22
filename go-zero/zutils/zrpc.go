package zutils

import (
	"fmt"
	"github.com/magic-lib/go-plat-cache/cache"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/utils"
	"github.com/orcaman/concurrent-map/v2"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type ZRpcClientConfig struct {
	ClientCfg     zrpc.RpcClientConf
	ClientOptions []zrpc.ClientOption
	MaxSize       int
	MaxUsage      time.Duration
}

var (
	singletonGrpcClient = cmap.New[any]()
)

func NewPoolGrpcClient[T any](cfg *ZRpcClientConfig, fun func(clientConn *grpc.ClientConn) T) (func() T, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cfg is nil")
	}
	if fun == nil {
		return nil, fmt.Errorf("client constructor function 'fun' is nil")
	}
	typeKey, err := utils.GetGenericTypeId[T]()
	if err != nil {
		return nil, fmt.Errorf("failed to get type unique id: %v", err)
	}
	if singletonGrpcClient.Has(typeKey) {
		if existingVal, ok := singletonGrpcClient.Get(typeKey); ok {
			if oneSingleton, ok := existingVal.(func() T); ok {
				return oneSingleton, nil
			}
		}
		return nil, fmt.Errorf("singleton cache already exists for type %s", typeKey)
	}

	connPool := newZRPCConnPool(cfg.ClientCfg, cfg.MaxSize, cfg.MaxUsage, cfg.ClientOptions...)

	oneSingleton := func() T {
		var retConn *grpc.ClientConn
		connWrapper, err := connPool.Get()
		if err == nil && connWrapper != nil {
			retConn = connWrapper.Get()
		} else {
			retConn = new(grpc.ClientConn)
		}

		// 通过连接构造具体的gRPC客户端
		client := fun(retConn)
		return client
	}
	singletonGrpcClient.Set(typeKey, oneSingleton)
	return oneSingleton, nil
}

// newZRPCConnPool 创建go-zero gRPC连接池
// 参数：
//
//	cfg - RPC客户端配置
//	maxSize - 连接池最大连接数（<=0时使用默认值10）
//	maxUsage - 连接最长使用时间（0时使用默认值30秒）
//	options - 额外的gRPC客户端选项
//
// 返回：gRPC连接池实例
func newZRPCConnPool(cfg zrpc.RpcClientConf, maxSize int, maxUsage time.Duration, options ...zrpc.ClientOption) *cache.CommPool[*grpc.ClientConn] {
	// 设置默认值
	if maxSize <= 0 {
		maxSize = 10
	}
	if maxUsage == 0 {
		maxUsage = 30 * time.Second
	}

	// 连接健康检查函数
	checkConnHealth := func(conn *grpc.ClientConn) error {
		state := conn.GetState()
		// 仅当连接状态为非就绪/非连接中/非空闲时判定为异常
		if state != connectivity.Idle && state != connectivity.Connecting && state != connectivity.Ready {
			return fmt.Errorf("invalid connection state: %s (etcd: %s, endpoints: %s)",
				state.String(), conv.String(cfg.Etcd), conv.String(cfg.Endpoints))
		}
		return nil
	}

	// 连接关闭函数
	closeConn := func(conn *grpc.ClientConn) error {
		if conn == nil {
			return nil
		}
		return conn.Close()
	}
	// 创建连接池
	poolConfig := &cache.ResPoolConfig[*grpc.ClientConn]{
		MaxSize:  maxSize,
		MaxUsage: maxUsage,
		// 新建连接的函数
		New: func() (*grpc.ClientConn, error) {
			// 使用go-zero创建gRPC客户端
			client, err := zrpc.NewClient(cfg, options...)
			if err != nil {
				return nil, fmt.Errorf("failed to create zrpc client: %w (etcd: %s, endpoints: %s)",
					err, conv.String(cfg.Etcd), conv.String(cfg.Endpoints))
			}
			if client == nil || client.Conn() == nil {
				return nil, fmt.Errorf("zrpc client or its connection is nil (etcd: %s, endpoints: %s)",
					conv.String(cfg.Etcd), conv.String(cfg.Endpoints))
			}
			return client.Conn(), nil
		},
		CheckFunc: checkConnHealth,
		CloseFunc: closeConn,
	}
	return cache.NewResPool(poolConfig)
}
