//go:build pro

package repository

// QueryInsights is a stub for premium query insights storage.
type QueryInsights struct{}

// NewQueryInsights returns a stub QueryInsights instance.
func NewQueryInsights() *QueryInsights { return &QueryInsights{} }

// DBMetrics is a stub for premium DB metrics storage.
type DBMetrics struct{}

// NewDBMetrics returns a stub DBMetrics instance.
func NewDBMetrics() *DBMetrics { return &DBMetrics{} }

// DBAlters is a stub for premium DB alters storage.
type DBAlters struct{}

// NewDBAlters returns a stub DBAlters instance.
func NewDBAlters() *DBAlters { return &DBAlters{} }
