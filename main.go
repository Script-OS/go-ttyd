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

func main() {
	port := flag.Int("p", 80, "port that http serve on")
	flag.Parse()
	cmdDesc := flag.Args()
	tty := ttyd.NewTTYd(ttyd.Config{
		OtherFS: nil,
		Gen: func() *exec.Cmd {
			cmd := exec.Command(cmdDesc[0], cmdDesc[1:]...)
			cmd.Env = append(os.Environ(),
				"TERM=xterm-webmedia-256color",
			)
			return cmd
		},
	})
	err := http.ListenAndServe(fmt.Sprintf(":%d", *port), tty)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
