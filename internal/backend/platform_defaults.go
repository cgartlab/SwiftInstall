package backend

// Default platform wiring for the shared baseline.
//
// Platform lines may override either entry (or add new GOOS values) by
// calling RegisterPlatform from their own init(). The baseline keeps both
// backends compilable and testable on every host — no build tags — so the
// Windows line can run the full test suite, including the Homebrew tests.
func init() {
	RegisterPlatform("windows", func() Backend { return NewWingetBackend() })
	RegisterPlatform("darwin", func() Backend { return NewHomebrewBackend() })
}
