package main

import (
	"context"
	"log"

	ipace "github.com/dgem/ipace-owners/functions/firebase-go"
)

func main() {
	if err := ipace.RegeneratePublicStatsSnapshot(context.Background()); err != nil {
		log.Fatalf("regenerate public statistics snapshot: %v", err)
	}
	log.Print("regenerated public statistics snapshot")
}
