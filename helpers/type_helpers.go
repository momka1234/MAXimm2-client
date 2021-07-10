package helpers

import "strconv"

func BoolAddr(b bool) *bool {
	boolVar := b
	return &boolVar
}

func IsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
