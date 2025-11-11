package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func generateKeys(outDir string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	privateKeyName := filepath.Join(outDir, "private.pem")
	if err := os.WriteFile(privateKeyName, privateKeyPEM, 0600); err != nil {
		return err
	}

	publicKeyName := filepath.Join(outDir, "public.pem")
	if err := os.WriteFile(publicKeyName, publicKeyPEM, 0644); err != nil {
		return err
	}

	return nil
}

func main() {
	execPath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	execDir := filepath.Dir(execPath)

	var outDir string
	flag.StringVar(&outDir, "out-dir", execDir, "")
	flag.Parse()

	if err := generateKeys(outDir); err != nil {
		log.Fatal(err)
	}

	fmt.Printf(
		"Private key generated: %s\n",
		filepath.Join(outDir, "private.pem"),
	)
	fmt.Printf("Public key generated: %s\n", filepath.Join(outDir, "public.pem"))
}
