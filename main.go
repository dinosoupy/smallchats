package main

import (
	"log"
	"net/http"
	"time"
	"math/rand"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/google/uuid"
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

type Message struct {
	MsgType string `json: "MsgType"`
	Data string	`json:"Data"`
}

func main() {
	go match() 

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		http.Error(w, "404: Page not found", 404)
	}) // 404 to all http requests
	http.HandleFunc("/ws", serveWs)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func parseResponse(jsonResponse []byte) string {
	var msg string
	_ = json.Unmarshal(jsonResponse, &msg)
	return msg
}

// func match() {
// 	for {
// 		if len(queue) > 1 {
// 			clientA := dequeue()
// 			clientB := dequeue()
			
// 			roomID:=uuid.New().String()
// 			log.Println(roomID, clientA.clientID, clientB.clientID)

// 			candidateA, _:=json.Marshal(&Candidate{
// 				ClientID: clientA.clientID,
// 				RoomID: roomID,
// 			})

// 			candidateB, _:=json.Marshal(&Candidate{
// 				ClientID: clientB.clientID,
// 				RoomID: roomID,
// 			})

// 			log.Printf("Trying to match users %v and %v", clientA.clientID, clientB.clientID)
// 			log.Printf("Sending candidates %v and %v", string(candidateA), string(candidateB))

// 			clientB.send<-candidateA
// 			clientA.send<-candidateB
			
// 		} else {
// 			log.Printf("Not enough users to match. Online: %d user(s)", len(queue))
// 			time.Sleep(10 * time.Second)
// 		}
// 	}
// }

func askForApproval(clientA, clientB *Client, approvalChannel chan bool) {
	candidateA, _:=json.Marshal(&Message{
		MsgType: "candidate",
		Data: clientA.clientID,
	})

	candidateB, _:=json.Marshal(&Message{
		MsgType: "candidate",
		Data: clientB.clientID,
	})

	log.Printf("Trying to match users %v and %v", clientA.clientID, clientB.clientID)
	log.Printf("Sending candidates %v and %v", string(candidateA), string(candidateB))

	clientB.send<-candidateA
	clientA.send<-candidateB

	responseA := parseResponse(<-clientA.receive)
	responseB := parseResponse(<-clientB.receive)

	for responseA != "" && responseB != "" {
		if responseA == "accept" && responseB == "accept" {
			approvalChannel <- true
		} else {
			approvalChannel <- false
		}
	}
}

func match() {
	for {
		if len(queue) > 1 {
			var clientA, clientB *Client
			var randIndexA, randIndexB int
			var approvalChannel = make(chan bool)

			// select random clients
			for { 
				randIndexA=rand.Intn(len(queue))
				clientA = queue[randIndexA]
				randIndexB=rand.Intn(len(queue))
				clientB = queue[randIndexB]
				if clientB != clientA {
					break
				}
			}

			go askForApproval(clientA, clientB, approvalChannel)	

			log.Println("Waiting for approval")
			approval := <-approvalChannel

			if approval {
				// Both clients accepted the match
        // Remove clients from queue
        log.Printf("Match successful: %v and %v", clientA.clientID, clientB.clientID)
        queue = append(queue[:randIndexA], queue[randIndexA+1:]...)
        queue = append(queue[:randIndexB], queue[randIndexB+1:]...)

        // create room id
        roomID:=uuid.New().String()
				log.Println(roomID, clientA.clientID, clientB.clientID)

				// selected candidates become peers, both have common roomID
				roomMsg, _:=json.Marshal(&Message{
					MsgType: "room",
					Data: roomID,
				})

				clientB.send<-roomMsg
				clientA.send<-roomMsg

			} else {
				log.Println("At least one response was not 'accept'.")
				continue
			}
			
		} else {
			log.Printf("Not enough users to match. Online: %d user(s)", len(queue))
			time.Sleep(10 * time.Second)
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Upgrade(w, r, nil, 512, 512)

  if err != nil {
      http.Error(w, "Internal server error", http.StatusInternalServerError)
      return // stops execution if connection upgrade fails
  }

	userID := r.URL.Query().Get("userid") // wss://localhost:8080?userid=0123435

	sendChannel := make(chan []byte, 512) // each message is a slice of bytes, 512 messages can be stored in buffer
	receiveChannel := make(chan []byte, 512)

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
