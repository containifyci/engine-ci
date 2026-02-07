package rust

import (
	"strings"
	"testing"

	"github.com/containifyci/engine-ci/pkg/cri/types"
)

func TestNewBuildScript(t *testing.T) {
	tests := []struct {
		name     string
		folder   string
		profile  string
		target   string
		features []string
		want     []string
		verbose  bool
	}{
		{
			name:     "basic build",
			folder:   ".",
			profile:  "release",
			target:   "",
			features: nil,
			verbose:  false,
			want:     []string{"cargo build", "--release", "cargo test"},
		},
		{
			name:     "debug build",
			folder:   ".",
			profile:  "debug",
			target:   "",
			features: nil,
			verbose:  false,
			want:     []string{"cargo build", "cargo test"},
		},
		{
			name:     "with target",
			folder:   ".",
			profile:  "release",
			target:   "x86_64-unknown-linux-musl",
			features: nil,
			verbose:  false,
			want:     []string{"cargo build", "--release", "--target x86_64-unknown-linux-musl"},
		},
		{
			name:     "with features",
			folder:   ".",
			profile:  "release",
			target:   "",
			features: []string{"feature1", "feature2"},
			verbose:  false,
			want:     []string{"cargo build", "--release", "--features feature1,feature2"},
		},
		{
			name:     "verbose build",
			folder:   ".",
			profile:  "release",
			target:   "",
			features: nil,
			verbose:  true,
			want:     []string{"set -xe", "cargo build", "--verbose"},
		},
		{
			name:     "with folder",
			folder:   "my-project",
			profile:  "release",
			target:   "",
			features: nil,
			verbose:  false,
			want:     []string{"cd my-project", "cargo build", "--release"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := NewBuildScript(tt.folder, tt.profile, tt.target, tt.features, tt.verbose, "/root/.cargo", []*types.PlatformSpec{})
			script := bs.Script()

			for _, expected := range tt.want {
				if !strings.Contains(script, expected) {
					t.Errorf("Expected script to contain %q, but it doesn't. Script:\n%s", expected, script)
				}
			}
		})
	}
}

func TestBuildScriptDefaults(t *testing.T) {
	// Test that empty folder defaults to "."
	bs := NewBuildScript("", "release", "", nil, false, "/root/.cargo", []*types.PlatformSpec{})
	if bs.Folder != "." {
		t.Errorf("Expected folder to default to '.', got %q", bs.Folder)
	}
}

func TestCargoBuildCmds(t *testing.T) {
	tests := []struct {
		name     string
		bs       *BuildScript
		wantCmds []string
	}{
		{
			name: "release build",
			bs: &BuildScript{
				Folder:  ".",
				Profile: "release",
			},
			wantCmds: []string{"cargo build --color never --release", "cargo test --color never --release"},
		},
		{
			name: "debug build",
			bs: &BuildScript{
				Folder:  ".",
				Profile: "debug",
			},
			wantCmds: []string{"cargo build --color never", "cargo test --color never"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := cargoBuildCmds(tt.bs)
			for _, wantCmd := range tt.wantCmds {
				if !strings.Contains(cmds, wantCmd) {
					t.Errorf("Expected commands to contain %q, but it doesn't. Commands:\n%s", wantCmd, cmds)
				}
			}
		})
	}
}
