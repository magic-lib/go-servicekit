package types

type ClientTokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`       // 必需，固定为 client_credentials
	ClientID     string `json:"client_id" form:"client_id"`         // 必需，客户端标识
	ClientSecret string `json:"client_secret" form:"client_secret"` // 必需，客户端密钥
	Scope        string `json:"scope" form:"scope"`                 // 可选，权限范围
}
type ClientTokenResponse struct {
	AccessToken string `json:"access_token"`    // 访问令牌
	TokenType   string `json:"token_type"`      // 令牌类型，固定为 Bearer
	ExpiresIn   int64  `json:"expires_in"`      // 有效期（秒）
	Scope       string `json:"scope,omitempty"` // 实际授予的权限范围
}
