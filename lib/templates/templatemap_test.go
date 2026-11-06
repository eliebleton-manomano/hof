package templates

import (
	"slices"
	"testing"
	"testing/fstest"
)

func TestImportFromFS(t *testing.T) {
	fsys := fstest.MapFS{
		"files/a.txt":       {Data: []byte("a {{ .x }}")},
		"files/.gitignore":  {Data: []byte("bin/")},
		"files/sub/b.txt":   {Data: []byte("b [[ .x ]]")},
		"other/ignored.txt": {Data: []byte("ignored")},
	}

	M := NewTemplateMap()
	err := M.ImportFromFS(fsys, "files/**/*", "files", Delims{}, map[string]Delims{
		"files/sub/*": {LHS: "[[", RHS: "]]"},
	})
	if err != nil {
		t.Fatal(err)
	}

	var keys []string
	for k := range M {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	want := []string{".gitignore", "a.txt", "sub/b.txt"}
	if !slices.Equal(keys, want) {
		t.Fatalf("got %q, want %q", keys, want)
	}

	// the delim override applies: b.txt renders its [[ ]] action
	out, err := M["sub/b.txt"].Render(map[string]any{"x": "y"})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "b y" {
		t.Fatalf("got %q, want %q", out, "b y")
	}

	// as with zglob, a missing literal path is an error
	if err := M.ImportFromFS(fsys, "nope/**/*", "nope", Delims{}, nil); err == nil {
		t.Fatal("want error for a missing directory")
	}
}
