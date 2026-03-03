package main

import (
	"strings"
	"testing"
)

// TestBuildMountArgs tests the logic for building mount command arguments
func TestBuildMountArgs(t *testing.T) {
	tests := []struct {
		name           string
		options        map[string]string
		boolFlags      []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "raw mountFlags passthrough",
			options: map[string]string{
				"mountFlags": "--all-squash 1000:1000 --cache-size 10G",
			},
			boolFlags: []string{"writeback", "no-bgjob"},
			wantContains: []string{
				"--all-squash",
				"1000:1000",
				"--cache-size",
				"10G",
			},
			wantNotContain: []string{
				"--mountFlags",
			},
		},
		{
			name: "boolean flag without value",
			options: map[string]string{
				"writeback": "true",
				"no-bgjob":  "true",
			},
			boolFlags: []string{"writeback", "no-bgjob"},
			wantContains: []string{
				"--writeback",
				"--no-bgjob",
			},
			wantNotContain: []string{
				"--writeback=",
				"--no-bgjob=",
			},
		},
		{
			name: "non-boolean flag with value",
			options: map[string]string{
				"cache-size":  "10G",
				"buffer-size": "300M",
			},
			boolFlags: []string{"writeback", "no-bgjob"},
			wantContains: []string{
				"--cache-size=10G",
				"--buffer-size=300M",
			},
		},
		{
			name: "mixed flags",
			options: map[string]string{
				"mountFlags": "--all-squash 1000:1000",
				"writeback":  "true",
				"cache-size": "10G",
			},
			boolFlags: []string{"writeback", "no-bgjob"},
			wantContains: []string{
				"--all-squash",
				"1000:1000",
				"--writeback",
				"--cache-size=10G",
			},
			wantNotContain: []string{
				"--writeback=",
			},
		},
		{
			name: "empty mountFlags",
			options: map[string]string{
				"mountFlags": "",
				"writeback":  "true",
			},
			boolFlags: []string{"writeback"},
			wantContains: []string{
				"--writeback",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildMountArgs(tt.options, tt.boolFlags)
			argsStr := strings.Join(args, " ")

			for _, want := range tt.wantContains {
				if !strings.Contains(argsStr, want) {
					t.Errorf("buildMountArgs() = %v, want to contain %q", args, want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(argsStr, notWant) {
					t.Errorf("buildMountArgs() = %v, should not contain %q", args, notWant)
				}
			}
		})
	}
}

// TestBuildFormatArgs tests the logic for building format command arguments
func TestBuildFormatArgs(t *testing.T) {
	tests := []struct {
		name           string
		options        map[string]string
		formatOptions  []string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name: "raw formatFlags passthrough",
			options: map[string]string{
				"formatFlags": "--compress lz4 --trash-days 0",
			},
			formatOptions: []string{"compress", "trash-days"},
			wantContains: []string{
				"--compress",
				"lz4",
				"--trash-days",
				"0",
			},
		},
		{
			name: "standard format options",
			options: map[string]string{
				"compress":   "lz4",
				"trash-days": "7",
			},
			formatOptions: []string{"compress", "trash-days"},
			wantContains: []string{
				"--compress=lz4",
				"--trash-days=7",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildFormatArgs(tt.options, tt.formatOptions)
			argsStr := strings.Join(args, " ")

			for _, want := range tt.wantContains {
				if !strings.Contains(argsStr, want) {
					t.Errorf("buildFormatArgs() = %v, want to contain %q", args, want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(argsStr, notWant) {
					t.Errorf("buildFormatArgs() = %v, should not contain %q", args, notWant)
				}
			}
		})
	}
}

// TestBuildAuthArgs tests the logic for building auth command arguments
func TestBuildAuthArgs(t *testing.T) {
	tests := []struct {
		name         string
		options      map[string]string
		wantContains []string
	}{
		{
			name: "raw authFlags passthrough",
			options: map[string]string{
				"authFlags": "--token mytoken --access-key mykey",
			},
			wantContains: []string{
				"--token",
				"mytoken",
				"--access-key",
				"mykey",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := buildAuthArgs(tt.options)
			argsStr := strings.Join(args, " ")

			for _, want := range tt.wantContains {
				if !strings.Contains(argsStr, want) {
					t.Errorf("buildAuthArgs() = %v, want to contain %q", args, want)
				}
			}
		})
	}
}

// buildMountArgs extracts and builds mount arguments from options
// This mirrors the logic from ceMount/eeMount for testing
func buildMountArgs(options map[string]string, boolFlags []string) []string {
	var args []string
	opts := make(map[string]string)
	for k, v := range options {
		opts[k] = v
	}

	// Handle raw mountFlags first
	if rawFlags, ok := opts["mountFlags"]; ok {
		args = append(args, strings.Fields(rawFlags)...)
		delete(opts, "mountFlags")
	}

	// Handle boolean flags
	for _, flag := range boolFlags {
		if _, ok := opts[flag]; ok {
			args = append(args, "--"+flag)
			delete(opts, flag)
		}
	}

	// Handle remaining options as key=value
	for k, v := range opts {
		args = append(args, "--"+k+"="+v)
	}

	return args
}

// buildFormatArgs extracts and builds format arguments from options
func buildFormatArgs(options map[string]string, formatOptions []string) []string {
	var args []string
	opts := make(map[string]string)
	for k, v := range options {
		opts[k] = v
	}

	// Handle raw formatFlags first
	if rawFlags, ok := opts["formatFlags"]; ok {
		args = append(args, strings.Fields(rawFlags)...)
		delete(opts, "formatFlags")
	}

	// Handle format options
	for _, opt := range formatOptions {
		if val, ok := opts[opt]; ok {
			args = append(args, "--"+opt+"="+val)
			delete(opts, opt)
		}
	}

	return args
}

// buildAuthArgs extracts and builds auth arguments from options
func buildAuthArgs(options map[string]string) []string {
	var args []string
	opts := make(map[string]string)
	for k, v := range options {
		opts[k] = v
	}

	// Handle raw authFlags first
	if rawFlags, ok := opts["authFlags"]; ok {
		args = append(args, strings.Fields(rawFlags)...)
		delete(opts, "authFlags")
	}

	return args
}
