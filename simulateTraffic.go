package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

const (
	baseURL   = "http://localhost:8000"
	joinRoute = "/join"
)

type User struct {
	UserID    string   `json:"userID"`
	Name      string   `json:"name"`
	Interests []string `json:"interests"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	// Number of simulated users to create
	numUsers := 10

	// Generate and send traffic
	for i := 0; i < numUsers; i++ {
		// Generate random user details
		user := generateRandomUser()

		// Send join request with user details
		err := sendJoinRequest(user)
		if err != nil {
			log.Println(err)
		}

		fmt.Println("Submitted random user: ", user.UserID)
		time.Sleep(1 * time.Second) // Sleep for some time between requests
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

func sendJoinRequest(user User) error {
	// Prepare the request body
	body, err := json.Marshal(user)
	if err != nil {
		return err
	}

	// Send the POST request to the /join endpoint
	url := baseURL + joinRoute
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	return nil
}
