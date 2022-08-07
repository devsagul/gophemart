package infra

type userRegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type userLoginRequest userRegisterRequest
