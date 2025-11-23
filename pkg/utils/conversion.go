package utils

import "strconv"

// StringToUint64 mengubah string angka menjadi uint64
// Berguna untuk parsing ID dari URL parameter
func StringToUint64(str string) uint64 {
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0 // Return 0 jika gagal parsing
	}
	return val
}

func StringToFloat(str string) float64 {
	val, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0
	}
	return val
}
