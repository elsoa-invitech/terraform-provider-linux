package linux

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type Config struct {
	Host       string
	Port       int
	User       string
	Password   string
	PrivateKey string
	UseSudo    bool
}

type Client struct {
	connection *ssh.Client
	useSudo    bool
}

func (c *Config) Client() (*Client, error) {
	var auths []ssh.AuthMethod

	if c.Password != "" {
		auths = append(auths, ssh.Password(c.Password))
	} else {
		keys := []ssh.Signer{}
		key, err := ioutil.ReadFile(c.PrivateKey)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				keys = append(keys, signer)
			}
		} else if !os.IsNotExist(err) || !strings.HasSuffix(c.PrivateKey, "/.ssh/id_rsa") {
			return nil, err
		}

		if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			signers, err := agent.NewClient(sshAgent).Signers()
			if err == nil {
				keys = append(keys, signers...)
			}
		}

		auths = append(auths, ssh.PublicKeys(keys...))
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.User,
		Auth:            auths,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial: %s", err)
	}

	log.Printf("SSH client configured")

	return &Client{
		connection: connection,
		useSudo:    c.UseSudo,
	}, nil
}
