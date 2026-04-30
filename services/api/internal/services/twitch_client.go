package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type twitchUsersResponse struct {
	Data []TwitchUserInfo `json:"data"`
}

func fetchTwitchUser(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token, clientID string) (*TwitchUserInfo, error) {
	client := cfg.Client(ctx, token)

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.twitch.tv/helix/users", nil)
	req.Header.Set("Client-Id", clientID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("twitch user fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("twitch API returned %d", resp.StatusCode)
	}

	var result twitchUsersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no twitch user data")
	}
	return &result.Data[0], nil
}

func fetchGoogleUser(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (*googleUserInfo, error) {
	client := cfg.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google user fetch: %w", err)
	}
	defer resp.Body.Close()

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
