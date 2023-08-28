//go:build !iconv
// +build !iconv

/*
	iconv_none.go
	iconv-related code used WITHOUT `-tags iconv` build flag
*/

package main

func useIconv() bool {
	return false
}

func convertCharsetFrom(charset, value string) (converted string, nonUTf8 bool, err error) {
	return value, false, nil
}
