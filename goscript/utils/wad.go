package utils

import (
	"math/big"
)

var (
	WADX11       = new(big.Int).Mul(WAD, big.NewInt(11))
	WADX13       = new(big.Int).Mul(WAD, big.NewInt(13))
	WAD1DOT1     = new(big.Int).Div(WADX11, big.NewInt(10)) // 1.1 WAD
	WAD1DOT3     = new(big.Int).Div(WADX13, big.NewInt(10)) // 1.3 WAD
	WADON10      = new(big.Int).Div(WAD, big.NewInt(10))    // 0.1 WAD
	WAD          = TenPowInt(18)
	HALF_WAD     = new(big.Int).Div(WAD, big.NewInt(2))
	WAD1DOT05    = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(20)))    // 1.05%
	WAD1DOT01    = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(100)))   // 1.01%
	WAD1DOT005   = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(200)))   // 1.005%
	WAD1DOT0005  = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(2000)))  // 1.005%
	WAD1DOT00005 = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(20000))) // 1.0005%
	WAD_2        = new(big.Int).Mul(big.NewInt(2), WAD)
)

func DetectScale(x *big.Int) int {
	switch {
	case x.Cmp(TenPowInt(30)) > 0:
		return 36
	case x.Cmp(TenPowInt(15)) > 0:
		return 18
	case x.Cmp(TenPowInt(6)) > 0:
		return 8
	default:
		return 0
	}
}

func NormalizeToWAD(x *big.Int, scale int) *big.Int {
	switch scale {
	case 36:
		return new(big.Int).Div(x, TenPowInt(18))
	case 18:
		return new(big.Int).Set(x)
	case 8:
		return new(big.Int).Mul(x, TenPowInt(10))
	default:
		return new(big.Int).Set(x)
	}
}
