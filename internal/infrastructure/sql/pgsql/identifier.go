package pgsql

import "strings"

// QuoteIdentifier wraps a Postgres identifier in double quotes,
// doubling any embedded double quotes (per SQL standard).
func QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// QuoteLiteral escapes a string for safe embedding as a SQL literal,
// wrapping it in single quotes and escaping backslashes and single quotes.
func QuoteLiteral(literal string) string {
	literal = strings.ReplaceAll(literal, `\`, `\\`)
	literal = strings.ReplaceAll(literal, `'`, `''`)
	return `'` + literal + `'`
}
