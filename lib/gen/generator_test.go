package gen

import (
	"maps"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/hofstadter-io/hof/lib/templates"
)

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

func TestGeneratorFS(t *testing.T) {
	G := &Generator{
		FS: fstest.MapFS{
			"files/a.txt":      {Data: []byte(`a {{ template "p.txt" . }}`)},
			"files/.gitignore": {Data: []byte("bin/")},
			"partials/p.txt":   {Data: []byte("p {{ .x }}")},
			".statics/s.txt":   {Data: []byte("s")},
		},
		CueModuleRoot: ".",
		Templates:     []*TemplateGlobs{{Globs: []string{"./files/**/*"}, TrimPrefix: "./files/"}},
		Partials:      []*TemplateGlobs{{Globs: []string{"./partials/**/*"}, TrimPrefix: "./partials/"}},
		// TrimPrefix "" cleans to ".": the static keeps its leading dot
		Statics:     []*StaticGlobs{{Globs: []string{"./.statics/**/*"}, OutPrefix: "out"}},
		PartialsMap: templates.NewTemplateMap(),
		TemplateMap: templates.NewTemplateMap(),
		Files:       make(map[string]*File),
	}

	var errs []error
	errs = append(errs, G.initStaticFiles()...)
	errs = append(errs, G.initPartials()...)
	errs = append(errs, G.initTemplates()...)
	if len(errs) > 0 {
		t.Fatal(errs)
	}

	keys := func(m templates.TemplateMap) []string {
		return slices.Sorted(maps.Keys(m))
	}
	if got, want := keys(G.TemplateMap), []string{".gitignore", "a.txt"}; !slices.Equal(got, want) {
		t.Fatalf("templates: got %q, want %q", got, want)
	}
	if got, want := keys(G.PartialsMap), []string{"p.txt"}; !slices.Equal(got, want) {
		t.Fatalf("partials: got %q, want %q", got, want)
	}
	if F := G.Files["out/.statics/s.txt"]; F == nil || string(F.RenderContent) != "s" {
		t.Fatalf("statics: got %v", slices.Sorted(maps.Keys(G.Files)))
	}

	out, err := G.TemplateMap["a.txt"].Render(map[string]any{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "a p y" {
		t.Fatalf("got %q, want %q", out, "a p y")
	}
}
