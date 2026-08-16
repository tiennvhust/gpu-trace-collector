package config

import (
	"path/filepath"
	"testing"
)

// TestShippedConfigsParse asserts that every config committed to this repo still
// parses with the code in this repo.
//
// » This is a test rather than a CI step on purpose. The failure it catches — a
// » field renamed in config.go and not in the YAML — otherwise surfaces at
// » container start, in a cluster, as a crash-looping pod. A check that only
// » lives in a workflow file is one you cannot run before pushing.
//
// » The glob is `collector*.yaml`, NOT `*.yaml`. configs/ also holds
// » dap-helper.yaml, dap-leader.yaml and privacy.yaml, which are configs for
// » the other binaries in this repo and have entirely different schemas —
// » feeding them to this package's Load would either fail or, worse, pass
// » while proving nothing. Each of those deserves its own test next to its own
// » loader.
func TestShippedConfigsParse(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "configs", "collector*.yaml"))
	if err != nil {
		t.Fatalf("glob configs: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("no collector*.yaml found under configs/ — did the directory move?")
	}

	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			if _, err := Load(p); err != nil {
				t.Errorf("does not parse: %v", err)
			}
		})
	}
}
