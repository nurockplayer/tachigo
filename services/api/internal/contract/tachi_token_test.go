package contract

import (
	"errors"
	"testing"
)

func TestMintReceiptError_AsAndIs(t *testing.T) {
	txHash := "0x1111111111111111111111111111111111111111111111111111111111111111"
	err := wrapMintReceiptError(txHash, "mint tx failed", ErrMintReceiptStatusFailed)

	var receiptErr *MintReceiptError
	if !errors.As(err, &receiptErr) {
		t.Fatal("expected errors.As to extract *MintReceiptError")
	}
	if receiptErr.TxHash != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash, receiptErr.TxHash)
	}
	if !errors.Is(err, ErrMintReceiptStatusFailed) {
		t.Fatal("expected errors.Is to match ErrMintReceiptStatusFailed")
	}
}

func TestMintReceiptError_WrapsUnderlyingWaitError(t *testing.T) {
	baseErr := errors.New("context deadline exceeded")
	txHash := "0x2222222222222222222222222222222222222222222222222222222222222222"
	err := wrapMintReceiptError(txHash, "wait mint receipt", baseErr)

	var receiptErr *MintReceiptError
	if !errors.As(err, &receiptErr) {
		t.Fatal("expected errors.As to extract *MintReceiptError")
	}
	if receiptErr.TxHash != txHash {
		t.Fatalf("expected tx hash %s, got %s", txHash, receiptErr.TxHash)
	}
	if !errors.Is(err, baseErr) {
		t.Fatal("expected errors.Is to match wrapped wait error")
	}
}
