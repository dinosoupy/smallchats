package main

import (
	"log"
	"net/http"
	"time"
	"bytes"
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
	receive 	chan []byte // channel for reciving messages
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

			candidateA.send <- []byte("requestOffer,")
			if response, ok:= <-candidateA.receive; ok {
				idx:=bytes.IndexRune(response, ',')
				// messageType:=string(response[:idx])
				messageBody:=string(response[idx+1:])
				// log.Println(messageType, messageBody)

				candidateB.send <- []byte("requestAnswer," + messageBody)
				if response, ok:= <-candidateB.receive; ok {
					idx:=bytes.IndexRune(response, ',')
					// messageType:=string(response[:idx])
					messageBody:=string(response[idx+1:])
					// log.Println(messageType, messageBody)

					candidateA.send <- []byte("answer," + messageBody)	

					go trickleIce(candidateA, candidateB)

					continue		
				}
			}
			
		} else {
			log.Println("Not enough users to match")
			time.Sleep(10 * time.Second)
		}
	}
}

// returns the messageType and messageBody from a give message of type []byte
func getMessageBody(message []byte) (string, string) {
	idx:=bytes.IndexRune(message, ',')
	messageType:=string(message[:idx])
	messageBody:=string(message[idx+1:])

	return messageType, messageBody
}

func trickleIce(candidateA, candidateB *Client) {
	responseA:= <-candidateA.receive
	_, messageA:= getMessageBody(responseA)

	responseB:= <-candidateB.receive
	_, messageB:= getMessageBody(responseB)
	
	log.Printf("Trickling ICE now: \n Ice candidate from A: %s \n Ice candidate from B: %s\n", messageA, messageB)

	for messageA!="" || messageB!="" {
		candidateB.send<-[]byte("iceCandidate," + messageA)
		candidateA.send<-[]byte("iceCandidate," + messageB)
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 131072, 131072)

  if err != nil {
      http.Error(w, "Internal server error", http.StatusInternalServerError)
      return // stops execution if connection upgrade fails
  }

	userID := r.URL.Query().Get("userid") // wss://localhost:8080?userid=0123435

	sendChannel := make(chan []byte, 131072) // each message is a slice of bytes, 131072 messages can be stored in buffer
	receiveChannel := make(chan []byte, 131072)

	newClient := &Client{
		clientID: userID,
		conn:     conn,
		send:     sendChannel,
		receive: receiveChannel,
	}

	enqueue(newClient)

	go newClient.writePump()
	go newClient.readPump()
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

func (c *Client) readPump() {
	defer func() {
		log.Println("Closed at location 1")
		c.conn.Close()
	}()
	c.conn.SetReadLimit(131072)
	c.conn.SetReadDeadline(time.Now().Add(60*time.Second))
	c.conn.SetPongHandler(func(string) error {
		log.Println("Closed at location 2")
		c.conn.SetReadDeadline(time.Now().Add(60*time.Second))
		return nil
	})
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println("Closed at location 3")
				log.Printf("UnexpectedCloseError: %v", err)
			}
			break
		}

		// slice upto the first ',' - this is the messageType
		// 
		// messageType := string(message[:idx])
		// messageBody:=string(message[idx+1:])

		c.receive <- message
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		log.Println("Closed at location 4")
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
			if !ok {
				log.Println("Closed at location 5")
				// The server closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return			
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Closed at location 6")
				log.Println(message)
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
				log.Println("Closed at location 7")
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10*time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Println("Closed at location 8")
				return
			}
		}
	}
}
