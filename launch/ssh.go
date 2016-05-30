package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"

	"github.com/relab/goxos/Godeps/_workspace/src/github.com/codegangsta/cli"

	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/go.crypto/ssh"
	"github.com/relab/goxos/Godeps/_workspace/src/code.google.com/p/go.crypto/ssh/agent"
)

var flags = []cli.Flag{
	cli.StringFlag{
		Name:  "user, u",
		Value: currentUser().Username,
		Usage: "User name for ssh connection (default is the local user)",
	},
	cli.StringFlag{
		Name:  "keypath, k",
		Value: filepath.Join(currentUser().HomeDir, ".ssh", "id_rsa"),
		Usage: "Path to private ssh key",
	},
}

// Wrapper for user.Current() to avoid handling error.
func currentUser() *user.User {
	usr, err := user.Current()
	if err != nil {
		panic("Unable to get current user:" + err.Error())
	}
	return usr
}

func sshAuthSock(userName string) (*ssh.ClientConfig, error) {
	conn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK"))
	if err != nil {
		return nil, err
	}
	// defer conn.Close()

	ag := agent.NewClient(conn)
	signers, err := ag.Signers()
	if err != nil {
		return nil, err
	}
	if len(signers) < 1 {
		return nil, errors.New("No signers registered with ssh-agent")
	}
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(ag.Signers),
		},
	}
	return config, nil
}

func sshConfig(userName, privKeyPath string) (*ssh.ClientConfig, error) {
	privateBytes, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return nil, err
	}
	priv, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, err
	}

	// To authenticate with the remote server you must pass at least one
	// implementation of AuthMethod via the Auth field in ClientConfig.
	config := &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(priv),
		},
	}
	return config, nil
}

// Run the provided cmd on host using the given client config.
func sshRun(config *ssh.ClientConfig, host, cmd string) error {
	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return err
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	var s, e bytes.Buffer
	session.Stdout = &s
	session.Stderr = &e
	err = session.Run(cmd)
	fmt.Print(s.String())
	fmt.Print(e.String())
	return err
}
