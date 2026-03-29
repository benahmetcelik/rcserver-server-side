package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultListenAddr   = ":3300"
	DefaultConfigPath     = "/etc/rcserver/config.yaml"
	DefaultTLSCert      = "/etc/rcserver/tls.crt"
	DefaultTLSKey       = "/etc/rcserver/tls.key"
	DefaultNginxSitesDir  = "/etc/nginx/sites-available"
	DefaultWWWRoot        = "/var/www"
	DefaultStateDir       = "/var/lib/rcserver"
)

type Config struct {
	ListenAddr    string   `yaml:"listen_addr"`
	Hash          string   `yaml:"hash"`
	TLSCert       string   `yaml:"tls_cert"`
	TLSKey        string   `yaml:"tls_key"`
	TLSEnabled    bool     `yaml:"tls_enabled"`
	FileRoots     []string `yaml:"file_roots"`
	NginxSitesDir string   `yaml:"nginx_sites_dir"`
	WWWRoot       string   `yaml:"www_root"`
	DeployDir     string   `yaml:"deploy_dir"`
	RatePerSecond float64  `yaml:"rate_per_second"`
	RateBurst     int      `yaml:"rate_burst"`
	ExecTimeoutSec int     `yaml:"exec_timeout_sec"`
	MaxOutputBytes int     `yaml:"max_output_bytes"`
}

func Default() *Config {
	deploy := filepath.Join(DefaultWWWRoot, "sites")
	return &Config{
		ListenAddr:     DefaultListenAddr,
		TLSCert:        DefaultTLSCert,
		TLSKey:         DefaultTLSKey,
		TLSEnabled:     true,
		FileRoots:      []string{DefaultWWWRoot, deploy, DefaultStateDir},
		NginxSitesDir:  DefaultNginxSitesDir,
		WWWRoot:        DefaultWWWRoot,
		DeployDir:      deploy,
		RatePerSecond:  20,
		RateBurst:      40,
		ExecTimeoutSec: 120,
		MaxOutputBytes: 2 * 1024 * 1024,
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := Default()
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if c.DeployDir == "" {
		c.DeployDir = filepath.Join(c.WWWRoot, "sites")
	}
	if len(c.FileRoots) == 0 {
		c.FileRoots = []string{c.WWWRoot, DefaultStateDir}
	}
	return c, nil
}

func Save(path string, c *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func EnsureDefaultFile(path string) (*Config, error) {
	if _, err := os.Stat(path); err == nil {
		return Load(path)
	}
	c := Default()
	c.Hash, _ = GenerateHashString()
	if err := Save(path, c); err != nil {
		return nil, err
	}
	return c, nil
}

func GenerateHashString() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func PortFromListen(addr string) int {
	addr = strings.TrimSpace(addr)
	if strings.HasPrefix(addr, ":") {
		p, err := strconv.Atoi(strings.TrimPrefix(addr, ":"))
		if err == nil {
			return p
		}
	}
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil || host == "" {
		if p, err := strconv.Atoi(strings.TrimPrefix(addr, ":")); err == nil {
			return p
		}
		return 3300
	}
	p, _ := strconv.Atoi(portStr)
	return p
}

func PrimaryIPv4() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "127.0.0.1"
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if v4 := ipnet.IP.To4(); v4 != nil {
					return v4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

func FormatBox(ip string, port int, hash string) string {
	line := func(label, val string) string {
		return fmt.Sprintf("| %s : %-40s |", label, val)
	}
	sep := strings.Repeat("-", 50)
	return fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		sep,
		line("IP", ip),
		line("PORT", fmt.Sprintf("%d", port)),
		line("HASH", hash),
		sep,
	)
}
