package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/kevinburke/nacl/box"
)

func main() {
	lockboxPubKey, lockboxPriKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	pub64 := base64.StdEncoding.EncodeToString(lockboxPubKey[:])
	pri64 := base64.StdEncoding.EncodeToString(lockboxPriKey[:])

	fmt.Fprintf(os.Stdout, "public:  %s\nprivate: %s\n", pub64, pri64)
}
