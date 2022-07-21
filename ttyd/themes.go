package ttyd

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

var DefaultTheme = ""

func ThemeList(fsys fs.FS) map[string]string {
	ret := map[string]string{".": DefaultTheme}
	if _, err := fs.Stat(fsys, "themes"); err != nil {
		return ret
	}
	entries, err := fs.ReadDir(fsys, "themes")
	if err != nil {
		fmt.Println(err)
		return ret
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".js" {
			continue
		}
		ret[name[:len(name)-3]] = "/themes/" + name
	}
	return ret
}
