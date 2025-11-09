package oauth2

//import (
//	"github.com/go-oauth2/oauth2/v4/errors"
//	"github.com/go-oauth2/oauth2/v4/manage"
//	"github.com/go-oauth2/oauth2/v4/server"
//	"github.com/go-oauth2/oauth2/v4/store"
//	oredis "github.com/go-oauth2/redis/v4"
//	"github.com/go-redis/redis/v8"
//	"log"
//	"net"
//)
//
//// initOAUTH 初始化，token存储到redis中，客户端存储到mysql中
//func GetOauthServer(redisConn *database.DBConnect, clientStore oauth2.ClientStore) *server.Server {
//	manager := GetOauthManager(redisConn, clientStore)
//
//	srv := server.NewDefaultServer(manager)
//	srv.SetAllowGetAccessRequest(true)
//	srv.SetClientInfoHandler(server.ClientFormHandler)
//
//	srv.SetInternalErrorHandler(func(err error) (re *errors.Response) {
//		log.Println("Internal Error:", err.Error())
//		return
//	})
//
//	srv.SetResponseErrorHandler(func(re *errors.Response) {
//		log.Println("Response Error:", re.Error.Error())
//		return
//	})
//	return srv
//}
//
//// GetOauthManager
//func GetOauthManager(redisConn *database.DBConnect, clientStore oauth2.ClientStore) oauth2.Manager {
//	manager := manage.NewDefaultManager()
//	if redisConn == nil {
//		// token memory store
//		manager.MustTokenStorage(store.NewMemoryTokenStore())
//	} else {
//		db := utils.ToInt64(redisConn.Database)
//		dbInt := int(db)
//		manager.MapTokenStorage(oredis.NewRedisStore(&redis.Options{
//			Addr: net.JoinHostPort(redisConn.Host, redisConn.Port),
//			DB:   dbInt,
//		}))
//	}
//
//	//用户列表的查询方式
//	manager.MapClientStorage(clientStore)
//
//	return manager
//}
