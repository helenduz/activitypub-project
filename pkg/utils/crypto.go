package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
)

func GetEncodedKeys() (privKey string, pubKey string) {
    privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
    if err != nil {
        panic(err)
    }

    privateKeyPEM := pem.EncodeToMemory(
        &pem.Block{
            Type:  "RSA PRIVATE KEY",
            Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
        },
    )

    publicKey := &privateKey.PublicKey
    publicKeyDER, err := x509.MarshalPKIXPublicKey(publicKey)
    if err != nil {
        panic(err)
    }

    publicKeyPEM := pem.EncodeToMemory(
        &pem.Block{
            Type:  "PUBLIC KEY",
            Bytes: publicKeyDER,
        },
    )

	return string(privateKeyPEM), string(publicKeyPEM)
}