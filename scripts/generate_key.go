//go:build ignore

package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/freeeakn/AetherWave/pkg/crypto"
)

func main() {
	key, err := crypto.GenerateKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(hex.EncodeToString(key))
}
