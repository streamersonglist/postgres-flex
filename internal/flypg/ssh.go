package flypg

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	privateKeyFile = "/data/.ssh/id_rsa"
	publicKeyFile  = "/data/.ssh/id_rsa-cert.pub"
)

func writeSSHKey() error {
	err := os.Mkdir("/data/.ssh", 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}

	if err := writePrivateKey(); err != nil {
		return fmt.Errorf("failed to write private key: %s", err)
	}

	if err := writePublicKey(); err != nil {
		return fmt.Errorf("failed to write cert: %s", err)
	}

	cmdStr := fmt.Sprintf("chmod 600 %s %s", privateKeyFile, publicKeyFile)
	cmd := exec.Command("sh", "-c", cmdStr)
	if _, err := cmd.Output(); err != nil {
		return err
	}

	return nil
}

func writePrivateKey() error {
	key := os.Getenv("SSH_KEY")

	keyFile, err := os.Create(privateKeyFile)
	if err != nil {
		return err
	}
	defer keyFile.Close()
	_, err = keyFile.Write([]byte(key))
	if err != nil {
		return err
	}

	return keyFile.Sync()
}

func writePublicKey() error {
	cert := os.Getenv("SSH_CERT")

	certFile, err := os.Create(publicKeyFile)
	if err != nil {
		return err
	}
	defer certFile.Close()

	_, err = certFile.Write([]byte(cert))
	if err != nil {
		return err
	}

	return certFile.Sync()
}