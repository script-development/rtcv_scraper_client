package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"muzzammil.xyz/jsonc"
)

var customEnvFileVarName = "RTCV_SCRAPER_CLIENT_ENV_FILE"
var envAsVarName = "RTCV_SCRAPER_CLIENT_ENV"

func mustReadEnv() Env {
	envFilename := "env.json"
	alternativeFileName := os.Getenv(customEnvFileVarName)
	if alternativeFileName != "" {
		envFilename = alternativeFileName
	}

	notFoundErr := func() string {
		if alternativeFileName != "" {
			return "no " + alternativeFileName + " file or $" + envAsVarName + " environment variable found, cannot continue"
		}
		return "no env.json(c) file or $" + envAsVarName + " environment variable found, cannot continue"
	}

	envFileBytes, err := ioutil.ReadFile(envFilename)
	if err == nil {
		return mustParseEnv(envFileBytes)
	}

	envFileBytes = []byte(os.Getenv(envAsVarName))
	if len(envFileBytes) != 0 {
		return mustParseEnv(envFileBytes)
	}

	if !os.IsNotExist(err) {
		log.Fatal("unable to read env file, error: " + err.Error())
	}

	if alternativeFileName != "" {
		log.Fatalln(notFoundErr())
	}

	envFilename = "env.jsonc"
	envFileBytes, err = ioutil.ReadFile(envFilename)
	if err != nil {
		log.Fatalln(notFoundErr())
	}

	return mustParseEnv(envFileBytes)
}

func mustParseEnv(b []byte) Env {
	envJSON := jsonc.ToJSON(b)
	if !json.Valid(envJSON) {
		log.Fatal("env file is not valid json or jsonc")
	}

	env := Env{}
	err := json.Unmarshal(envJSON, &env)
	if err != nil {
		log.Fatal("unable to parse env file, error: " + err.Error())
	}

	err = env.validate()
	if err != nil {
		log.Fatal("validating env failed, error: " + err.Error())
	}

	return env
}

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
