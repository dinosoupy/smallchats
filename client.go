package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write message to peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to the peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9)/10

	// Maximum message size allowed from the peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
}

type Client struct {
	clientID	[]byte
	isBusy	bool
	ms *MatchingServer
	// Websocket connection
	conn *websocket.Conn
	// Buffered channel of outbound messages
	send chan []byte
}

// readPump pumps messages form the websocket connection to the matching server
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all 
// reads from this goroutine. 
func (c *Client) readPump() {
	defer func() {
		c.ms.unregister <- c
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
				log.Printf("UnexpectedCloseError: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.ms.broadcast <-message
	}
}

// writePump pumps messages from the matching server to the connections
// The application runs writePump in a per-connection goroutine. The application
// ensures that there is at most one writer on a connection by executing all 
// writes from this goroutine. 
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)7
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The server closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return			
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func serveWs(ms *MatchingServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{ms: ms, conn: conn, send: make(chan []byte, 256)}
	client.ms.register <- client

	// Allow collection of memory referenced by the caller by doing
	// all work in new goroutines
	go client.writePump()
	go client.readPump()

	// wait 30 seconds
	// select candidate from client list at random
	// keep sending candidate to client.send channel
	
}

