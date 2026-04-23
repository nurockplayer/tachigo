package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TachiyaClient interface {
	RedeemCoupon(couponID string, tcgCost int64) (string, error)
}

type TachiyaHTTPClient struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewTachiyaHTTPClient(baseURL, secret string) *TachiyaHTTPClient {
	return &TachiyaHTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *TachiyaHTTPClient) RedeemCoupon(couponID string, tcgCost int64) (string, error) {
	reqBody := struct {
		CouponID string `json:"coupon_id"`
		TCGCost  int64  `json:"tcg_cost"`
	}{
		CouponID: couponID,
		TCGCost:  tcgCost,
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(reqBody); err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/coupons/redeem", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tachiya-Internal-Secret", c.secret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("tachiya redeem coupon failed with status %d", resp.StatusCode)
	}

	var respBody struct {
		VoucherCode string `json:"voucher_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", err
	}

	return respBody.VoucherCode, nil
}
