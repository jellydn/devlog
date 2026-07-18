// Package shellescape provides POSIX shell single-quote escaping helpers.
package shellescape

import "strings"

// Quote returns a shell-safe single-quoted representation of s suitable for
// interpolation into a POSIX shell command line (e.g. sh -lc).
// Empty strings become ” so they remain a single argument.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	// Wrap in single quotes; escape any embedded ' as '\'' (end quote, escaped
	// literal quote, reopen quote).
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
