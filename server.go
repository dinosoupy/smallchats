package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type User struct {
	UserID    string   `json:"userID"`
	Name      string   `json:"name"`
	Interests []string `json:"interests"`
	busy			bool
	Conn      *websocket.Conn
}

type MatchingServer struct {
	mutex sync.Mutex
	users []*User
}

func NewMatchingServer() *MatchingServer {

	// Create MatchingServer instance
	server := &MatchingServer{
		users: make([]*User, 0),
	}

	// Create two random users
	for i := 0; i < 5; i++ {
		userID := fmt.Sprintf("user%d", i)
		name := fmt.Sprintf("User %d", i)
		interests := []string{"music", "sports", "movies"}

		// Create user object
		user := &User{
			UserID:    userID,
			Name:      name,
			Interests: interests,
			busy:      false,
			Conn:      nil,
		}

		// Add the user to the server
		server.AddUser(user)
	}

	return server
}

// Appends user object to array of users
// TODO: joinedAt time stamp to user struct
func (s *MatchingServer) AddUser(user *User) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.users = append(s.users, user)
}

func (s *MatchingServer) RemoveUser(user *User) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for i, u := range s.users {
		if u == user {
			s.users = append(s.users[:i], s.users[i+1:]...)
			break
		}
	}
}

// Returns a pair of active users who have been matched
// Sets both selected users to be "busy"
// TODO: 
// 1. check if users are busy before selecting
func (s *MatchingServer) MatchUsers() (*User, *User) {
	// Select two users at random
	indexA := rand.Intn(len(s.users))
	userA := s.users[indexA]

	indexB := rand.Intn(len(s.users))
	userB := s.users[indexB]

	userA.busy = true 
	userB.busy = true

	fmt.Printf("Matched userA: %s, userB: %s\n", userA.Name, userB.Name)
	return userA, userB
}

// Takes an existing HTTP connection and returns a websocket.Conn connection object
func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// Continuously listen for messages on the websocket connection
// This function is to be run concurrently 
// Add messageType switch cases to handle each type of messages received 
func listenForMessages(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			return
		}

		// Process the message
		var parsedMessage struct {
			MessageType string `json:"messageType"`
			Content     string `json:"content"`
			// Include other fields as needed
		}

		err = json.Unmarshal(message, &parsedMessage)
		if err != nil {
			log.Println(err)
			continue
		}

		// Parse incoming messages according to their messageType
		switch parsedMessage.MessageType {
		case "receivedUserData":
			// create a user object and add conn to its Conn field
			// Add user to active users list by calling AddUser(user) function
			// start timer for 5 seconds
			// run matchUsers and store the paired up users in two variables userA, userB
			// select userA as host
			// run requestSDP(userA) function and wait for response with messageType offerSDP
			var user User
			user.Conn = conn
			err = json.Unmarshal([]byte(parsedMessage.Content), &user)
			if err != nil {
				log.Println(err)
				continue
			}

			s.AddUser(&user)

			time.Sleep(5 * time.Second)
			a, b := s.MatchUsers()

			host := a
			requestOfferSDP(host)

		case "receivedOfferSDP":
			// create sdp object offerSDP
			// read json into the sdp object 
			// run forwardSDP(userB, offerSDP) which sends offer sdp to userB, then wait for response with messageType answerSDP
			// Parse the received offer SDP
			var offerSDP webrtc.SessionDescription
			err := json.Unmarshal([]byte(parsedMessage.Content), &offerSDP)
			if err != nil {
				log.Println(err)
				return
			}

			// Forward the offer SDP to userB
			err = forwardSDP(offerSDP, b)
			if err != nil {
				log.Println(err)
				return
			}

		case "receivedAnswerSDP":
			// create sdp object answerSDP
			// read json data into sdp object
			// run forward(userA, answerSDP) which sends answer sdp to userA
			var answerSDP webrtc.SessionDescription
			err := json.Unmarshal([]byte(parsedMessage.Content), &answerSDP)
			if err != nil {
				log.Println(err)
				return
			}

			// Forward the offer SDP to userB
			err = forwardSDP(answerSDP, a)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func forwardSDP(offerSDP webrtc.SessionDescription, user *User) error {
	// Create a message containing the offer SDP
	message := struct {
		MessageType string                 `json:"messageType"`
		Content     webrtc.SessionDescription `json:"content"`
	}{
		MessageType: "forwardSDP",
		Content:     offerSDP,
	}

	// Encode the message as JSON
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// Send the message to userB's connection
	err = user.Conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		return err
	}

	return nil
}


// Given a user object, it sends a message through its websocket
// connection. The message is of the type requestSDP to indicate to the 
// client that an offer sdp is requested
func requestOfferSDP(user *User) {
	// Create a request message
	message := struct {
		MessageType string `json:"messageType"`
	}{
		MessageType: "requestOfferSDP",
	}

	// Convert the message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		log.Println(err)
		return
	}

	// Send the JSON message to the user
	err = user.Conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Request SDP message sent to user:", user.UserID)
}

// Expose /join endpoint to receive client join requests
// Automatically upgrades http connection to websocket 
// if appropriate upgrade header is found
func joinHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received a request at /join")

	// Check if the request contains the necessary headers for WebSocket upgrade
	if r.Header.Get("Upgrade") != "websocket" || r.Header.Get("Connection") != "Upgrade" {
		http.Error(w, "Upgrade to WebSocket required", http.StatusBadRequest)
		return
	}

	fmt.Println("Upgrading connection to WebSocket")

	// Upgrade the connection to WebSocket
	conn, err := upgradeToWebSocket(w, r)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	fmt.Println("Connection upgraded to WebSocket")

	// Start a goroutine to listen for messages on the WebSocket connection
	go listenForMessages(conn)

	fmt.Println("Waiting for user messages...")
}


var (
	upgrader = websocket.Upgrader{}
	s = NewMatchingServer()
)

func main() {
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM)

	go func() {
		// Start WebSocket server
		http.HandleFunc("/join", joinHandler)
		fmt.Println("Matching server is online at port 8000... (waiting for users to join)")
		log.Fatal(http.ListenAndServe(":8000", nil))
	}()

	// Wait for a termination signal
	<-terminate
	fmt.Println("\nShutting down the server...")

	fmt.Println("Server gracefully stopped.")
}
