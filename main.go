package main

import (
	"flag"
	"fmt"
	"go-ttyd/ttyd"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const newTermName = "xterm-webmedia-256color"

func prepareTerminfo() string {
	dir := os.TempDir() + "/go-ttyd/terminfo"
	_ = os.MkdirAll(dir+"/x", 0777)
	_ = os.Symlink("/usr/lib/terminfo/x/xterm-256color", dir+"/x/"+newTermName)
	return dir
}

func main() {
	port := flag.Int("p", 80, "port that http serve on")
	flag.Parse()
	cmdDesc := flag.Args()

	infoDir := prepareTerminfo()

	tty := ttyd.NewTTYd(ttyd.Config{
		OtherFS: nil,
		Gen: func() *exec.Cmd {
			cmd := exec.Command(cmdDesc[0], cmdDesc[1:]...)
			cmd.Env = append(os.Environ(),
				fmt.Sprintf("TERM=%s", newTermName),
				fmt.Sprintf("TERMINFO=%s", infoDir),
			)
			return cmd
		},
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), tty)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
