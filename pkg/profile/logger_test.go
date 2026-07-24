package profile

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was written. Success uses fmt.Println (stdout), unlike the leveled logs.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	fn()
	require.NoError(t, w.Close())
	os.Stdout = orig
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// captureStderr is captureStdout for the leveled logs, which the underlying
// logger writes to stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	fn()
	require.NoError(t, w.Close())
	os.Stderr = orig
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestDebugLevel(t *testing.T) {
	t.Run("debug output reaches stderr at debug level", func(t *testing.T) {
		out := captureStderr(t, func() {
			NewLogger("debug").Debug("looking up %s", "sub-a")
		})
		require.Contains(t, out, "looking up sub-a")
	})

	t.Run("suppressed at info level", func(t *testing.T) {
		out := captureStderr(t, func() {
			NewLogger("info").Debug("looking up %s", "sub-a")
		})
		require.Equal(t, "", strings.TrimSpace(out))
	})

	t.Run("warn still prints at info level", func(t *testing.T) {
		out := captureStderr(t, func() {
			NewLogger("info").Warn("careful")
		})
		require.Contains(t, out, "careful")
	})

	t.Run("warn suppressed at error level", func(t *testing.T) {
		out := captureStderr(t, func() {
			NewLogger("error").Warn("careful")
		})
		require.Equal(t, "", strings.TrimSpace(out))
	})
}

func TestSuccessQuiet(t *testing.T) {
	t.Run("prints by default", func(t *testing.T) {
		out := captureStdout(t, func() {
			NewLogger("info").Success("switched context to: %s", "sub-a")
		})
		require.Contains(t, out, "switched context to: sub-a")
	})

	t.Run("suppressed when quiet", func(t *testing.T) {
		out := captureStdout(t, func() {
			NewLogger("info").SetQuiet(true).Success("switched context to: %s", "sub-a")
		})
		require.Equal(t, "", strings.TrimSpace(out))
	})
}
