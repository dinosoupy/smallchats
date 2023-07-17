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

	"github.com/gorilla/websocket"
)

type User struct {
	UserID    string   `json:"userID"`
	Name      string   `json:"name"`
	Interests []string `json:"interests"`
	Conn      *websocket.Conn
}

type MatchingServer struct {
	mutex sync.Mutex
	users []*User
}

func NewMatchingServer() *MatchingServer {
	return &MatchingServer{
		users: make([]*User, 0),
	}
}

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

func sendCandidate(conn *websocket.Conn, candidate *User) error {
	// Marshal the candidate object to JSON
	candidateJSON, err := json.Marshal(candidate)
	if err != nil {
		return err
	}

	// Send the candidate JSON over the WebSocket connection
	err = conn.WriteMessage(websocket.TextMessage, candidateJSON)
	if err != nil {
		return err
	}

	return nil
}

func (s *MatchingServer) MatchUsers() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.users) < 2 {
		return
	}

	// Select two users at random
	indexA := rand.Intn(len(s.users))
	userA := s.users[indexA]
	// s.users = append(s.users[:indexA], s.users[indexA+1:]...)

	indexB := rand.Intn(len(s.users))
	userB := s.users[indexB]

	// Send candidate to userA
	candidateA := userB
	sendCandidate(userA.Conn, candidateA)

	// Send candidate to userB
	candidateB := userA
	sendCandidate(userB.Conn, candidateB)

	fmt.Printf("Matched userA: %s, userB: %s\n", userA.Name, userB.Name)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Received a request at /join")

	// Check if the request contains the necessary headers for WebSocket upgrade
	if r.Header.Get("Upgrade") != "websocket" || r.Header.Get("Connection") != "Upgrade" {
		http.Error(w, "Upgrade to WebSocket required", http.StatusBadRequest)
		return
	}

	fmt.Println("Upgrading connection to WebSocket")

	// Upgrade the connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	fmt.Println("Connection upgraded to WebSocket")

	// Send confirmation response to the client
	err = conn.WriteMessage(websocket.TextMessage, []byte("Connection upgraded to WebSocket"))
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Println("Confirmation message sent to the client")

	// // Create a new user with the WebSocket connection
	// var user = User{
	// 	Conn: conn,
	// }

	// // Add the user to the matching server
	// matchServer.AddUser(&user)

	// // Match users
	// matchServer.MatchUsers()

	// fmt.Printf("%s joined. Active users: %d\n", user.UserID, len(matchServer.users))

	// // Handle WebSocket messages
	// for {
	// 	// Read the message from the WebSocket connection
	// 	_, message, err := conn.ReadMessage()
	// 	if err != nil {
	// 		log.Println(err)
	// 		break
	// 	}
	// }
}


var (
	upgrader    = websocket.Upgrader{}
	matchServer = NewMatchingServer()
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
