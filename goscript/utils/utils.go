package utils

import (
	"fmt"
	"math"
	"math/big"
)

func ParseBigInt(s string) *big.Int {

	result := new(big.Int)
	if s == "" || s == "0" {
		return result
	}
	result.SetString(s, 10)
	return result

}

func ParseBigFloatToBigInt(s string) *big.Int {
	if s == "" || s == "0" {
		return big.NewInt(0)
	}

	f, _, err := big.ParseFloat(s, 10, 256, big.ToNearestEven)
	if err != nil {
		return big.NewInt(0)
	}

	// scale par 1e18 pour garder les décimales
	scale := new(big.Float).SetInt(TenPowInt(18))
	f.Mul(f, scale)

	result := new(big.Int)
	f.Int(result) // tronque vers zéro
	return result
}

// returns 10 ^ y
func TenPowInt(y uint) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(y)), nil)
}

func FormatWAD(v *big.Int) string {
	f, _ := new(big.Float).Quo(
		new(big.Float).SetInt(v),
		new(big.Float).SetInt(WAD),
	).Float64()
	return fmt.Sprintf("%.4f", f)
}

// for token amounts where you know decimals
func FormatDecimals(x *big.Int, decimals int) string {
	if x == nil {
		return "nil"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	f := new(big.Float).SetInt(x)
	f.Quo(f, new(big.Float).SetInt(divisor))
	return f.Text('f', 4)
}

func BigIntToFloat(v *big.Int) float64 {
	if v == nil {
		return 0
	}
	f, acc := new(big.Float).SetPrec(128).SetInt(v).Float64()
	if acc == big.Above && f == math.Inf(1) {
		return math.MaxFloat64
	}
	return f
}

func BigIntWADToFloat(v *big.Int) float64 {
	return BigIntToFloat(v) / 1e18
}
func FloatE36Int(f float64) *big.Int {
	bigF := new(big.Float).SetFloat64(f)
	wadF := new(big.Float).SetFloat64(1e36)
	bigF = new(big.Float).Mul(bigF, wadF)
	result := new(big.Int)
	bigF.Int(result)
	return result
}
