package oauth2

/*

oauth.StartGinOAuthServer(router, oauth.OauthConfig)


授权码模式（authorization code）
1、http://localhost:8083/oauth2/authorize?grant_type=authorization_code&scope=aaaa&client_id=aaaa&
response_type=code&redirect_uri=http://localhost/aaa
2、http://localhost/aaa?code=PKGBRPYUOWK_IRJIWXHPNA
3、http://localhost:8083/oauth2/token?grant_type=authorization_code&scope=aaaa&client_id=aaaa&
client_secret=827ccb0eea8a706c4c34a16891f84e7b&code=PKGBRPYUOWK_IRJIWXHPNA&redirect_uri=http://localhost/aaa
{
    "access_token": "UCIXDWWKNGAGAOTCNS_KVW",
    "expires_in": 7200,
    "refresh_token": "K0F-4RK-UCCZBOAR7RMM4G",
    "scope": "insert_from",
    "token_type": "Bearer"
}

简化模式（implicit）
不需要第三方服务器
如果用户登录获取是通过别的code方式获取到的话，则用下面方式,此code为rtx登录以后返回的code
1、http://localhost:8083/oauth2/authorize?grant_type=authorization_code&scope=plat_ulink&client_id=plat_ulink&
response_type=token&redirect_uri=http://localhost/aaa&code=xxxxxxxxxxxxx
注意，这里response_type=token
无需传递client secret，传递client_id只是为了验证在auth server配置的redirect_uri是否一致
redirect_uri中如果携带参数，则最好对url编码再作为参数传递过去
2、http://localhost/aaa#access_token=UQWIF1Y0NP2_LXFYF55JUQ&expires_in=3600&scope=plat_ulink&token_type=Bearer


客户端模式（client credentials）(主要用于api认证，跟用户无关)
1、http://localhost:8083/oauth2/token?grant_type=client_credentials&client_id=aaaa&
client_secret=827ccb0eea8a706c4c34a16891f84e7b&scope=aaaa
{
    "access_token": "BK7MO2DEMIE3SV9WRBVHJG",
    "expires_in": 7200,
    "scope": "aaaa",
    "token_type": "Bearer"
}
2、接口访问
http://localhost:8083/oauth2/read
Authorization: Bearer BK7MO2DEMIE3SV9WRBVHJG
{
    "ClientID": "000000",
    "UserID": "",
    "RedirectURI": "",
    "Scope": "insert_from",
    "Code": "",
    "CodeCreateAt": "0001-01-01T00:00:00Z",
    "CodeExpiresIn": 0,
    "Access": "VR37N7MKO2UX6M0VHIJVAA",
    "AccessCreateAt": "2021-03-10T15:17:15.419168+08:00",
    "AccessExpiresIn": 7200000000000,
    "Refresh": "",
    "RefreshCreateAt": "0001-01-01T00:00:00Z",
    "RefreshExpiresIn": 0
}


刷新token消息
http://localhost:8083/oauth2/token?grant_type=refresh_token&client_id=aaaa&
client_secret=827ccb0eea8a706c4c34a16891f84e7b&scope=aaaa&refresh_token=6S3C0HQZVJWAETDLA5OMLQ
说明：token被刷新以后，前面的token就用不了了
*/

//// GinOauthOption
//type GinOauthOption struct {
//	RouteFrontPath           string                          //路径的前缀，比如需要加上/v1/等等
//	ClientStore              oauth2.ClientStore              //client存储在mysql中 必传
//	UserAuthorizationHandler server.UserAuthorizationHandler //获取用户的信息的接口 必传
//	RedisConnect             *database.DBConnect             //token存储在redis中
//	ClientAuthorizedHandler  server.ClientAuthorizedHandler  //是否允许该客户端使用authorization_code或 __implicit 功能，
//	// 如果不设置，则会使用ClientScopeHandler对scope范围进行判断
//	ClientScopeHandler     server.ClientScopeHandler     //客户端传进来的scope是否正确的判断
//	AuthorizeScopeHandler  server.AuthorizeScopeHandler  //User传进来的scope是否正确的判断
//	ExtensionFieldsHandler server.ExtensionFieldsHandler //返回token信息时，可扩展展示一些信息，比如用户名
//	ErrorHandleFunc        ginserver.ErrorHandleFunc     //HandleTokenVerify 如果验证出错的话，
//	// 怎么处理, 默认全局处理
//	DefaultAuthorizeCodeTokenCfg *manage.Config                           //token过期时间的默认设置
//	DefaultClientTokenCfg        *manage.Config                           //设置Client过期时间和refreash
//	ReadUserCallbackHandler      func(token oauth2.TokenInfo) interface{} //read个人信息时，对个人信息进行特殊处理后输出
//}
//
//func initGinOAuthServer(oauthConfig *GinOauthOption) *server.Server {
//	manager := manage.NewDefaultManager()
//	isSetRedis := false
//	if oauthConfig.RedisConnect != nil {
//		db := utils.ToInt64(oauthConfig.RedisConnect.Database)
//		dbInt := int(db)
//		storeTemp, err := v3redis.NewTokenStore(&v3redis.Config{
//			Addr:     net.JoinHostPort(oauthConfig.RedisConnect.Host, oauthConfig.RedisConnect.Port),
//			Password: oauthConfig.RedisConnect.Password,
//			DB:       dbInt,
//		})
//
//		if err != nil {
//			log.Println("init Redis Error:", oauthConfig.RedisConnect)
//			log.Println(err)
//		}
//
//		if err == nil {
//			isSetRedis = true
//			manager.MapTokenStorage(storeTemp)
//		}
//	}
//
//	if !isSetRedis {
//		storyDefault, err := store.NewMemoryTokenStore()
//		if err != nil {
//			log.Println(err)
//			return nil
//		}
//		manager.MapTokenStorage(storyDefault)
//	}
//
//	//用户列表的查询方式
//	manager.MapClientStorage(oauthConfig.ClientStore)
//
//	// Initialize the oauth2 service
//	servers := ginserver.InitServer(manager)
//	ginserver.SetAllowGetAccessRequest(true)
//	ginserver.SetClientInfoHandler(server.ClientFormHandler)
//	ginserver.SetUserAuthorizationHandler(oauthConfig.UserAuthorizationHandler)
//	if oauthConfig.ClientScopeHandler != nil {
//		ginserver.SetClientScopeHandler(oauthConfig.ClientScopeHandler)
//	}
//	if oauthConfig.AuthorizeScopeHandler != nil {
//		ginserver.SetAuthorizeScopeHandler(oauthConfig.AuthorizeScopeHandler)
//	}
//	if oauthConfig.ExtensionFieldsHandler != nil {
//		ginserver.SetExtensionFieldsHandler(oauthConfig.ExtensionFieldsHandler)
//	}
//	if oauthConfig.ErrorHandleFunc != nil {
//		ginserver.DefaultConfig.ErrorHandleFunc = oauthConfig.ErrorHandleFunc
//	}
//	if oauthConfig.DefaultAuthorizeCodeTokenCfg != nil {
//		manage.DefaultAuthorizeCodeTokenCfg = oauthConfig.DefaultAuthorizeCodeTokenCfg
//	}
//	if oauthConfig.DefaultClientTokenCfg != nil {
//		manage.DefaultClientTokenCfg = oauthConfig.DefaultClientTokenCfg
//	} else {
//		//如果为空的话，默认为authorcode模式，方便后端对token进行刷新操作
//		manage.DefaultClientTokenCfg = manage.DefaultAuthorizeCodeTokenCfg
//	}
//	//auth.GET("/token", ginserver.HandleTokenRequest)
//	//auth.GET("/authorize", ginserver.HandleAuthorizeRequest)
//	//api.Use(ginserver.HandleTokenVerify())
//	//ti, exists := c.Get(ginserver.DefaultConfig.TokenKey)
//	return servers
//}
//
//// StartGinOAuthServer 启动一个gin框架的oauth服务
//func StartGinOAuthServer(oauthRoot *gin.RouterGroup, oauthConfig *GinOauthOption) bool {
//	if oauthConfig == nil {
//		return false
//	}
//	if oauthConfig.ClientStore == nil {
//		return false
//	}
//
//	serverTemp := initGinOAuthServer(oauthConfig)
//	if serverTemp == nil {
//		return false
//	}
//
//	auth := oauthRoot.Group("/oauth2")
//	{
//		auth.GET("/token", ginserver.HandleTokenRequest)
//		auth.GET("/authorize", ginserver.HandleAuthorizeRequest)
//		auth.POST("/authorize", ginserver.HandleAuthorizeRequest) //如果有内容比较多的情况时，不方便用GET
//		//验证并获取登录用户信息
//		middleHandle := ginserver.Config{}
//		if oauthConfig.ErrorHandleFunc != nil {
//			middleHandle.ErrorHandleFunc = oauthConfig.ErrorHandleFunc
//		} else {
//			// 默认的错误输出方式
//			middleHandle.ErrorHandleFunc = func(c *gin.Context, e error) {
//				utils.WriteCommResponse(c.Writer, &utils.CommResponse{
//					Code:    http.StatusUnauthorized,
//					Message: http.StatusText(http.StatusUnauthorized),
//				}, http.StatusUnauthorized)
//				c.Abort()
//			}
//		}
//		auth.GET("/read", ginserver.HandleTokenVerify(middleHandle), func(c *gin.Context) {
//			ti, exists := c.Get(ginserver.DefaultConfig.TokenKey)
//			if exists {
//				resp := &utils.CommResponse{
//					Data: ti,
//				}
//				if oauthConfig.ReadUserCallbackHandler != nil {
//					token, ok := ti.(oauth2.TokenInfo)
//					if ok {
//						tokenInfo := oauthConfig.ReadUserCallbackHandler(token)
//						resp.Data = tokenInfo
//					}
//				}
//				if resp.Data != nil {
//					utils.WriteCommResponse(c.Writer, resp)
//					return
//				}
//			}
//			utils.WriteCommResponse(c.Writer, &utils.CommResponse{
//				Code:    http.StatusUnauthorized,
//				Message: http.StatusText(http.StatusUnauthorized),
//			})
//		})
//	}
//	return true
//}
