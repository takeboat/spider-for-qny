package spider

import (
	"bytes"
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

func InitSSH(host string) (client *ssh.Client, err error) {
	passwords := []string{"123456", "ft0323"}
	for _, password := range passwords {
		config := &ssh.ClientConfig{
			User: "root",
			Auth: []ssh.AuthMethod{
				ssh.Password(password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		}
		client, err = ssh.Dial("tcp", host, config)
		if err == nil {
			return
		}
	}
	return nil, fmt.Errorf("failed to connect to %s: 所有密码都尝试失败 err: %v", host, err)
}

func RunSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	if err != nil {
		return "", fmt.Errorf("failed to run command: %v", err)
	}
	return stdout.String(), nil
}
