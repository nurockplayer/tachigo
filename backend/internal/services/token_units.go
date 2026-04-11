package services

import "math/big"

var tachiTokenRawUnit = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func tachiWholeTokensToRawUnits(amount int64) *big.Int {
	return new(big.Int).Mul(big.NewInt(amount), tachiTokenRawUnit)
}
