package cli

import (
	"io"
	"testing"
)

// TestResolveUIMode verifies ui mode decision logic.
func TestResolveUIMode(t *testing.T) {
	cases := []struct {
		name       string
		mode       string
		verbose    bool
		isTTY      bool
		expectLive bool
		wantWarn   bool
		wantErr    bool
	}{
		{name: "auto tty", mode: "auto", verbose: false, isTTY: true, expectLive: true},
		{name: "auto non-tty", mode: "auto", verbose: false, isTTY: false, expectLive: false},
		{name: "plain", mode: "plain", verbose: false, isTTY: true, expectLive: false},
		{name: "verbose disables", mode: "auto", verbose: true, isTTY: true, expectLive: false},
		{name: "live tty", mode: "live", verbose: false, isTTY: true, expectLive: true},
		{name: "live non-tty warning", mode: "live", verbose: false, isTTY: false, expectLive: false, wantWarn: true},
		{name: "invalid mode", mode: "nope", verbose: false, isTTY: true, wantErr: true},
	}

	original := isTerminal
	t.Cleanup(func() { isTerminal = original })

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			isTerminal = func(_ io.Writer) bool { return tc.isTTY }
			decision, err := resolveUIMode(tc.mode, tc.verbose, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if decision.useLive != tc.expectLive {
				t.Fatalf("expected useLive=%v, got %v", tc.expectLive, decision.useLive)
			}
			if tc.wantWarn && decision.warning == "" {
				t.Fatalf("expected warning")
			}
			if !tc.wantWarn && decision.warning != "" {
				t.Fatalf("did not expect warning")
			}
		})
	}
}
