package main

import (
	"testing"
)

func TestFindSubcommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCmd  string
		wantRest []string
		wantOK   bool
	}{
		{
			name:     "update as first arg",
			args:     []string{"planck", "update"},
			wantCmd:  "update",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "update with --check",
			args:     []string{"planck", "update", "--check"},
			wantCmd:  "update",
			wantRest: []string{"--check"},
			wantOK:   true,
		},
		{
			name:     "version subcommand",
			args:     []string{"planck", "version"},
			wantCmd:  "version",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "attach subcommand",
			args:     []string{"planck", "attach"},
			wantCmd:  "attach",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "flag before subcommand: --folder path update",
			args:     []string{"planck", "--folder", "/tmp", "update"},
			wantCmd:  "update",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "short flag before subcommand: -f path update",
			args:     []string{"planck", "-f", "/tmp", "update"},
			wantCmd:  "update",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "flag=value before subcommand",
			args:     []string{"planck", "--folder=/tmp", "update"},
			wantCmd:  "update",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:     "bool flag before subcommand",
			args:     []string{"planck", "-v", "version"},
			wantCmd:  "version",
			wantRest: []string{},
			wantOK:   true,
		},
		{
			name:    "no args",
			args:    []string{"planck"},
			wantCmd: "",
			wantOK:  false,
		},
		{
			name:    "only flags",
			args:    []string{"planck", "--folder", "/tmp"},
			wantCmd: "",
			wantOK:  false,
		},
		{
			name:    "unknown positional arg",
			args:    []string{"planck", "foobar"},
			wantCmd: "",
			wantOK:  false,
		},
		{
			name:     "subcommand after --folder flag value with subcmd flag",
			args:     []string{"planck", "-folder", "/some/path", "update", "--check"},
			wantCmd:  "update",
			wantRest: []string{"--check"},
			wantOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, rest, found := findSubcommand(tt.args)
			if found != tt.wantOK {
				t.Errorf("found = %v, want %v", found, tt.wantOK)
			}
			if cmd != tt.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, tt.wantCmd)
			}
			if tt.wantOK {
				if len(rest) == 0 && len(tt.wantRest) == 0 {
					// both empty, ok
				} else if len(rest) != len(tt.wantRest) {
					t.Errorf("rest = %v, want %v", rest, tt.wantRest)
				} else {
					for i := range rest {
						if rest[i] != tt.wantRest[i] {
							t.Errorf("rest[%d] = %q, want %q", i, rest[i], tt.wantRest[i])
						}
					}
				}
			}
		})
	}
}

func TestKnownSubcommands(t *testing.T) {
	expected := []string{"update", "version", "attach"}
	for _, cmd := range expected {
		if !knownSubcommands[cmd] {
			t.Errorf("expected %q to be a known subcommand", cmd)
		}
	}

	if knownSubcommands["foobar"] {
		t.Error("unexpected subcommand 'foobar' was found")
	}
}
