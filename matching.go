package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"

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

func (s *MatchingServer) MatchUsers() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if len(s.users) < 2 {
		return
	}

	// Select two users at random
	indexA := rand.Intn(len(s.users))
	userA := s.users[indexA]
	s.users = append(s.users[:indexA], s.users[indexA+1:]...)

	indexB := rand.Intn(len(s.users))
	userB := s.users[indexB]
	s.users = append(s.users[:indexB], s.users[indexB+1:]...)

	// Initiate P2P connection between the pair
	// You can implement the logic for establishing the P2P connection here

	fmt.Printf("Matched userA: %s, userB: %s\n", userA.Name, userB.Name)
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	// Parse JSON data from request body
	var user User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		log.Println(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Add the user to the matching server
	matchServer.AddUser(&user)

	// Match users
	matchServer.MatchUsers()

	fmt.Println("New user joined. Active users:", len(matchServer.users))
}


var (
	upgrader     = websocket.Upgrader{}
	matchServer  = NewMatchingServer()
)

func main() {
	http.HandleFunc("/join", joinHandler)

	fmt.Printf("Starting server at port 8000\n")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
