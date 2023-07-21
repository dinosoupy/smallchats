package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

type User struct {
	UserID    string   `json:"userID"`
	Name      string   `json:"name"`
	Interests []string `json:"interests"`
}

func joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse form data
		userID := r.FormValue("userID")
		name := r.FormValue("name")
		interestsStr := r.FormValue("interests")

		// Split interests by comma and trim spaces
		interests := strings.Split(interestsStr, ",")
		for i := range interests {
			interests[i] = strings.TrimSpace(interests[i])
		}

		// Create User object
		user := User{
			UserID:    userID,
			Name:      name,
			Interests: interests,
		}

		// Convert User to JSON
		userJSON, err := json.Marshal(user)
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Send JSON to localhost server at port 8000
		resp, err := http.Post("http://localhost:8000/join", "application/json", bytes.NewBuffer(userJSON))
		if err != nil {
			log.Println(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Handle the response if needed

		fmt.Fprint(w, "Form submitted successfully!")
		return
	}

	// Render the HTML form
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Join Form</title>
		<style>
			video {
				width: 50%;
				height: auto;
			}
		</style>
	</head>
	<body>
		<video id="videoPreview" autoplay></video>
		<br>
		<form method="POST" action="/join">
			<label for="userID">User ID:</label>
			<input type="text" id="userID" name="userID" required><br><br>

			<label for="name">Name:</label>
			<input type="text" id="name" name="name" required><br><br>

			<label for="interests">Interests (comma-separated):</label>
			<input type="text" id="interests" name="interests" required><br><br>

			<input type="submit" value="Join">
		</form>

		<script>
			navigator.mediaDevices.getUserMedia({ video: true })
				.then(function(stream) {
					var videoPreview = document.getElementById('videoPreview');
					videoPreview.srcObject = stream;
				})
				.catch(function(error) {
					console.log('Error accessing webcam:', error);
				});
		</script>
	</body>
	</html>
	`

	fmt.Fprint(w, tmpl)
}

func main() {
	http.HandleFunc("/join", joinHandler)
	fmt.Printf("Starting server at port 8080\n")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
