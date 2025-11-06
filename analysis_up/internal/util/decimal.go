package util

import "math/big"

// FmtFloat 将 *big.Float* 格式化为十进制字符串（去除多余尾零）
func FmtFloat(x *big.Float, prec int) string {
	if x == nil {
		return "0"
	}
	s := x.Text('f', prec)
	// 去尾零
	for len(s) > 1 && s[len(s)-1] == '0' && s[len(s)-2] != '.' {
		s = s[:len(s)-1]
	}
	if len(s) > 0 && s[len(s)-1] == '.' {
		s = s[:len(s)-1]
	}
	return s
}
