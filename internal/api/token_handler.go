package api

import (
	"github.com/ap-rmit/scraper-don/internal/auth"
)

type TokenHandler struct {
	tokenService *auth.TokenService
}

func NewTokenHandler(tokenService *auth.TokenService) *TokenHandler {
	return &TokenHandler{tokenService: tokenService}
}

func (th *TokenHandler) GetToken() (string, error) {
	return th.tokenService.GetCurrentToken(), nil
}
