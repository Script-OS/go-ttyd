package ttyd

import (
	"github.com/creack/pty"
	"github.com/gorilla/websocket"
	"os/exec"
	"runtime"
)

func logError(err error) {
	_, path, line, ok := runtime.Caller(1)
	if !ok {
		Logger.Println(err)
	} else {
		Logger.Printf("[%s:%d] %s\n", path, line, err.Error())
	}
}

func ServePTY(c *websocket.Conn, cmd *exec.Cmd, finCh chan bool) error {
	conn := &WsProtocol{c}
	ptyFile, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer ptyFile.Close()
	go func() {
		defer cmd.Process.Kill()
		for {
			msg, err := conn.Recv()
			if _, ok := err.(*websocket.CloseError); ok {
				break
			} else if err != nil {
				logError(err)
				break
			}
			if data, ok := msg.([]byte); ok {
				_, err = ptyFile.Write(data)
				if err != nil {
					break
				}
			} else if data, ok := msg.(*SizeMeta); ok {
				_ = pty.Setsize(ptyFile, &pty.Winsize{
					Rows: uint16(data.Rows),
					Cols: uint16(data.Cols),
				})
			}
		}
		finCh <- true
	}()
	go func() {
		defer cmd.Process.Kill()
		buf := make([]byte, 1024)
		for {
			n, err := ptyFile.Read(buf)
			if n > 0 {
				err = c.WriteMessage(websocket.BinaryMessage, buf[:n])
				if _, ok := err.(*websocket.CloseError); ok {
					break
				} else if err != nil {
					logError(err)
					break
				}
			}
			if err != nil {
				break
			}
		}
		finCh <- true
	}()
	_ = cmd.Wait()
	return nil
}
