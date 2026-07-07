package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"golang.org/x/crypto/ssh"
)

type Node struct { Name, Host string; Port int; User string }
type Config struct { PrivateKeyPath string; Nodes []Node }

func runRemoteCommand(node Node, signer ssh.Signer, cmd string, wg *sync.WaitGroup) {
	defer wg.Done()
	config := &ssh.ClientConfig{User: node.User, Auth: []ssh.AuthMethod{ssh.PublicKeys(signer)}, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	client, _ := ssh.Dial("tcp", fmt.Sprintf("%s:%d", node.Host, node.Port), config)
	defer client.Close()
	session, _ := client.NewSession()
	defer session.Close()
	out, _ := session.CombinedOutput(cmd)
	fmt.Printf("[%s]: %s\n", node.Name, string(out))
}

func main() {
	configData, _ := ioutil.ReadFile("config.json")
	var config Config
	json.Unmarshal(configData, &config)
	key, _ := ioutil.ReadFile(config.PrivateKeyPath)
	signer, _ := ssh.ParsePrivateKey(key)
	var wg sync.WaitGroup
	for _, node := range config.Nodes {
		wg.Add(1)
		go runRemoteCommand(node, signer, os.Args[1], &wg)
	}
	wg.Wait()
}
