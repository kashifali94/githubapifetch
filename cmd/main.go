package main

import (
	"githubassign/service"
	"log"
)

func main() {
	ser, err := service.NewService()
	if err != nil {
		log.Fatalf("Failed to initialize ser: %v", err)
	}
	defer func() {
		if err := ser.Close(); err != nil {
			log.Printf("Error during ser shutdown: %v", err)
		}
	}()

	if err := ser.Start(); err != nil {
		log.Fatalf("Service error: %v", err)
	}
}
