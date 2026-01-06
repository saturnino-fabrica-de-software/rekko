package main

import (
	"fmt"
	"os"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

func main() {
	keyType := domain.KeyTypeSecret
	env := domain.EnvLive

	if len(os.Args) > 1 && os.Args[1] == "pk" {
		keyType = domain.KeyTypePublic
	}
	if len(os.Args) > 2 && os.Args[2] == "test" {
		env = domain.EnvTest
	}

	key, hash, prefix, err := domain.GenerateAPIKey(keyType, env)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("KEY=%s\nHASH=%s\nPREFIX=%s\n", key, hash, prefix)
}
