package crypto

import (
	crypto_rand "crypto/rand"
	"encoding/base64"
	"errors"
	"log"

	"golang.org/x/crypto/nacl/box"
)

func CreateKeys() (pubBase64 string, privBase64 string) {
	pub, priv, err := box.GenerateKey(crypto_rand.Reader)
	if err != nil {
		log.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(pub[:]), base64.StdEncoding.EncodeToString(priv[:])
}

func LoadAndVerivyKeys(pubBase64 string, privBase64 string) *Key {
	pubBytes, err := base64.StdEncoding.DecodeString(pubBase64)
	if err != nil {
		log.Fatal(err)
	}
	privBytes, err := base64.StdEncoding.DecodeString(privBase64)
	if err != nil {
		log.Fatal(err)
	}

	if len(pubBytes) != 32 {
		log.Fatal("base64 decoded public key is not 32 bytes long")
	}
	if len(privBytes) != 32 {
		log.Fatal("base64 decoded private key is not 32 bytes long")
	}

	k := &Key{
		pub:          new([32]byte),
		PublicBase64: pubBase64,
		priv:         new([32]byte),
	}

	copy(k.pub[:], pubBytes)
	copy(k.priv[:], privBytes)

	encryptedTest, err := box.SealAnonymous(nil, []byte("test"), k.pub, crypto_rand.Reader)
	if err != nil {
		log.Fatal(err)
	}

	decryptedTest, err := k.rawDecrypt(encryptedTest)
	if err != nil {
		log.Fatal("decryption failed, private and/or public key are probably incorrect")
	}

	if string(decryptedTest) != "test" {
		log.Fatal("decrypted test message does not match the original message")
	}

	return k
}

type Key struct {
	pub          *[32]byte
	PublicBase64 string
	priv         *[32]byte
}

func (k *Key) rawDecrypt(val []byte) ([]byte, error) {
	decryptedValue, ok := box.OpenAnonymous(nil, val, k.pub, k.priv)
	if !ok {
		return nil, errors.New("decryption failed, either private/public/message are incorrect")
	}
	return decryptedValue, nil
}

func (k *Key) DecryptScraperPassword(encryptedPassword string) (string, error) {
	encryptedPasswordBytes, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", nil
	}

	passwordBytes, err := k.rawDecrypt(encryptedPasswordBytes)
	if err != nil {
		return "", nil
	}

	// Note that the first 32 bytes of a encrypted bytes are just junk to make the encrypted value harder to guess "hopefully"
	return string(passwordBytes[32:]), nil
}
