// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package ai

import (
	"regexp"
	"strings"
)

var sqlBlockRe = regexp.MustCompile(`(?s)^\s*(?:` + "`" + `{3}\s*sql\s*\n?|` + "`" + `{3}\s*\n?)(.*?)` + "`" + `{3}\s*$`)

// StripSQLCodeFence removes markdown code fences (```sql or ```) from AI-generated SQL
// so the API returns plain SQL for the editor.
func StripSQLCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if m := sqlBlockRe.FindStringSubmatch(s); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return s
}
