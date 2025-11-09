package oauth2_test

import (
	"encoding/json"
	"fmt"
	goOauth2 "github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/generates"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/golang-jwt/jwt/v5"
	"github.com/magic-lib/go-plat-utils/utils/httputil/param"
	"github.com/magic-lib/go-servicekit/oauth2"
	"log"
	"net/http"
	"testing"
)

func TestClientCredentials(t *testing.T) {

	clientStore := store.NewClientStore()
	clientStore.Set("tianlin0", &models.Client{
		ID:     "tianlin0",
		Secret: "12345",
		Domain: "tianlin0.qq.com",
		UserID: "tianlin0",
	})

	cTemp, err := oauth2.NewClientCredentials(&oauth2.ClientCredentials{
		ClientStorage: clientStore,
		JWTAccessGenerate: &generates.JWTAccessGenerate{
			SignedKey:    []byte("secret"),
			SignedMethod: jwt.SigningMethodHS256,
		},
		ClientScopeHandler: func(tgr *goOauth2.TokenGenerateRequest) (allowed bool, err error) {
			allowed = true
			return
		},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	handler, path := cTemp.GetHttpServerHandler()

	http.HandleFunc(path, handler)
	http.HandleFunc("/check", func(writer http.ResponseWriter, r *http.Request) {
		headers := param.NewParam().GetAllHeaders(r)
		tokenStr := headers.Get("Authorization")

		clientInfo, err := cTemp.GetClientInfo(r.Context(), tokenStr)
		if err != nil {
			_ = json.NewEncoder(writer).Encode(err.Error())
			return
		}
		//根据userId再去从数据库查询用户信息
		fmt.Println(clientInfo.GetUserID())

		_ = json.NewEncoder(writer).Encode(clientInfo)
	})

	// 启动服务端（监听9096端口）
	log.Println("Server running on :9096")
	go func() {
		log.Fatal(http.ListenAndServe(":9096", nil))
	}()
	select {}
}
