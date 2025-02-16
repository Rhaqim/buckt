package model

type UserModel struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}
