package main

import (
	"encoding/binary"
	"encoding/json"
	"github.com/azurity/go-onefile"
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var upgrader = websocket.Upgrader{} // use default options

const (
	DATA_BUFFER int = 1
	SIZE_BUFFER     = 2
)

type SizeMeta struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

type WsProtocol struct {
	*websocket.Conn
}

func (c *WsProtocol) Recv() (interface{}, error) {
	_, msg, err := c.ReadMessage()
	if err != nil {
		return nil, err
	}
	t := int(binary.LittleEndian.Uint32(msg[0:4]))
	switch t {
	case DATA_BUFFER:
		return msg[4:], nil
	case SIZE_BUFFER:
		out := &SizeMeta{}
		_ = json.Unmarshal(msg[4:], out)
		return out, nil
	default:
		return nil, nil
	}
}

func serveConn(c *websocket.Conn) {
	defer c.Close()
	protocol := &WsProtocol{c}
	cmd := exec.Command("bash")
	f, err := pty.Start(cmd)
	if err != nil {
		log.Println("run:", err)
		return
	}
	go func() {
		for {
			msg, err := protocol.Recv()
			//_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}
			if data, ok := msg.([]byte); ok {
				_, err = f.Write(data)
				if err != nil {
					log.Println("read:", err)
					break
				}
			} else if data, ok := msg.(*SizeMeta); ok {
				_ = pty.Setsize(f, &pty.Winsize{
					Rows: uint16(data.Rows),
					Cols: uint16(data.Cols),
				})
			}
		}
	}()

	buf := make([]byte, 1024)
	for {
		n, err := f.Read(buf)
		if n > 0 {
			err = c.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Println("write:", err)
				break
			}
		}
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func ws(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	go serveConn(c)
}

func main() {
	http.Handle("/", onefile.New(os.DirFS("frontend"), &onefile.Overwrite{
		Fsys: nil,
		Pair: map[string]string{},
	}, "/index.html"))
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws(w, r)
	})
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
