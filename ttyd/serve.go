package ttyd

import (
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"log"
	"os/exec"
	"runtime"
)

func logError(err error) {
	_, path, line, ok := runtime.Caller(1)
	if !ok {
		log.Println(err)
	} else {
		log.Printf("[%s:%d] %s\n", path, line, err.Error())
	}
}

func ServePTY(c *websocket.Conn, cmd *exec.Cmd) error {
	conn := &WsProtocol{c}
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	go func() {
		for {
			msg, err := conn.Recv()
			if err != nil {
				logError(err)
				break
			}
			if data, ok := msg.([]byte); ok {
				_, err = ptyFile.Write(data)
				if err != nil {
					logError(err)
					break
				}
			} else if data, ok := msg.(*SizeMeta); ok {
				_ = pty.Setsize(ptyFile, &pty.Winsize{
					Rows: uint16(data.Rows),
					Cols: uint16(data.Cols),
				})
			}
		}
	}()
	buf := make([]byte, 1024)
	for {
		n, err := ptyFile.Read(buf)
		if n > 0 {
			err = c.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				logError(err)
				break
			}
		}
		if err != nil {
			logError(err)
			break
		}
	}
	return nil
}
