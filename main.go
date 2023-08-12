package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var queue []*Client // global queue

var (
	newline = []byte{'\n'}
	space = []byte{' '}
)

type Client struct {
	clientID string          // userID in supabase
	conn     *websocket.Conn // websocket connection object
	send     chan []byte    // channel for sending messages
}

func main() {
	go match() 

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "404: Page not found", 404)
	}) // 404 to all http requests
	http.HandleFunc("/ws", serveWs)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func match() {
	for {
		if len(queue) > 1 {
			candidateA := dequeue()
			candidateB := dequeue()
			log.Println("Trying to match users", candidateA.clientID, "and", candidateB.clientID)

			candidateA.send <- []byte(candidateB.clientID)
			candidateB.send <- []byte(candidateA.clientID)
		} else {
			log.Println("Not enough users to match")
			time.Sleep(10 * time.Second)
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)

  if err != nil {
      http.Error(w, "Internal server error", http.StatusInternalServerError)
      return // stops execution if connection upgrade fails
  }

	userID := r.URL.Query().Get("userid") // wss://localhost:8080?userid=0123435

	sendChannel := make(chan []byte, 256) // each message is a slice of bytes, 256 messages can be stored in buffer

	newClient := &Client{
		clientID: userID,
		conn:     conn,
		send:     sendChannel,
	}

	enqueue(newClient)

	go newClient.writePump()
}

func enqueue(user *Client) {
	log.Println("New user added to queue: ", user.clientID)
	queue = append(queue, user)
}

func dequeue() *Client {
	top := queue[0]
	if len(queue) == 1 {
		queue = make([]*Client, 0)
		return top
	} else {
		queue = queue[1:]
		return top
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
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
			c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
