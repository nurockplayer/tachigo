package handlers

import (
	"time"

	"github.com/tachigo/tachigo/internal/models"
)

// AuthResponse is the data payload returned on successful auth.
type AuthResponse struct {
	User   models.User      `json:"user"`
	Tokens BrowserTokenPair `json:"tokens"`
}

// TokensResponse is the data payload returned on token refresh.
type TokensResponse struct {
	Tokens BrowserTokenPair `json:"tokens"`
}

// MessageResponse is a generic message payload.
type MessageResponse struct {
	Message string `json:"message"`
}

// UserResponse wraps a single user.
type UserResponse struct {
	User models.User `json:"user"`
}

// ProvidersResponse wraps a list of auth providers.
type ProvidersResponse struct {
	Providers []models.AuthProvider `json:"providers"`
}

// AddressResponse wraps a single address.
type AddressResponse struct {
	Address models.ShippingAddress `json:"address"`
}

// AddressesResponse wraps a list of addresses.
type AddressesResponse struct {
	Addresses []models.ShippingAddress `json:"addresses"`
}

// NonceResponse wraps a Web3 nonce.
type NonceResponse struct {
	Nonce    string `json:"nonce"`
	IssuedAt string `json:"issued_at"`
}

// WalletResponse wraps the bound wallet address.
type WalletResponse struct {
	Address string `json:"address"`
}

type PointsBalanceResponse struct {
	CumulativeTotal  int64 `json:"cumulative_total"`
	SpendableBalance int64 `json:"spendable_balance"`
}

type PointsHistoryItem struct {
	Type      string    `json:"type" enums:"earn,spend"`
	Amount    int64     `json:"amount"`
	SKU       *string   `json:"sku,omitempty"`
	Note      *string   `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at" format:"date-time"`
}

type PointsHistoryResponse struct {
	Transactions []PointsHistoryItem `json:"transactions"`
}
