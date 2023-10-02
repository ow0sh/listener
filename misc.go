package main

import (
	"math"
	"math/big"
)

func divideBigIntByDecimals(x *big.Int, y uint8) *big.Float {
	num1 := x
	num2 := big.NewInt(int64(math.Pow(10, float64(y))))
	aflt1 := new(big.Float).SetInt(num1)
	aflt2 := new(big.Float).SetInt(num2)
	amount := new(big.Float).Quo(aflt1, aflt2)
	return amount
}
