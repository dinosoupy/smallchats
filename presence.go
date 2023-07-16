package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/supabase/supabase-go"
)

func main() {
	// Initialize Supabase client
	supabaseURL := "https://dkkrcwfebmkutqtdncgb.supabase.co"
	supabaseKey := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImRra3Jjd2ZlYm1rdXRxdGRuY2diIiwicm9sZSI6ImFub24iLCJpYXQiOjE2ODk0MDkxNTIsImV4cCI6MjAwNDk4NTE1Mn0.rgO36VvXbFcqHoUjBApsrFflF6MR-uXiS7IWP0FIFH8"
	client, err := supabase.NewClient(supabaseURL, supabaseKey)
	if err != nil {
		log.Fatal(err)
	}

	// Create a channel to receive presence updates
	presenceUpdates := make(chan supabase.PresenceEvent)

	// Create a context for cancelling the subscription
	ctx, cancel := context.WithCancel(context.Background())

	// Subscribe to presence updates
	subscription, err := client.Realtime.Presence().Subscribe(ctx, presenceUpdates)
	if err != nil {
		log.Fatal(err)
	}

	// Create a queue to store logged-in users
	loggedInUsers := make([]string, 0)

	// Start a goroutine to process presence updates
	go func() {
		for event := range presenceUpdates {
			// Handle presence updates
			if event.Action == "join" {
				loggedInUsers = append(loggedInUsers, event.User.ID)
				fmt.Printf("%s joined. Total logged-in users: %d\n", event.User.ID, len(loggedInUsers))
			} else if event.Action == "leave" {
				// Remove the user from the queue
				for i, user := range loggedInUsers {
					if user == event.User.ID {
						loggedInUsers = append(loggedInUsers[:i], loggedInUsers[i+1:]...)
						break
					}
				}
				fmt.Printf("%s left. Total logged-in users: %d\n", event.User.ID, len(loggedInUsers))
			}
		}
	}()

	// Wait for termination signal to stop the subscription
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	// Cancel the subscription and clean up
	cancel()
	if err := subscription.Close(); err != nil {
		log.Fatal(err)
	}
}

