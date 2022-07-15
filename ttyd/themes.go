package ttyd

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

var ConfigFS fs.FS

func init() {
	confDir, err := os.UserConfigDir()
	if err != nil {
		log.Panicln(err)
	}
	confDir = filepath.Join(confDir, "go-ttyd")
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		err := os.MkdirAll(confDir, 0777)
		if err != nil {
			log.Panicln(err)
		}
	}

	ConfigFS = os.DirFS(confDir)
}

func ThemeList() map[string]string {
	ret := map[string]string{}
	if _, err := fs.Stat(ConfigFS, "themes"); err != nil {
		return ret
	}
	entries, err := fs.ReadDir(ConfigFS, "themes")
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
