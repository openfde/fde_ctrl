package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type hub struct {
	connections map[*connection]bool
	broadcast   chan []byte
	register    chan *connection
	unregister  chan *connection
	mutex       sync.Mutex
}

func (h *hub) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Error upgrading connection: %v", err)
		return
	}
	c := &connection{ws: ws, send: make(chan []byte, 256)}
	h.register <- c
	defer func() { h.unregister <- c }()
	go c.writePump()
	c.readPump(h)
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type connection struct {
	ws   *websocket.Conn
	send chan []byte
}

func newHub() *hub {
	return &hub{
		connections: make(map[*connection]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *connection),
		unregister:  make(chan *connection),
	}
}

func (h *hub) run() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println("recover in hub run")
			return
		}
	}()
	for {
		select {
		case conn := <-h.register:
			h.mutex.Lock()
			h.connections[conn] = true
			h.mutex.Unlock()
		case conn := <-h.unregister:
			if _, ok := h.connections[conn]; ok {
				close(conn.send)
				h.mutex.Lock()
				delete(h.connections, conn)
				h.mutex.Unlock()
			}
		case message := <-h.broadcast:
			h.mutex.Lock()
			for conn := range h.connections {
				select {
				case conn.send <- message:
				default:
					close(conn.send)
					delete(h.connections, conn)
				}
			}
			h.mutex.Unlock()
		}
	}
}

func (h *hub) broadcastHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("hello")
	w.Write([]byte("Hello, World!"))
	h.broadcast <- []byte("new broadcast")
	return
}

func (c *connection) readPump(h *hub) {
	defer func() {
		h.unregister <- c
		c.ws.Close()
	}()
	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		// h.broadcast <- message
		fmt.Printf("Received message: %s\n", message)
	}
}

func (c *connection) writePump() {
	defer c.ws.Close()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			err := c.ws.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Printf("error: %v", err)
				return
			}
		}
	}
}
