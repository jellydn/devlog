package shellescape

import (
	"os/exec"
	"strings"
	"testing"
	"testing/quick"
)

func TestQuote(t *testing.T) {
	inputs := []string{
		"",
		"simple",
		"a'b",
		"; rm -rf $HOME; '",
		"$HOME",
		"hello world",
	}
	for _, in := range inputs {
		got := Quote(in)
		var want string
		if in == "" {
			want = "''"
		} else {
			want = "'" + strings.ReplaceAll(in, "'", "'\\''") + "'"
		}
		if got != want {
			t.Errorf("Quote(%q) = %q, want %q", in, got, want)
		}
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("Quote(%q) = %q, should be single-quoted", in, got)
		}
	}
}

func TestQuote_InjectionCannotBreakOut(t *testing.T) {
	// A hostile cmd that would run rm if unquoted or poorly escaped.
	hostile := `; rm -rf /tmp/devlog-shellescape-should-not-exist; echo pwned; '`
	quoted := Quote(hostile)

	// When used as the argument to printf via sh -c, the shell should treat
	// the whole string as data, not execute rm.
	cmd := exec.Command("sh", "-c", "printf %s "+quoted)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sh failed: %v output=%q", err, out)
	}
	if string(out) != hostile {
		t.Errorf("round-trip = %q, want original %q", out, hostile)
	}
}

func TestQuote_RoundTripProperty(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	f := func(s string) bool {
		// Cap size to keep the property test fast.
		if len(s) > 200 {
			return true
		}
		quoted := Quote(s)
		cmd := exec.Command("sh", "-c", "printf %s "+quoted)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("sh error for %q quoted=%q: %v %s", s, quoted, err, out)
			return false
		}
		return string(out) == s
	}
	if err := quick.Check(f, &quick.Config{MaxCount: 100}); err != nil {
		t.Error(err)
	}
}
