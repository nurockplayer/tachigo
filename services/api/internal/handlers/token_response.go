package handlers

import "github.com/tachigo/tachigo/internal/services"

type BrowserTokenPair struct {
	AccessToken string `json:"access_token"`
}

func browserTokenPair(tokens *services.TokenPair) BrowserTokenPair {
	if tokens == nil {
		return BrowserTokenPair{}
	}
	return BrowserTokenPair{AccessToken: tokens.AccessToken}
}
