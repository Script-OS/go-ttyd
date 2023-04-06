package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/Script-OS/go-ttyd/ttyd"
	"golang.org/x/term"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
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
	url.Host = req.Host
	target := url.String()
	//log.Println("url:", url, "target:", target, "host:", req.Host)
	http.Redirect(w, req, target,
		// see comments below and consider the codes 308, 302, or 301
		http.StatusTemporaryRedirect)
}

const CredentialBinName = "@login"

func doCredential(credential string, command []string) {
	faultCounter := 1
	checked := func() bool {
		for faultCounter <= 3 {
			const prompt = "Enter password:"
			oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
			if err != nil {
				return false
			}
			defer term.Restore(int(os.Stdin.Fd()), oldState)
			fmt.Print(prompt)
			password := []rune{}
			rd := bufio.NewReader(os.Stdin)
			for {
				if c, _, err := rd.ReadRune(); err != nil {
					if err == io.EOF {
						break
					} else {
						fmt.Print("\r\n")
						fmt.Print(err)
						fmt.Print("\r\n")
						return false
					}
				} else {
					finish := false
					switch c {
					case '\r':
						fallthrough
					case '\n':
						fallthrough
					case 0x04:
						finish = true
						break
					case 0x03:
						fmt.Print("^C\r\n")
						return false
					default:
						password = append(password, c)
					}
					if finish {
						break
					}
				}
			}
			if string(password) == credential {
				fmt.Print("\033[2J\033[1;1H")
				return true
			} else {
				time.Sleep(time.Second * 1)
				fmt.Printf("\r\nWrong password. You tried %d/3 times\r\n", faultCounter)
				faultCounter++
			}
		}
		fmt.Print("\033[31mDisconnected\033[0m\r\n")
		return false
	}()
	if checked {
		cmd := exec.Command(command[0], command[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
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
	if os.Args[0] == CredentialBinName {
		doCredential(os.Args[1], os.Args[2:])
		return
	}
	host := flag.String("host", "", "host that http serve on, default is 0.0.0.0 ")
	port := flag.Int("port", 0, "port that http serve on")
	theme := flag.String("theme", "", "default theme")
	SSL := flag.Bool("SSL", false, "use SSL or not, default is false")
	crtFile := flag.String("crt", "https.crt", "path to https crt file")
	keyFile := flag.String("key", "https.key", "path to https key file")
	max := flag.Int("max", 0, "max number of connections, 0 means no limit")
	credential := flag.String("credit", "", "credential for authentication")
	noCheckOrigin := flag.Bool("no-check-origin", false, "do not check origin")
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

	generator := func() *exec.Cmd {
		cmd := exec.Command(cmdDesc[0], cmdDesc[1:]...)
		cmd.Env = append(os.Environ(),
			fmt.Sprintf("TERM=%s", newTermName),
			fmt.Sprintf("TERMINFO=%s", infoDir),
		)
		return cmd
	}

	if *credential != "" {
		implGen := generator
		generator = func() *exec.Cmd {
			cmd := implGen()
			cmd.Path = os.Args[0]
			cmd.Args = append([]string{CredentialBinName, *credential}, cmd.Args...)
			return cmd
		}
	}

	var originChecker func(r *http.Request) bool = nil
	if *noCheckOrigin {
		originChecker = func(r *http.Request) bool { return true }
	}

	tty := ttyd.NewTTYd(ttyd.Config{
		OtherFSList: fsList,
		Gen:         generator,
		MaxConn:     int32(*max),
		CheckOrigin: originChecker,
	})
	portString := fmt.Sprintf("%s:%d", *host, *port)
	_, crtErr := os.Stat(*crtFile)
	_, keyErr := os.Stat(*keyFile)
	if *SSL && (crtErr == nil && keyErr == nil) {
		var jumpPortString string
		if *port == 0 {
			portString = fmt.Sprintf("%s:%d", *host, 443)
		} else {
			jumpPortString = fmt.Sprintf("%s:%d", *host, *port+1)
			log.Println("[+]http redirect addr is", jumpPortString)
		}
		go func() {
			http.ListenAndServe(jumpPortString, http.HandlerFunc(Redirect))
		}()
		log.Fatal(http.ListenAndServeTLS(portString, *crtFile, *keyFile, tty))
	} else {
		if *port == 0 {
			portString = fmt.Sprintf("%s:%d", *host, 80)
		}
		log.Fatal(http.ListenAndServe(portString, tty))
	}
}
