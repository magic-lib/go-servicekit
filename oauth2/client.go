package oauth2

import (
	"context"
	"fmt"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	jwtRequest "github.com/golang-jwt/jwt/v4/request"
	"github.com/golang-jwt/jwt/v5"
	"github.com/magic-lib/go-plat-utils/conv"
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"github.com/magic-lib/go-servicekit/oauth2/types"
	"log"
	"net/http"
	"time"
)

/*
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


https://github.com/go-oauth2/oauth2/blob/master/generate.go
*/

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

type ClientCredentials struct {
	PathGroup          string
	TokenStorage       oauth2.TokenStore            //token存储方式
	ClientStorage      oauth2.ClientStore           //client存储方式
	JWTAccessGenerate  *generates.JWTAccessGenerate //jwt的配置，如果配置了，则会使用jwt生成方式
	ClientScopeHandler server.ClientScopeHandler    //判断权限scope的范围是否合法

	getAccessToken server.AuthorizeScopeHandler
	server         *server.Server

	serverName           string
	jwtSecret            string
	jwtTokenTTL          time.Duration
	clientSecretHandler  func(clientId string) string   // 根据clientId获取客户端的密码
	allowedScopesHandler func(clientId string) []string // 根据clientId获取客户端所有的权限列表
}

func NewClientCredentials(cfg *ClientCredentials) (*ClientCredentials, error) {
	if cfg.PathGroup == "" {
		cfg.PathGroup = "oauth2"
	}
	if cfg.TokenStorage == nil {
		var err error
		cfg.TokenStorage, err = store.NewMemoryTokenStore()
		if err != nil {
			return nil, err
		}
	}
	if cfg.ClientStorage == nil {
		return nil, errors.New("ClientStorage nil: token storage is required, example use: store.NewClientStore()")
	}
	if cfg.JWTAccessGenerate != nil {
		if cfg.JWTAccessGenerate.SignedMethod == nil {
			cfg.JWTAccessGenerate.SignedMethod = jwt.SigningMethodHS512
		}
	}
	_ = cfg.GetServer()
	return cfg, nil
}

func (c *ClientCredentials) GetServer() *server.Server {
	if c.server != nil {
		return c.server
	}

	manager := manage.NewDefaultManager()
	manager.MapTokenStorage(c.TokenStorage)
	manager.MapClientStorage(c.ClientStorage)
	if c.JWTAccessGenerate != nil {
		manager.MapAccessGenerate(generates.NewJWTAccessGenerate(c.JWTAccessGenerate.SignedKeyID, c.JWTAccessGenerate.SignedKey, c.JWTAccessGenerate.SignedMethod))
	}

	srv := server.NewDefaultServer(manager)
	srv.SetClientInfoHandler(server.ClientFormHandler)
	srv.SetAllowGetAccessRequest(true)
	clientInfoHandler := func(r *http.Request) (clientID, clientSecret string, err error) {
		req := new(types.ClientTokenRequest)
		query := param.NewParam().GetAllString(r)
		err = conv.Unmarshal(query, req)
		if err != nil {
			return "", "", errors.ErrInvalidRequest
		}
		if req.ClientID == "" || req.ClientSecret == "" {
			return "", "", errors.ErrInvalidClient
		}
		return req.ClientID, req.ClientSecret, nil
	}
	srv.SetClientInfoHandler(clientInfoHandler)

	if c.ClientScopeHandler != nil {
		srv.SetClientScopeHandler(c.ClientScopeHandler)
	}
	c.server = srv

	return srv
}
func (c *ClientCredentials) GetTokenData(ctx context.Context, clientInfo *oauth2.TokenGenerateRequest) (map[string]any, error) {
	srv := c.GetServer()
	ti, err := srv.GetAccessToken(ctx, oauth2.ClientCredentials, clientInfo)
	if err != nil {
		return nil, err
	}
	return srv.GetTokenData(ti), nil
}

func (c *ClientCredentials) getTokenPath() string {
	path := fmt.Sprintf("/%s/%s", c.PathGroup, "token")
	return path
}

func (c *ClientCredentials) GetHttpServerHandler() (http.HandlerFunc, string) {
	srv := c.GetServer()
	return func(w http.ResponseWriter, r *http.Request) {
		if err := srv.HandleTokenRequest(w, r); err != nil {
			log.Print("Internal Error:", err.Error())
		}
	}, c.getTokenPath()
}
func (c *ClientCredentials) getTokenInfo(ctx context.Context, token string) (oauth2.TokenInfo, error) {
	srv := c.GetServer()

	token, _ = jwtRequest.AuthorizationHeaderExtractor.Filter(token)

	if c.JWTAccessGenerate != nil {
		jwtTokenInfo, err := jwt.ParseWithClaims(token, &generates.JWTAccessClaims{}, func(t *jwt.Token) (interface{}, error) {
			return c.JWTAccessGenerate.SignedKey, nil
		})
		if err != nil {
			return nil, err
		}

		claims, ok := jwtTokenInfo.Claims.(*generates.JWTAccessClaims)
		if !ok || !jwtTokenInfo.Valid {
			return nil, errors.ErrInvalidAccessToken
		}
		ti := models.NewToken()
		ti.SetClientID(claims.Audience[0])
		ti.SetUserID(claims.Subject)
		ti.SetAccess(token)
		if claims.IssuedAt == nil {
			now := jwt.NewNumericDate(time.Now())
			claims.IssuedAt = now
		}
		ti.SetAccessCreateAt(time.Unix(claims.IssuedAt.Unix(), 0))
		if claims.ExpiresAt != nil {
			ti.SetAccessExpiresIn(claims.ExpiresAt.Sub(claims.IssuedAt.Time))
		}
		return ti, nil
	}

	tokenInfo, err := srv.Manager.LoadAccessToken(ctx, token)

	if err != nil {
		return nil, err
	}
	if tokenInfo == nil {
		return nil, errors.ErrInvalidAccessToken
	}

	now := time.Now()
	expireTime := tokenInfo.GetAccessCreateAt().Add(tokenInfo.GetAccessExpiresIn())
	if now.After(expireTime) {
		return nil, errors.ErrExpiredAccessToken
	}
	return tokenInfo, nil
}
func (c *ClientCredentials) GetClientInfo(ctx context.Context, token string) (oauth2.ClientInfo, error) {
	tokenInfo, err := c.getTokenInfo(ctx, token)
	if err != nil {
		return nil, err
	}
	clientInfo, err := c.ClientStorage.GetByID(ctx, tokenInfo.GetClientID())
	if err != nil {
		return nil, err
	}
	return clientInfo, nil
}
