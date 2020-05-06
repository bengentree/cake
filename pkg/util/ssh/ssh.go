package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"strings"

	"golang.org/x/crypto/ssh"
)

// GenerateRSAKeyPair as strings private, public
func GenerateRSAKeyPair() (string, string, error) {
	var privateKey, publicKey string
	reader := rand.Reader
	bitSize := 4096

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return privateKey, publicKey, err
	}
	privKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	buf := pem.EncodeToMemory(privKey)
	privateKey = strings.TrimSpace(string(buf))

	buf2, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return privateKey, publicKey, err
	}
	publicKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(buf2)))

	return privateKey, publicKey, nil
}
