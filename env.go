package main

import (
	"errors"
	"fmt"
)

// Env contains the structure of an env.json file
type Env struct {
	PrivateKey         string      `json:"private_key"`
	PublicKey          string      `json:"public_key"`
	PrimaryServer      EnvServer   `json:"primary_server"`
	AlternativeServers []EnvServer `json:"alternative_servers"`
	MockMode           bool        `json:"mock_mode"`
	MockUsers          []EnvUser   `json:"mock_users"`
}

func (e *Env) validate() error {
	if e.MockMode {
		if len(e.MockUsers) == 0 {
			fmt.Println(`"mock_users" is empty in env.json, most scrapers require at least one user to login.`)
			fmt.Println(`For documentation about mocking see https://github.com/script-development/rtcv_scraper_client`)
			if e.MockUsers == nil {
				e.MockUsers = []EnvUser{}
			}
		}
		return nil
	}

	err := e.PrimaryServer.validate()
	if err != nil {
		return fmt.Errorf("primary_server.%s", err.Error())
	}

	for idx, server := range e.AlternativeServers {
		err := server.validate()
		if err != nil {
			return fmt.Errorf("%s[%d].%s", "alternative_servers", idx, err.Error())
		}
	}

	keyPairHelpMsg := `, use the go program inside the "gen_key" folder to generate a key pair`
	if e.PrivateKey == "" && e.PublicKey == "" {
		return errors.New(`"public_key" and "private_key" are required` + keyPairHelpMsg)
	} else if e.PrivateKey == "" {
		return errors.New(`"private_key" required` + keyPairHelpMsg)
	} else if e.PublicKey == "" {
		return errors.New(`"public_key" required` + keyPairHelpMsg)
	}

	return nil
}

// EnvServer contains the structure of the primary_server and alternative_servers inside the .env file
type EnvServer struct {
	ServerLocation string `json:"server_location"`
	APIKeyID       string `json:"api_key_id"`
	APIKey         string `json:"api_key"`
}

func (e *EnvServer) validate() error {
	if e.ServerLocation == "" {
		return errors.New("server_location is required")
	}
	if e.APIKeyID == "" {
		return errors.New("api_key_id is required")
	}
	if e.APIKey == "" {
		return errors.New("api_key is required")
	}

	return nil
}

func (e *EnvServer) toCredArg(isPrimary bool) SetCredentialsArg {
	return SetCredentialsArg{
		ServerLocation: e.ServerLocation,
		APIKeyID:       e.APIKeyID,
		APIKey:         e.APIKey,
		Primary:        isPrimary,
	}
}

// EnvUser contains the structure of the login_users inside the .env file
type EnvUser struct {
	Username          string `json:"username"`
	Password          string `json:"password"`
	EncryptedPassword string `json:"encryptedPassword,omitempty"`
}
