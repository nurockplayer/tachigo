package services

import (
	"math/big"
	"testing"
)

func TestTachiWholeTokensToRawUnits(t *testing.T) {
	got := tachiWholeTokensToRawUnits(100)
	want, ok := new(big.Int).SetString("100000000000000000000", 10)
	if !ok {
		t.Fatal("invalid expected raw unit amount")
	}

	if got.Cmp(want) != 0 {
		t.Fatalf("expected 100 $TACHI to equal %s raw units, got %s", want.String(), got.String())
	}
}
