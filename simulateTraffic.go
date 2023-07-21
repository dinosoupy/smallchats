package main

import (
	"net/http"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"time"

	"github.com/gorilla/websocket"
)

const (
	baseURL   = "localhost:8000"
	joinRoute = "/join"
)

type User struct {
	UserID    string   `json:"userID"`
	Name      string   `json:"name"`
	Interests []string `json:"interests"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Connect to the WebSocket endpoint
		conn, err := websocketConnect()
		if err != nil {
			log.Println(err)
		}

		// Generate random user details
		user := generateRandomUser()

		// Send the join request over WebSocket
		err = sendJoinRequest(conn, user)
		if err != nil {
			log.Println(err)
		}

		// Wait for the server response
		_, _, err = conn.ReadMessage()
		if err != nil {
			log.Println(err)
		}
}

func generateRandomUser() User {
	// Generate random user details
	userID := fmt.Sprintf("user%d", rand.Intn(1000))
	name := fmt.Sprintf("User %d", rand.Intn(1000))
	interests := []string{"music", "sports", "movies"}

	return User{
		UserID:    userID,
		Name:      name,
		Interests: interests,
	}
}

func websocketConnect() (*websocket.Conn, error) {
	// Prepare the WebSocket URL
	url := "ws://" + baseURL + joinRoute

	// Create custom headers for upgrade and connection
	headers := make(http.Header)

	// Establish a WebSocket connection with custom headers
	conn, _, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func sendJoinRequest(conn *websocket.Conn, user User) error {
	// Encode the user object to JSON
	message, err := json.Marshal(user)
	if err != nil {
		return err
	}

	// Send the join request over WebSocket
	err = conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		return err
	}

	fmt.Println("Submitted random user: ", user.UserID)
	return nil
}
