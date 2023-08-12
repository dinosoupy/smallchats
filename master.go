// An implementation that includes matching functionality as well and signalling functoinality
// It maintains list of users
// It serves ws requests
// It implements readPump and writePump for communicating with users
package main 

type Client struct {
	clientID	[]byte
	isBusy	bool
	conn 		*websocket.Conn 
	send chan []byte
}

type MatchingServer struct {
	clients	map[*Client]*websocket.Conn 
}

func newMatchingServer() *MatchingServer {
	ms := &MatchingServer{
		clients: make(map[*Client]bool),
	}

	// Wait for 30 seconds
	duration := 30 * time.Second
	time.Sleep(duration)

	// Keep creating matches indefinitely
	for {
		go createMatches()
	}
}


// This finds two random clients which are !isBusy
// It sends both of them each others clientIDs
// Deterministic function: always ends by running sendCandidates
// Can be run as a go routine 
func (*MatchingServer) createMatches() {
	// Filter clients who are not busy
	var availableClients []Client
	for _, c := range ms.clients {
		if !c.IsBusy {
			// Add client to availableClients array
			availableClients = append(availableClients, c)
		}
	}
	// Randomly select two clients from the available clients
	rand.Seed(time.Now().UnixNano())
	index1 := rand.Intn(len(availableClients))
	client1 = &availableClients[index1]
	client1.isBusy = true

	index2 := rand.Intn(len(availableClients) - 1)
	if index2 == index1 {
		index2++
	}
	client2 = &availableClients[index2]
	client2.isBusy = true

	sendCandidates(client1, client2)
}

func sendCandidates(c1 *Client, c2 *Client) {
	client1.send <- client2.clientID
	client2.send <- client1.clientID
}

func (*MatchingServer) addUser(userID []byte, conn *websocket.Conn) {
	clients[userID] = conn 
	return 
}


func serveWs(ms *MatchingServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	// Extract the user ID from the URL
	userID := r.URL.Query().Get("userid")
	// Create a client object -> a representation of the user to the server
	newClient := &Client{clientID: userID, isBusy: true, conn: conn, send: new chan([]byte, 256)}
	ms.addUser(newClient, conn)

	// Wait for 30 seconds
	duration := 30 * time.Second
	time.Sleep(duration)

	newClient.isBusy = false

	go newClient.readPump()
	log.Println("***readPump Created")
	go newClient.writePump()
	log.Println("***writePump Created")
}

func (c *Client) readPump() {
	defer func() {
		c.conn.Close()
	}()
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

// writePump pumps messages that have been sent to the client.send channel
// The application runs writePump in a per-connection goroutine. The application
// ensures that there is at most one writer on a connection by executing all 
// writes from this goroutine. 
func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message := <-c.send:
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
		}
	}
}

func main() {
	newMatchingServer()
	http.HandleFunc("/", httpHandler)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request){
		serveWs(ms, w, r)
	})
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	log.Println("***Websocket server online")
}