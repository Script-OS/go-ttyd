package ttyd

import (
	"encoding/binary"
	"encoding/json"
	"github.com/gorilla/websocket"
)

const (
	PROTOCOL_DATA uint32 = 1
	PROTOCOL_SIZE        = 2
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
	t := binary.LittleEndian.Uint32(msg[0:4])
	switch t {
	case PROTOCOL_DATA:
		return msg[4:], nil
	case PROTOCOL_SIZE:
		out := &SizeMeta{}
		_ = json.Unmarshal(msg[4:], out)
		return out, nil
	default:
		return nil, nil
	}
}
