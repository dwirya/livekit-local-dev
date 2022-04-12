//go:build mage

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/livekit/protocol/auth"
	"github.com/magefile/mage/sh"
	"gopkg.in/yaml.v3"
)

func Init() error {
	return sh.Run("docker", "run --rm -v $PWD:/output livekit/generate --local")
}

func Livekit() error {
	// Get node IP
	nodeIP, err := getLinuxNodeIP()
	if err != nil {
		return err
	}

	// Get volume mapping for copying the config file
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	volumeMap := fmt.Sprintf("%s/livekit.yaml:/livekit.yaml", wd)

	return sh.Run(
		"docker",
		"run", "--rm",
		"-p", "7880:7880",
		"-p", "7881:7881",
		"-p", "7882:7882/udp",
		"-v", volumeMap,
		"livekit/livekit-server",
		"--config", "/livekit.yaml",
		"--node-ip", nodeIP,
	)
}

func getLinuxNodeIP() (string, error) {
	// Linux specific command
	hostname, err := sh.Output("hostname", "-I")
	if err != nil {
		return "", err
	}

	// Output should be: <IP in network, starts with 192.168...> , <Local IP, starts with 172...>
	hostIPs := strings.Split(hostname, " ")
	nodeIP := hostIPs[0]
	return nodeIP, nil
}

type LiveKitConfig struct {
	// We're only interested in the `keys` field
	Keys map[string]string `yaml:"keys"`
}

func Token(room string, identity string) error {
	kp, err := getKeyPairFromFile("livekit.yaml")
	if err != nil {
		return err
	}

	if kp == nil || len(kp) < 1 {
		return errors.New("no key-value pairs seen")
	}

	// Convert to array for easier manipulation
	var keys []string
	var secrets []string
	for k, s := range kp {
		keys = append(keys, k)
		secrets = append(secrets, s)
	}

	// Use the first one seen
	apiKey, apiSecret := keys[0], secrets[0]

	// Generate token
	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomAdmin: true,
		RoomJoin:  true,
		Room:      room,
	}
	token, err := at.AddGrant(grant).SetIdentity(identity).SetName(identity).ToJWT()
	if err != nil {
		return err
	}

	// Print token
	print("Token: ", token, "\n")
	return nil
}

func getKeyPairFromFile(filename string) (map[string]string, error) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	c := LiveKitConfig{}
	err = yaml.Unmarshal(file, &c)
	return c.Keys, err
}
