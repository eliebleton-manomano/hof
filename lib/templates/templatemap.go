package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/mattn/go-zglob"
)

type TemplateMap map[string]*Template

func NewTemplateMap() TemplateMap {
	return make(map[string]*Template)
}

func CreateTemplateMapFromFolder(glob, prefix string, delims Delims, delimMap map[string]Delims) (tplMap TemplateMap, err error) {
	tplMap = NewTemplateMap()
	err = tplMap.ImportFromFolder(glob, prefix, delims, delimMap)
	if err != nil {
		return nil, fmt.Errorf("while importing %q: %w", glob, err)
	}
	return tplMap, nil
}

func CreateTemplateMapFromFS(fsys fs.FS, glob, prefix string, delims Delims, delimMap map[string]Delims) (tplMap TemplateMap, err error) {
	tplMap = NewTemplateMap()
	err = tplMap.ImportFromFS(fsys, glob, prefix, delims, delimMap)
	if err != nil {
		return nil, fmt.Errorf("while importing %q: %w", glob, err)
	}
	return tplMap, nil
}

func (M TemplateMap) ImportTemplateFile(filename string, delims Delims) error {
	return M.importTemplate(filename, "", delims)
}

func (M TemplateMap) ImportFromFolder(glob, prefix string, delims Delims, delimMap map[string]Delims) error {
	// all templates
	matches, err := zglob.Glob(glob)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		// return fmt.Errorf("No templates found for '%s'", glob)
		return nil
	}
	// fmt.Printf("matches %q %q %d\n", glob, prefix, len(matches))

	// delimOverrides
	overrides := make(map[string]Delims)
	for g, d := range delimMap {
		gmatches, err := zglob.Glob(g)
		if err != nil {
			return err
		}
		if len(gmatches) > 0 {
			for _, fn := range gmatches {
				overrides[fn] = d
			}
		}
	}

	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			return err
		}

		if info.IsDir() {
			continue
		}

		// override delims?
		d := delims
		if len(overrides) > 0 {
			D, ok := overrides[match]
			if ok {
				d = D
			}
		}

		err = M.importTemplate(match, prefix, d)
		if err != nil {
			return err
		}
	}

	return nil
}

// ImportFromFS is ImportFromFolder reading fsys, so globs and prefix are fs.FS paths
func (M TemplateMap) ImportFromFS(fsys fs.FS, glob, prefix string, delims Delims, delimMap map[string]Delims) error {
	// all templates
	matches, err := FSGlob(fsys, glob)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return nil
	}

	// delimOverrides
	overrides := make(map[string]Delims)
	for g, d := range delimMap {
		gmatches, err := FSGlob(fsys, g)
		if err != nil {
			return err
		}
		for _, fn := range gmatches {
			overrides[fn] = d
		}
	}

	for _, match := range matches {
		info, err := fs.Stat(fsys, match)
		if err != nil {
			return err
		}

		if info.IsDir() {
			continue
		}

		// override delims?
		d := delims
		if D, ok := overrides[match]; ok {
			d = D
		}

		source, err := fs.ReadFile(fsys, match)
		if err != nil {
			return err
		}

		err = M.addTemplate(match, string(source), prefix, d)
		if err != nil {
			return err
		}
	}

	return nil
}

// FSGlob globs fsys the way zglob.Glob globs the disk: symlinked directories are not walked,
// and a pattern whose leading literal part does not exist is an error.
// Unlike zglob, a trailing ** matches at any depth.
func FSGlob(fsys fs.FS, glob string) ([]string, error) {
	return doublestar.Glob(fsys, glob, doublestar.WithNoFollow(), doublestar.WithFailOnPatternNotExist())
}

func (M TemplateMap) importTemplate(filePath, prefix string, delims Delims) error {
	// fmt.Println("import:", filePath, prefix)
	source, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return M.addTemplate(filePath, string(source), prefix, delims)
}

func (M TemplateMap) addTemplate(filePath, content, prefix string, delims Delims) error {
	T, err := CreateFromString(filePath, content, delims)
	if err != nil {
		return fmt.Errorf("while parsing template file: %s\n%w", filePath, err)
	}

	// clean up filename before inserting into map, so when users reference in their generators, we align
	filePath = strings.TrimPrefix(filePath, prefix)
	filePath = strings.TrimPrefix(filePath, "/") // be resilent to trailing slashes in the last value, or the lack therein
	filePath = filepath.Clean(filePath)

	T.Name = filePath
	// fmt.Println("  final:", filePath)

	M[filePath] = T
	return nil
}
