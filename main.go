package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

type AssetEvent struct {
	PlayerID string `json:"player_id"`
	AssetID  string `json:"asset_id"`
	Kind     string `json:"kind"`
}

func main() {
	client, err := NewQueueClient()
	if err != nil {
		log.Fatal(err)
	}
	event := AssetEvent{PlayerID: "player-42", AssetID: "asset-9001", Kind: "character-skin"}
	key := os.Getenv("WEBHOOK_EVENT_KEY")
	if key == "" {
		key = event.AssetID
	}
	published, err := client.Publish(event, key)
	if err != nil {
		log.Fatal(err)
	}
	batch, err := client.Consume(10, 30)
	if err != nil {
		log.Fatal(err)
	}
	for _, message := range batch.Messages {
		var incoming AssetEvent
		if err := json.Unmarshal(message.Payload, &incoming); err != nil {
			log.Printf("leave %s visible for retry: %v", message.MessageID, err)
			continue
		}
		fmt.Printf("moderate asset=%s player=%s message=%s\n", incoming.AssetID, incoming.PlayerID, message.MessageID)
		if err := client.Ack(message.MessageID); err != nil {
			log.Printf("ack %s after moderation: %v", message.MessageID, err)
		}
	}
	fmt.Println("published message:", published.MessageID)
}
