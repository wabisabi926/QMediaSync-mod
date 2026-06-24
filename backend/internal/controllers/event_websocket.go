package controllers

import (
	"net/http"

	gorillaws "github.com/gorilla/websocket"

	"Q115-STRM/internal/websocket"

	"github.com/gin-gonic/gin"
)

var eventUpgrader = gorillaws.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// EventWebSocket WebSocket 事件推送端点
func EventWebSocket(c *gin.Context) {
	conn, err := eventUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &websocket.Client{
		Hub:  websocket.GlobalEventHub,
		Conn: conn,
		Send: make(chan []byte, 256),
	}

	client.Hub.RegisterClient(client)

	go client.WritePump()
	go client.ReadPump()
}
