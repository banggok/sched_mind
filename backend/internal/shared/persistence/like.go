package persistence

import "strings"

// EscapeLike escapes characters that have special meaning in SQL LIKE patterns.
func EscapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
