// Copyright (c) 2026 Sai Mouli Bandari. Licensed under Business Source License 1.1.

package access

import (
	"sessiondb/internal/engine"
	"testing"
)

// Ensure Engine implements engine.AccessEngine.
func TestEngineImplementsAccessEngine(t *testing.T) {
	var _ engine.AccessEngine = (*Engine)(nil)
}
