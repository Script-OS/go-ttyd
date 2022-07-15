package main

import (
	"flag"
	"fmt"
	"go-ttyd/ttyd"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const newTermName = "xterm-webmedia-256color"

func prepareTerminfo() string {
	dir := os.TempDir() + "/go-ttyd/terminfo"
	_ = os.MkdirAll(dir+"/x", 0777)
	_ = os.Symlink("/usr/lib/terminfo/x/xterm-256color", dir+"/x/"+newTermName)
	return dir
}

func Redirect(w http.ResponseWriter, req *http.Request) {
	// remove/add not default ports from req.Host
	target := "https://" + req.Host + req.URL.Path
	if len(req.URL.RawQuery) > 0 {
		target += "?" + req.URL.RawQuery
	}
	log.Printf("nofound to: %s", target)
	http.Redirect(w, req, target,
		// see comments below and consider the codes 308, 302, or 301
		http.StatusTemporaryRedirect)
}

type StringArray []string

func (arr *StringArray) String() string {
	return strings.Join(*arr, "\n")
}

func (arr *StringArray) Set(value string) error {
	*arr = append(*arr, value)
	return nil
}

func main() {
	port := flag.Int("p", 0, "port that http serve on")
	SSL := flag.Bool("SSL", false, "open SSL or not, default is true")
	crtFile := flag.String("crt", "https.crt", "path to https crt file")
	keyFile := flag.String("key", "https.key", "path to https key file")
	statics := StringArray{}
	flag.Var(&statics, "static", "folder to provide extra static files")

	flag.Parse()
	cmdDesc := flag.Args()

	infoDir := prepareTerminfo()

	fsList := []fs.FS{}
	for _, path := range statics {
		fsList = append(fsList, os.DirFS(path))
	}

	tty := ttyd.NewTTYd(ttyd.Config{
		OtherFSList: fsList,
		Gen: func() *exec.Cmd {
			cmd := exec.Command(cmdDesc[0], cmdDesc[1:]...)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("TERM=%s", newTermName),
				fmt.Sprintf("TERMINFO=%s", infoDir),
			)
			return cmd
		},
	})
	portString := fmt.Sprintf(":%d", *port)
	_, crtErr := os.Stat(*crtFile)
	_, keyErr := os.Stat(*keyFile)
	if *SSL && (crtErr == nil && keyErr == nil) {
		if *port == 0 {
			portString = fmt.Sprintf(":%d", 443)
		}
		go func() {
			http.ListenAndServe(":80", http.HandlerFunc(Redirect))
		}()
		log.Fatal(http.ListenAndServeTLS(portString, *crtFile, *keyFile, tty))
	} else {
		if *port == 0 {
			portString = fmt.Sprintf(":%d", 80)
		}
		log.Fatal(http.ListenAndServe(portString, tty))
	}
}
