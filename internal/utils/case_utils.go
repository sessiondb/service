// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package utils

import (
	"strings"
	"unicode"
)

// ToPascalCase converts snake_case or kebab-case to Pascal Case with spaces.
// Example: read_only -> Read Only
func ToPascalCase(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			runes := []rune(parts[i])
			runes[0] = unicode.ToUpper(runes[0])
			parts[i] = string(runes)
		}
	}
	return strings.Join(parts, " ")
}

// ToSnakeCase converts Pascal Case or Space Case to snake_case.
// Example: Database Maintainer -> database_maintainer
func ToSnakeCase(s string) string {
	var res strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && !unicode.IsUpper(rune(s[i-1])) && s[i-1] != ' ' {
				res.WriteRune('_')
			}
			res.WriteRune(unicode.ToLower(r))
		} else if r == ' ' || r == '-' {
			if i > 0 && res.Len() > 0 && res.String()[res.Len()-1] != '_' {
				res.WriteRune('_')
			}
		} else {
			res.WriteRune(r)
		}
	}
	return res.String()
}
