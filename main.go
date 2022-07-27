package main

import (
	"flag"
	"fmt"
	"github.com/Script-OS/go-ttyd/ttyd"
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
	url := *req.URL
	url.Scheme = "https"
	target := url.String()
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
	theme := flag.String("theme", "", "default theme")
	SSL := flag.Bool("SSL", false, "use SSL or not, default is false")
	crtFile := flag.String("crt", "https.crt", "path to https crt file")
	keyFile := flag.String("key", "https.key", "path to https key file")
	max := flag.Int("max", 0, "max number of connections, 0 means no limit")
	statics := StringArray{}
	flag.Var(&statics, "static", "folder to provide extra static files")

	flag.CommandLine.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), "  go-ttyd [options] <command> [<args of your command>...]")
		fmt.Fprintln(flag.CommandLine.Output(), "Options:")
		flag.PrintDefaults()
	}

	flag.Parse()
	if flag.NArg() == 0 {
		flag.CommandLine.Usage()
		return
	}
	cmdDesc := flag.Args()

	infoDir := prepareTerminfo()

	fsList := []fs.FS{}
	for _, path := range statics {
		fsList = append(fsList, os.DirFS(path))
	}

	ttyd.DefaultTheme = *theme

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
		MaxConn: int32(*max),
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
