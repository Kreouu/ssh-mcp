package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

type proxyConn struct {
	net.Conn
	onClose func() error
}

func (p *proxyConn) Close() error {
	connErr := p.Conn.Close()
	if p.onClose == nil {
		return connErr
	}
	return errors.Join(connErr, p.onClose())
}

func (a *App) dialSSH(host *HostConfig, timeoutSec int) (*ssh.Client, error) {
	return a.dialSSHWithVisited(host, timeoutSec, map[string]bool{})
}

func (a *App) dialSSHWithVisited(host *HostConfig, timeoutSec int, visited map[string]bool) (*ssh.Client, error) {
	if visited[host.Name] {
		return nil, fmt.Errorf("ssh proxy jump cycle detected at %s", host.Name)
	}
	visited[host.Name] = true

	if jumpName := strings.TrimSpace(host.ProxyJump); jumpName != "" {
		jumpHost, err := a.getHost(jumpName)
		if err != nil {
			return nil, fmt.Errorf("resolve proxy jump for %s: %w", host.Name, err)
		}
		jumpClient, err := a.dialSSHWithVisited(jumpHost, timeoutSec, visited)
		if err != nil {
			return nil, err
		}

		addr := fmt.Sprintf("%s:%d", host.Address, host.Port)
		conn, err := jumpClient.Dial("tcp", addr)
		if err != nil {
			_ = jumpClient.Close()
			return nil, fmt.Errorf("dial through proxy %s to %s: %w", jumpName, addr, err)
		}
		if timeoutSec > 0 {
			_ = conn.SetDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))
		}

		cfg, err := buildSSHClientConfig(host, timeoutSec)
		if err != nil {
			_ = conn.Close()
			_ = jumpClient.Close()
			return nil, err
		}

		sshConn, chans, reqs, err := ssh.NewClientConn(&proxyConn{Conn: conn, onClose: jumpClient.Close}, addr, cfg)
		if err != nil {
			_ = conn.Close()
			_ = jumpClient.Close()
			return nil, err
		}
		if timeoutSec > 0 {
			_ = conn.SetDeadline(time.Time{})
		}
		return ssh.NewClient(sshConn, chans, reqs), nil
	}

	cfg, err := buildSSHClientConfig(host, timeoutSec)
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)
	return ssh.Dial("tcp", addr, cfg)
}

func buildSSHClientConfig(host *HostConfig, timeoutSec int) (*ssh.ClientConfig, error) {
	auths, err := buildAuthMethods(host)
	if err != nil {
		return nil, err
	}
	if len(auths) == 0 {
		return nil, fmt.Errorf("no auth methods for host %s", host.Name)
	}

	hostKeyCallback, err := buildHostKeyCallback(host)
	if err != nil {
		return nil, err
	}

	hostKeyAlgorithms, err := resolveHostKeyAlgorithms(host)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:              host.User,
		Auth:              auths,
		HostKeyCallback:   hostKeyCallback,
		HostKeyAlgorithms: hostKeyAlgorithms,
		Timeout:           time.Duration(timeoutSec) * time.Second,
	}, nil
}

func buildAuthMethods(host *HostConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if host.PrivateKey != "" || host.PrivateKeyPath != "" {
		key, err := loadPrivateKey(host)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(key))
	}
	if host.Password != "" {
		methods = append(methods, ssh.Password(host.Password))
	}
	return methods, nil
}

func loadPrivateKey(host *HostConfig) (ssh.Signer, error) {
	var keyBytes []byte
	if host.PrivateKey != "" {
		keyBytes = []byte(host.PrivateKey)
	} else if host.PrivateKeyPath != "" {
		data, err := os.ReadFile(host.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read private key: %w", err)
		}
		keyBytes = data
	} else {
		return nil, fmt.Errorf("no private key configured")
	}
	if host.PrivateKeyPassphrase != "" {
		return ssh.ParsePrivateKeyWithPassphrase(keyBytes, []byte(host.PrivateKeyPassphrase))
	}
	return ssh.ParsePrivateKey(keyBytes)
}

func buildHostKeyCallback(host *HostConfig) (ssh.HostKeyCallback, error) {
	mode := host.HostKey.Mode
	if mode == "" {
		mode = "known_hosts"
	}
	switch mode {
	case "insecure_ignore":
		return ssh.InsecureIgnoreHostKey(), nil
	case "known_hosts":
		path := host.HostKey.KnownHostsPath
		if path == "" {
			return nil, fmt.Errorf("known_hosts path is empty for host %s", host.Name)
		}
		callback, err := knownhosts.New(path)
		if err != nil {
			return nil, fmt.Errorf("load known_hosts: %w", err)
		}
		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return callback(hostname, remote, key)
		}, nil
	default:
		return nil, fmt.Errorf("unknown host_key.mode %q for host %s", mode, host.Name)
	}
}

func resolveHostKeyAlgorithms(host *HostConfig) ([]string, error) {
	if len(host.HostKey.Algorithms) > 0 {
		return slices.Clone(host.HostKey.Algorithms), nil
	}
	if host.HostKey.Mode != "known_hosts" {
		return nil, nil
	}
	return inferHostKeyAlgorithms(host.Address, host.Port, host.HostKey.KnownHostsPath)
}

func inferHostKeyAlgorithms(address string, port int, knownHostsPath string) ([]string, error) {
	if knownHostsPath == "" {
		return nil, nil
	}

	file, err := os.Open(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("open known_hosts: %w", err)
	}
	defer file.Close()

	target := knownhosts.Normalize(net.JoinHostPort(address, strconv.Itoa(port)))
	scanner := bufio.NewScanner(file)

	var algorithms []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		marker, hosts, pubKey, _, _, err := ssh.ParseKnownHosts([]byte(line))
		if err != nil {
			continue
		}
		if marker == "revoked" {
			continue
		}

		matched := false
		for _, host := range hosts {
			if matchesKnownHost(host, target) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		for _, algo := range expandHostKeyAlgorithms(pubKey.Type()) {
			if !slices.Contains(algorithms, algo) {
				algorithms = append(algorithms, algo)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan known_hosts: %w", err)
	}
	if len(algorithms) == 0 {
		return nil, nil
	}
	return algorithms, nil
}

func matchesKnownHost(pattern, target string) bool {
	if pattern == "" {
		return false
	}
	if strings.HasPrefix(pattern, "!") {
		return false
	}
	if strings.HasPrefix(pattern, "|") {
		return false
	}
	if strings.ContainsAny(pattern, "*?") {
		return false
	}
	return knownhosts.Normalize(pattern) == target
}

func expandHostKeyAlgorithms(keyType string) []string {
	switch keyType {
	case ssh.KeyAlgoRSA:
		return []string{ssh.KeyAlgoRSASHA512, ssh.KeyAlgoRSASHA256, ssh.KeyAlgoRSA}
	default:
		return []string{keyType}
	}
}
