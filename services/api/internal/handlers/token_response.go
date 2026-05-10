package handlers

import "github.com/tachigo/tachigo/internal/services"

type BrowserTokenPair struct {
	AccessToken string `json:"access_token" binding:"required"`
	ExpiresIn   int    `json:"expires_in" binding:"required"`
}

func browserTokenPair(tokens *services.TokenPair) BrowserTokenPair {
	if tokens == nil {
		return BrowserTokenPair{}
	}
	return BrowserTokenPair{
		AccessToken: tokens.AccessToken,
		ExpiresIn:   tokens.ExpiresIn,
	}
}
