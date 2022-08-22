package main

import (
	"encoding/json"
	"fmt"

	"github.com/script-development/rtcv_scraper_client/v2/crypto"
)

func main() {
	pub, priv := crypto.CreateKeys()

	fmt.Println("Generated keys:")
	fmt.Println("don't lose these keys!!")

	fmt.Println()
	fmt.Println("Public key:")
	fmt.Println(pub)
	fmt.Println("Private key:")
	fmt.Println(priv)

	fmt.Println()
	fmt.Println("Add them like so to your scraper's `env.json`")
	envAddition, _ := json.MarshalIndent(map[string]any{
		"private_key":    priv,
		"public_key":     pub,
		"primary_server": map[string]string{},
	}, "", "  ")
	fmt.Println(string(envAddition))

	fmt.Println()
	fmt.Println("Don't forget to add the Public Key to the scraper api key on the RT-CV dashboard")
}
