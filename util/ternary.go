package util

func Ternary(condition bool, a, b string) string {
	if condition {
		return a
	}
	return b
}
