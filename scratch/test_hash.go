package main

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
	passwords := []string{
		"password", "admin", "correct horse battery staple", "correct horse battery staple\x00",
		"admin123", "trackflow", "marketer",
	}
	for _, p := range passwords {
		err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p))
		if err == nil {
			fmt.Printf("MATCH: %s\n", p)
			return
		} else {
			fmt.Printf("For '%s': %v\n", p, err)
		}
	}
}
