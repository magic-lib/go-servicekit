package types

/*
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

刷新token消息
http://localhost:8083/oauth2/token?grant_type=refresh_token&client_id=aaaa&
client_secret=827ccb0eea8a706c4c34a16891f84e7b&scope=aaaa&refresh_token=6S3C0HQZVJWAETDLA5OMLQ
说明：token被刷新以后，前面的token就用不了了
*/
