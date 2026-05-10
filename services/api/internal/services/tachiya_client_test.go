package services

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/tachigo/tachigo/internal/testutil"
)

func TestTachiyaHTTPClient_RedeemCoupon_Success(t *testing.T) {
	client := NewTachiyaHTTPClient("https://tachiya.test", "shared-secret")
	client.httpClient = testutil.NewHTTPClient(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/coupons/redeem" {
			t.Fatalf("expected /coupons/redeem, got %s", r.URL.Path)
		}
		if got := r.Header.Get("X-Tachiya-Internal-Secret"); got != "shared-secret" {
			t.Fatalf("expected internal secret header, got %q", got)
		}

		var req struct {
			CouponID string `json:"coupon_id"`
			TCGCost  int64  `json:"tcg_cost"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.CouponID != "coupon-123" {
			t.Fatalf("expected coupon_id=coupon-123, got %s", req.CouponID)
		}
		if req.TCGCost != 100 {
			t.Fatalf("expected tcg_cost=100, got %d", req.TCGCost)
		}

		resp := testutil.NewStringResponse(http.StatusOK, `{"voucher_code":"VOUCHER-XYZ"}`)
		resp.Header.Set("Content-Type", "application/json")
		return resp, nil
	})

	voucherCode, err := client.RedeemCoupon(context.Background(), "coupon-123", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if voucherCode != "VOUCHER-XYZ" {
		t.Fatalf("expected voucher code VOUCHER-XYZ, got %s", voucherCode)
	}
}

func TestTachiyaHTTPClient_RedeemCoupon_NonOKStatus(t *testing.T) {
	client := NewTachiyaHTTPClient("https://tachiya.test", "shared-secret")
	client.httpClient = testutil.NewHTTPClient(func(r *http.Request) (*http.Response, error) {
		return testutil.NewStringResponse(http.StatusInternalServerError, ""), nil
	})

	if _, err := client.RedeemCoupon(context.Background(), "coupon-123", 100); err == nil {
		t.Fatal("expected error but got nil")
	}
}
