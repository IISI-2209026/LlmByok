package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/IISI-2209026/LlmByok/internal/updater"
	"github.com/spf13/cobra"
)

// stubPostRunFetcher implements fetcher for PostRun tests.
type stubPostRunFetcher struct {
	version string
	err     error
	calls   int
}

func (s *stubPostRunFetcher) Channel(v string) string { return "stable" }

func (s *stubPostRunFetcher) LatestRelease(ctx context.Context, channel string) (updater.Release, error) {
	s.calls++
	if s.err != nil {
		return updater.Release{}, s.err
	}
	return updater.Release{Version: s.version, Assets: []updater.Asset{{Name: "byok_windows_amd64.zip"}}}, nil
}

func (s *stubPostRunFetcher) IsNewer(current, latest string) (bool, error) {
	if s.err != nil {
		return false, s.err
	}
	return latest > current, nil
}

func (s *stubPostRunFetcher) DownloadAndReplace(ctx context.Context, rel updater.Release, goos, goarch string) error {
	return nil
}

// withPostRunFetcher swaps defaultFetcher for the duration of the test.
func withPostRunFetcher(f fetcher, fn func()) {
	orig := defaultFetcher
	defaultFetcher = f
	defer func() { defaultFetcher = orig }()
	fn()
}

// newNamedCmd creates a cobra.Command with the given Use (name) and a buffer
// for stderr, returning the command and the buffer.
func newNamedCmd(use string) (*cobra.Command, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	c := &cobra.Command{Use: use}
	c.SetErr(buf)
	return c, buf
}

func TestRootPostRun_PrintsHintOnNewer(t *testing.T) {
	s := &stubPostRunFetcher{version: "9.9.9"}
	withPostRunFetcher(s, func() {
		c, buf := newNamedCmd("config")
		runStartupUpdateCheck(c, "0.1.0")
		out := buf.String()
		if !strings.Contains(out, "update") {
			t.Errorf("expected hint containing 'update' on stderr, got %q", out)
		}
		if s.calls != 1 {
			t.Errorf("expected 1 API call, got %d", s.calls)
		}
	})
}

func TestRootPostRun_NoHintWhenUpToDate(t *testing.T) {
	s := &stubPostRunFetcher{version: "0.1.0"}
	withPostRunFetcher(s, func() {
		c, buf := newNamedCmd("config")
		runStartupUpdateCheck(c, "0.1.0")
		if buf.String() != "" {
			t.Errorf("expected no stderr output when up to date, got %q", buf.String())
		}
	})
}

func TestRootPostRun_SkipsLaunch(t *testing.T) {
	s := &stubPostRunFetcher{version: "9.9.9"}
	withPostRunFetcher(s, func() {
		c, _ := newNamedCmd("launch")
		runStartupUpdateCheck(c, "0.1.0")
		if s.calls != 0 {
			t.Errorf("startup check should be skipped for launch, got %d API calls", s.calls)
		}
	})
}

func TestRootPostRun_SkipsUpdate(t *testing.T) {
	s := &stubPostRunFetcher{version: "9.9.9"}
	withPostRunFetcher(s, func() {
		c, _ := newNamedCmd("update")
		runStartupUpdateCheck(c, "0.1.0")
		if s.calls != 0 {
			t.Errorf("startup check should be skipped for update, got %d API calls", s.calls)
		}
	})
}

func TestRootPostRun_NetworkSilent(t *testing.T) {
	s := &stubPostRunFetcher{err: errors.New("network down")}
	withPostRunFetcher(s, func() {
		c, buf := newNamedCmd("config")
		runStartupUpdateCheck(c, "0.1.0")
		if buf.String() != "" {
			t.Errorf("network errors should be silent, got %q", buf.String())
		}
	})
}

func TestRootPostRun_EnvDisabled(t *testing.T) {
	t.Setenv("BYOK_NO_UPDATE_CHECK", "1")
	s := &stubPostRunFetcher{version: "9.9.9"}
	withPostRunFetcher(s, func() {
		c, _ := newNamedCmd("config")
		runStartupUpdateCheck(c, "0.1.0")
		if s.calls != 0 {
			t.Errorf("BYOK_NO_UPDATE_CHECK=1 should skip, got %d API calls", s.calls)
		}
	})
}