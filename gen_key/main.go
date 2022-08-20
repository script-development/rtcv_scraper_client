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
	fmt.Println("Add them here your scrapers `env.json` file using")
	envAddition, _ := json.MarshalIndent(map[string]any{
		"primary_server": map[string]string{
			"server_location": "..",
			"api_key_id":      "..",
			"api_key":         "..",
			"private_key":     priv,
			"public_key":      pub,
		},
	}, "", "  ")
	fmt.Println(string(envAddition))

	fmt.Println()
	fmt.Println("Don't forget to add the Public Key to the scraper api key on the RT-CV dashboard")
}
