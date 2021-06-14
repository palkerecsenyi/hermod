package service

import (
	"github.com/gorilla/websocket"
	"log"
)

func wsSendError(conn *websocket.Conn, err error) {
	e := conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
	if e != nil {
		log.Println(e)
	}
}

func wsSendBinary(conn *websocket.Conn, data *[]byte) {
	_ = conn.WriteMessage(websocket.BinaryMessage, *data)
}
