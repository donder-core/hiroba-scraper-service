package models

type LoginResponse struct {
	Status   int    `json:"status"`
	Redirect string `json:"redirect"`
	Message  string `json:"message"`
}
