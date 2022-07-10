package ttyd

import (
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
	OtherFS fs.FS
	Gen     CmdGenerator
}

func NewTTYd(conf Config) *TTYd {
	ttyd := &TTYd{
		mux: http.NewServeMux(),
	}
	frontend, _ := fs.Sub(frontendFS, "frontend")
	serveFS := frontend
	if conf.OtherFS != nil {
		serveFS = mergefs.Merge(frontend, conf.OtherFS)
	}
	ttyd.mux.Handle("/", onefile.New(serveFS, &onefile.Overwrite{
		Fsys: nil,
		Pair: map[string]string{},
	}, "/index.html"))
	ttyd.mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws(w, r, conf.Gen)
	})
	return ttyd
}

func (ttyd *TTYd) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ttyd.mux.ServeHTTP(w, r)
}
