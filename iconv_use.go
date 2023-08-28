//go:build iconv
// +build iconv

/*
	iconv_use.go
	iconv-related code used with `-tags iconv` build flag
*/

package main

import iconv "github.com/djimenez/iconv-go"

func useIconv() bool {
	return true
}

func convertCharsetFrom(charset, value string) (converted string, nonUTf8 bool, err error) {
	if charset == UTF8 {
		// no converstion
		return value, false, nil
	}
	converted, err = iconv.ConvertString(value, UTF8, charset) // Note that it's safe to store non-UTF8 bytes in Go string, because it's internally just a []byte
	if err != nil {
		// converstion failed
		return "", true, err
	}
	return converted, true, err
}
