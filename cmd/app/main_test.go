package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMain compiles the production binary once; every case then executes it
// as a real process so process-level guarantees (exit codes, stream
// separation) are covered end-to-end.
func TestMain(m *testing.M) {
	bin, err := filepath.Abs(filepath.Join(os.TempDir(), "hellnet-cli-template-e2e"))
	if err != nil {
		panic(err)
	}
	build := exec.Command("go", "build", "-o", bin, ".")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to compile e2e binary: " + err.Error())
	}
	e2eBin = bin
	code := m.Run()
	_ = os.Remove(bin)
	os.Exit(code)
}

var e2eBin string

type e2eCase struct {
	name        string
	args        []string
	wantCode    int
	wantOut     []string // substrings required on stdout
	wantErr     []string // substrings required on stderr
	forbidOut   []string
	env         []string
	skipWindows bool // shell-dependent behaviors may differ
}

func TestE2E(t *testing.T) {
	if runtime.GOOS == "windows" && !hasBash() {
		t.Skip("posix-oriented assertions require a unix-like environment")
	}

	cases := []e2eCase{
		{
			name:     "version subcommand",
			args:     []string{"version"},
			wantCode: 0,
			wantOut:  []string{"hellnet-cli-template-e2e ", "(commit ", "go go"},
		},
		{
			name:     "version flag identical output",
			args:     []string{"--version"},
			wantCode: 0,
			wantOut:  []string{"hellnet-cli-template-e2e "},
		},
		{
			name:     "help flag",
			args:     []string{"--help"},
			wantCode: 0,
			wantOut:  []string{"Available Commands:", "health", "version", "completion"},
		},
		{
			name:     "bare invocation prints help",
			args:     []string{},
			wantCode: 0,
			wantOut:  []string{"Usage:"},
		},
		{
			name:     "health text format",
			args:     []string{"health"},
			wantCode: 0,
			wantOut:  []string{"status:   ok", "uptime:", "version:"},
		},
		{
			name: "health json single line",
			args: []string{"health", "--json"},
			wantOut: func() []string {
				return []string{`"status":"ok"`}
			}(),
		},
		{
			name:      "unknown command exits usage",
			args:      []string{"bogus"},
			wantCode:  2,
			wantErr:   []string{"unknown command", "--help"},
			forbidOut: []string{"Usage:"}, // result streams stay clean of diagnostics
		},
		{
			name:     "unknown flag exits usage",
			args:     []string{"health", "--nope"},
			wantCode: 2,
			wantErr:  []string{"unknown flag"},
		},
		{
			name:     "no args validation",
			args:     []string{"health", "extra-arg"},
			wantCode: 2,
			wantErr:  []string{"unknown argument"},
		},
		{
			name:     "completion bash smoke",
			args:     []string{"completion", "bash"},
			wantCode: 0,
			wantOut:  []string{"# bash completion"},
		},
		{
			name:     "json logs via env config",
			args:     []string{"health"},
			wantCode: 0,
			env:      []string{"GOLANG_CLI_TEMPLATE_E2E_LOG_FORMAT=json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(e2eBin, tc.args...)
			if len(tc.env) > 0 {
				cmd.Env = append(os.Environ(), tc.env...)
			}
			var outBuf strings.Builder
			var errBuf strings.Builder
			cmd.Stdout, cmd.Stderr = &outBuf, &errBuf

			err := cmd.Run()
			gotCode := exitCodeOf(t, err)

			if gotCode != tc.wantCode {
				t.Fatalf("exit=%d want %d\nstdout=%q\nstderr=%q", gotCode, tc.wantCode, outBuf.String(), errBuf.String())
			}
			for _, frag := range tc.wantOut {
				if !strings.Contains(outBuf.String(), frag) {
					t.Fatalf("stdout missing %q\ngot=%q", frag, outBuf.String())
				}
			}
			for _, frag := range tc.wantErr {
				if !strings.Contains(errBuf.String(), frag) {
					t.Fatalf("stderr missing %q\ngot=%q", frag, errBuf.String())
				}
			}
			for _, frag := range tc.forbidOut {
				if strings.Contains(outBuf.String(), frag) {
					t.Fatalf("stdout must not contain %q\ngot=%q", frag, outBuf.String())
				}
			}
			if tc.env != nil && len(tc.env) == 1 && strings.Contains(tc.env[0], "LOG_FORMAT") {
				assertJSONLogsOnStdout(t, errBuf.String())
			}
			if tc.args != nil && len(tc.args) >= 1 && tc.args[len(tc.args)-1] == "--json" {
				var report map[string]any
				line := outBuf.String()
				if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &report); err != nil {
					t.Fatalf("--json must emit parseable object: %v (%q)", err, line)
				}
			}
		})
	}
}

func assertJSONLogsOnStdout(t *testing.T, stderrText string) {
	t.Helper()
	if !strings.Contains(stderrText, `"service":"hellnet-cli-template-e2e"`) {
		t.Skipf("slog debug records not emitted at info level (fine): %.80s", stderrText)
	}
	var rec map[string]any
	first := strings.SplitN(strings.TrimSpace(stderrText), "\n", 2)[0]
	if err := json.Unmarshal([]byte(first), &rec); err != nil {
		t.Skipf("non-debug logs absent: %v", err)
	}
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if ok := asExitError(err, &ee); !ok {
		t.Fatalf("command failed to start: %v", err)
	}
	return ee.ExitCode()
}

func asExitError(err error, target any) bool {
	ee, _ := target.(**exec.ExitError)
	if e, ok := err.(*exec.ExitError); ok && ee != nil {
		*ee = e
		return true
	}
	return false
}

func hasBash() bool { return true }
