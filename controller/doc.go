package controller

// 仅用于 Swagger 文档生成的响应结构，不参与实际业务。

type registerResult struct {
	ID    uint   `json:"id" example:"1"`
	Name  string `json:"name" example:"alice"`
	Phone string `json:"phone" example:"13800000001"`
}

type refreshResult struct {
	AccessToken string `json:"access_token" example:"eyJhbGciOi..."`
}

type meResult struct {
	ID    uint   `json:"id" example:"1"`
	Name  string `json:"name" example:"alice"`
	Phone string `json:"phone" example:"13800000001"`
}
