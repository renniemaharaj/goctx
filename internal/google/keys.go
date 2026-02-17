package google

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type GoogleApiKey struct {
	model string
	key   string
}

var GoogleApiKeysChannel chan GoogleApiKey

func BorrowOneGoogleApiKey() GoogleApiKey {
	return <-GoogleApiKeysChannel
}

func FreeMultipleGoogleApiKey(keys []GoogleApiKey) {
	for _, key := range keys {
		FreeOneGoogleApiKey(key)
	}
}

func FreeOneGoogleApiKey(key GoogleApiKey) {
	go func() {
		time.Sleep(time.Second * 2)
		GoogleApiKeysChannel <- key
	}()
}

func loadKeysFromEnv() error {
	keysString := os.Getenv("GOOGLE_GEMINI_API_KEYS_FOR_GO")
	keysStructs := &[]GoogleApiKey{}
	if keysString != "" {
		if err := json.Unmarshal([]byte(keysString), &keysStructs); err != nil {
			return fmt.Errorf("fatal GOOGLE_GEMINI_API_KEYS_FOR_GO could not me unmarshaled")
		}

		for _, key := range *keysStructs {
			GoogleApiKeysChannel <- key
		}

		return nil
	}
	return nil
}
