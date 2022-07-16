package ttyd

import (
	"encoding/json"
	"github.com/azurity/go-onefile"
	"github.com/gorilla/websocket"
	"github.com/laher/mergefs"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"sync/atomic"
)

type TTYd struct {
	mux *http.ServeMux
}

type CmdGenerator func() *exec.Cmd

var upgrader = websocket.Upgrader{} // use default options

func ws(w http.ResponseWriter, r *http.Request, gen CmdGenerator, connCounter *int32, max int32) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	go func() {
		defer func() {
			_ = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		}()
		log.Println("client connect")
		c.SetCloseHandler(func(code int, text string) error {
			log.Printf("client disconnect, reason: %d\n", code)
			_ = c.Close()
			return nil
		})
		count := atomic.AddInt32(connCounter, 1)
		defer atomic.AddInt32(connCounter, -1)
		if max > 0 && count > max {
			err := "The number of connections reaches max threshold."
			_ = c.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31m"+err+"\x1b[0m"))
			log.Println(err)
			return
		}
		err := ServePTY(c, gen())
		if err != nil {
			_ = c.WriteMessage(websocket.BinaryMessage, []byte("\x1b[31m"+"Unable to start target program."+"\x1b[0m"))
			log.Println(err)
		}
	}()
}

type Config struct {
	OtherFSList []fs.FS
	Gen         CmdGenerator
	MaxConn     int32
}

func NewTTYd(conf Config) *TTYd {
	ttyd := &TTYd{
		mux: http.NewServeMux(),
	}
	if conf.MaxConn < 0 {
		conf.MaxConn = 0
	}
	connCounter := int32(0)
	frontend, _ := fs.Sub(frontendFS, "frontend")
	serveFS := frontend
	fsList := []fs.FS{frontend, ConfigFS}
	if conf.OtherFSList != nil {
		fsList = append(fsList, conf.OtherFSList...)
		serveFS = mergefs.Merge(fsList...)
	}
	ttyd.mux.Handle("/", onefile.New(serveFS, &onefile.Overwrite{
		Fsys: nil,
		Pair: map[string]string{},
	}, "/index.html"))
	ttyd.mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws(w, r, conf.Gen, &connCounter, conf.MaxConn)
	})
	ttyd.mux.HandleFunc("/themes.json", func(w http.ResponseWriter, r *http.Request) {
		themes := ThemeList()
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
