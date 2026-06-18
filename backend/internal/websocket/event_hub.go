package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 事件类型常量
const (
	EventScraperTaskStart     = "scraper_task_start"
	EventScraperTaskComplete  = "scraper_task_complete"
	EventScraperItemComplete  = "scraper_item_complete"
	EventStrmSyncTaskStart    = "strm_sync_task_start"
	EventStrmSyncTaskComplete = "strm_sync_task_complete"
)

// WSEvent WebSocket事件结构
type WSEvent struct {
	EventType string    `json:"event_type"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data"`
}

// Client WebSocket客户端
type Client struct {
	Hub  *EventHub
	Conn *websocket.Conn
	Send chan []byte
}

// EventHub WebSocket事件中心，管理所有连接和事件广播
type EventHub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mutex      sync.RWMutex
}

// 全局事件中心实例
var GlobalEventHub *EventHub

// NewEventHub 创建新的事件中心
func NewEventHub() *EventHub {
	return &EventHub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run 启动事件中心
func (h *EventHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// 发送失败，关闭连接
					h.mutex.RUnlock()
					h.mutex.Lock()
					delete(h.clients, client)
					close(client.Send)
					h.mutex.Unlock()
					h.mutex.RLock()
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// BroadcastEvent 广播事件到所有客户端
func BroadcastEvent(eventType string, data any) {
	if GlobalEventHub == nil {
		return
	}
	event := WSEvent{
		EventType: eventType,
		Timestamp: time.Now(),
		Data:      data,
	}
	msg, err := json.Marshal(event)
	if err != nil {
		return
	}
	GlobalEventHub.broadcast <- msg
}

// RegisterClient 注册客户端
func (h *EventHub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient 注销客户端
func (h *EventHub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// ClientCount 获取当前连接数
func (h *EventHub) ClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// WritePump 从hub读取消息并发送到WebSocket连接
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		message, ok := <-c.Send
		if !ok {
			// channel已关闭
			c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}
		if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
			return
		}
	}
}

// ReadPump 从WebSocket连接读取消息（心跳/断开检测）
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.UnregisterClient(c)
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(512)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
	}
}
