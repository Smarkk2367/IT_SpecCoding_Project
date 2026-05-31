package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("ERR: %v\n", err)
		return
	}
	fmt.Printf("HASH: %s\n", string(hash))
}
