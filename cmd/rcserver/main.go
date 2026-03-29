package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"github.com/rcservers/rcserver/internal/api"
	"github.com/rcservers/rcserver/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "serve":
		runServe(os.Args[2:])
	case "generate":
		if len(os.Args) >= 3 && os.Args[2] == "hash" {
			runGenerateHash(os.Args[3:])
			return
		}
		fallthrough
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Kullanım:\n  %s serve [--config path]\n  %s generate hash [--config path]\n", os.Args[0], os.Args[0])
}

func configPathFromArgs(args []string) string {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	cfgPath := fs.String("config", envOr("RC_SERVER_CONFIG", config.DefaultConfigPath), "yapılandırma dosyası")
	_ = fs.Parse(args)
	return *cfgPath
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func runServe(args []string) {
	cfgPath := configPathFromArgs(args)
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("yapılandırma okunamadı (%s): %v", cfgPath, err)
	}
	if cfg.Hash == "" {
		log.Fatal("config içinde hash boş; önce 'rcserver generate hash' çalıştırın")
	}

	h := api.NewRouter(cfg)

	addr := cfg.ListenAddr
	if cfg.TLSEnabled {
		if cfg.TLSCert == "" || cfg.TLSKey == "" {
			log.Fatal("tls_enabled için tls_cert ve tls_key gerekli")
		}
		log.Printf("HTTPS dinleniyor: %s", addr)
		log.Fatal(http.ListenAndServeTLS(addr, cfg.TLSCert, cfg.TLSKey, h))
	}
	log.Printf("HTTP dinleniyor: %s (üretimde TLS önerilir)", addr)
	log.Fatal(http.ListenAndServe(addr, h))
}

func runGenerateHash(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	cfgPath := fs.String("config", envOr("RC_SERVER_CONFIG", config.DefaultConfigPath), "yapılandırma dosyası")
	_ = fs.Parse(args)

	cfg, err := config.EnsureDefaultFile(*cfgPath)
	if err != nil {
		log.Fatalf("yapılandırma: %v", err)
	}
	newHash, err := config.GenerateHashString()
	if err != nil {
		log.Fatal(err)
	}
	cfg.Hash = newHash
	if err := config.Save(*cfgPath, cfg); err != nil {
		log.Fatal(err)
	}
	ip := config.PrimaryIPv4()
	port := config.PortFromListen(cfg.ListenAddr)
	fmt.Print(config.FormatBox(ip, port, cfg.Hash))
}
