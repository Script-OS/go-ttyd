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
)

type TTYd struct {
	mux *http.ServeMux
}

type CmdGenerator func() *exec.Cmd

var upgrader = websocket.Upgrader{} // use default options

func ws(w http.ResponseWriter, r *http.Request, gen CmdGenerator) {
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
		err := ServePTY(c, gen())
		if err != nil {
			log.Println(err)
		}
	}()
}

type Config struct {
	OtherFSList []fs.FS
	Gen         CmdGenerator
}

func NewTTYd(conf Config) *TTYd {
	ttyd := &TTYd{
		mux: http.NewServeMux(),
	}
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
		ws(w, r, conf.Gen)
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
