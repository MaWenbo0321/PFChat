package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096 // 增加消息大小限制
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client WebSocket 客户端
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	userID   uint
	mu       sync.Mutex // 添加互斥锁保护连接写入
	isClosed bool
}

// Hub 管理所有客户端
type Hub struct {
	clients    map[uint]*Client
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex // 添加读写锁
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[uint]*Client),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 如果用户已存在连接，先关闭旧连接
			if oldClient, ok := h.clients[client.userID]; ok {
				oldClient.closeConnection()
				delete(h.clients, client.userID)
			}
			h.clients[client.userID] = client
			clientCount := len(h.clients)
			h.mu.Unlock()
			log.Printf("User %d connected, total clients: %d", client.userID, clientCount)

		case client := <-h.unregister:
			h.mu.Lock()
			if existingClient, ok := h.clients[client.userID]; ok {
				// 只有当是同一个客户端时才删除
				if existingClient == client {
					delete(h.clients, client.userID)
					client.closeConnection()
					log.Printf("User %d disconnected, total clients: %d", client.userID, len(h.clients))
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				if !client.trySend(message) {
					go func(c *Client) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// trySend 将“检查是否关闭”和“写入 channel”放在同一把锁下，
// 防止连接关闭时并发发送触发 send on closed channel。
func (c *Client) trySend(message []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isClosed {
		return false
	}
	select {
	case c.send <- message:
		return true
	default:
		return false
	}
}

// 关闭客户端连接
func (c *Client) closeConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.isClosed {
		c.isClosed = true
		close(c.send)
	}
}

// 发送消息给特定用户
func (h *Hub) sendToUser(userID uint, message []byte) {
	h.mu.RLock()
	client, ok := h.clients[userID]
	h.mu.RUnlock()

	if ok {
		if client.trySend(message) {
			log.Printf("Message sent to user %d", userID)
		} else {
			log.Printf("Failed to send message to user %d, channel full", userID)
			go func() {
				h.unregister <- client
			}()
		}
	} else {
		log.Printf("User %d not connected", userID)
	}
}

// WebSocket 连接处理
func serveWs(hub *Hub, c *gin.Context) {
	// 从 query 参数获取 token
	token := c.Query("token")
	if token == "" {
		c.JSON(401, gin.H{"error": "未提供token"})
		return
	}

	// 验证 token
	claims, err := parseTokenClaims(token)
	if err != nil {
		c.JSON(401, gin.H{"error": "token无效"})
		return
	}
	var user User
	if err := db.First(&user, claims.UserID).Error; err != nil || user.Role == RoleBot {
		c.JSON(401, gin.H{"error": "用户不存在或不可登录"})
		return
	}

	// 升级为 WebSocket 连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   claims.UserID,
		isClosed: false,
	}

	client.hub.register <- client

	// 启动读写协程
	go client.writePump()
	go client.readPump()
}

// 读取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %d: %v", c.userID, err)
			}
			break
		}

		// 处理接收到的消息
		c.handleMessage(message)
	}
}

// 发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.mu.Lock()
			if c.isClosed {
				c.mu.Unlock()
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				c.mu.Unlock()
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

		case <-ticker.C:
			c.mu.Lock()
			if c.isClosed {
				c.mu.Unlock()
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
		}
	}
}

// 处理接收到的消息
func (c *Client) handleMessage(message []byte) {
	var wsMsg WSMessage
	if err := json.Unmarshal(message, &wsMsg); err != nil {
		log.Println("JSON unmarshal error:", err)
		return
	}

	switch wsMsg.Type {
	case "ping":
		log.Printf("Received ping from user %d", c.userID)
	}
}
