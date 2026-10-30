package gen

import "testing"

func TestCalcBasePath(t *testing.T) {
	deps := map[string]string{
		"root.test/mod":                  "/root",
		"example.com/foo":                "/cache/foo",
		"github.com/org/repo":            "/cache/repo",
		"git.example.com/group/sub/repo": "/cache/deep",
		"git.example.com/group":          "/cache/group",
		"github.com/org/repo-other":      "/cache/other",
	}

	tests := []struct {
		module  string
		want    string
		wantErr bool
	}{
		{module: "", want: "/root"},
		{module: "root.test/mod", want: "/root"},
		{module: "example.com/foo", want: "/cache/foo"},
		{module: "github.com/org/repo", want: "/cache/repo"},
		{module: "git.example.com/group/sub/repo", want: "/cache/deep"},
		// sub-packages inside a module
		{module: "example.com/foo/gen", want: "/cache/foo"},
		{module: "github.com/org/repo/gen/sub", want: "/cache/repo"},
		{module: "git.example.com/group/sub/repo/gen", want: "/cache/deep"},
		// shorter key still wins when the longer one doesn't match
		{module: "git.example.com/group/other", want: "/cache/group"},
		// prefix only at a path boundary
		{module: "github.com/org/repo-other/gen", want: "/cache/other"},
		{module: "github.com/org/rep", wantErr: true},
		{module: "example.com", wantErr: true},
		{module: "unknown.com/x/y/z", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			G := &Generator{
				CueModuleRoot:  "/root",
				RootModuleName: "root.test/mod",
				ModuleName:     tt.module,
				DepMapping:     deps,
			}
			got, err := G.calcBasePath()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("want error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
