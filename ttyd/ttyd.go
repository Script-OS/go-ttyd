package ttyd

import (
	"encoding/json"
	"github.com/azurity/go-onefile"
	"github.com/gorilla/websocket"
	//"github.com/laher/mergefs"
	"github.com/Script-OS/go-ttyd/mergefs"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
)

var ConfigFS fs.FS

var Logger = log.New(os.Stderr, "", log.LstdFlags)

func init() {
	confDir, err := os.UserConfigDir()
	if err != nil {
		Logger.Panicln(err)
	}
	confDir = filepath.Join(confDir, "go-ttyd")
	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		err := os.MkdirAll(confDir, 0777)
		if err != nil {
			Logger.Panicln(err)
		}
	}

	ConfigFS = os.DirFS(confDir)

	mergefs.Logger.SetOutput(ioutil.Discard)
}

type TTYd struct {
	mux      *http.ServeMux
	upgrader websocket.Upgrader
}

type CmdGenerator func() *exec.Cmd

func ws(upgrader websocket.Upgrader, w http.ResponseWriter, r *http.Request, gen CmdGenerator, connCounter *int32, max int32) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		Logger.Print("upgrade:", err)
		return
	}
	go func() {
		Logger.Println("client connect")
		c.SetCloseHandler(func(code int, text string) error {
			Logger.Printf("client disconnect, reason: %d\n", code)
			return nil
		})
		count := atomic.AddInt32(connCounter, 1)
		defer atomic.AddInt32(connCounter, -1)
		if max > 0 && count > max {
			err := "The number of connections reaches max threshold."
			_ = c.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31m"+err+"\x1b[0m"))
			_ = c.Close()
			Logger.Println(err)
			return
		}
		err := ServePTY(c, gen())
		if err != nil {
			_ = c.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31m"+"Unable to start target program."+"\x1b[0m"))
			_ = c.Close()
			Logger.Println(err)
			return
		}
		_ = c.Close()
		Logger.Println("client finished")
	}()
}

type Config struct {
	OtherFSList []fs.FS      // Other fs that need to be served as static files.
	Gen         CmdGenerator // A generator that creates the actual command.
	MaxConn     int32        // Maximum number of connections. Unlimited if <= 0.
	CheckOrigin func(r *http.Request) bool
}

func NewTTYd(conf Config) *TTYd {
	ttyd := &TTYd{
		mux:      http.NewServeMux(),
		upgrader: websocket.Upgrader{CheckOrigin: conf.CheckOrigin},
	}
	if conf.MaxConn < 0 {
		conf.MaxConn = 0
	}
	connCounter := int32(0)
	frontend, _ := fs.Sub(frontendFS, "frontend")
	fsList := []fs.FS{frontend, ConfigFS}
	if conf.OtherFSList != nil {
		fsList = append(fsList, conf.OtherFSList...)
	}
	serveFS := mergefs.Merge(fsList...)
	ttyd.mux.Handle("/", onefile.New(serveFS, &onefile.Overwrite{
		Fsys: nil,
		Pair: map[string]string{},
	}, "/index.html"))
	ttyd.mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws(ttyd.upgrader, w, r, conf.Gen, &connCounter, conf.MaxConn)
	})
	ttyd.mux.HandleFunc("/themes.json", func(w http.ResponseWriter, r *http.Request) {
		themes := ThemeList(serveFS)
		encoded, err := json.Marshal(&themes)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write(encoded)
		}
	})
	return ttyd
}

func (ttyd *TTYd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ttyd.mux.ServeHTTP(w, r)
}
