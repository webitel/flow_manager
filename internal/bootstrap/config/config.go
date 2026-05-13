package config

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"os"

	"github.com/BoRuDar/configuration/v4"
)

func Load() (*Config, error) {
	var cfg Config
	configurator := configuration.New(
		&cfg,
		configuration.NewEnvProvider(),
		configuration.NewFlagProvider(),
		configuration.NewDefaultProvider(),
	).SetOptions(configuration.OnFailFnOpt(func(err error) {}))

	if err := configurator.InitValues(); err != nil {
		// non-fatal: missing flags/envs use defaults
	}

	if cfg.ConfigFile != nil && *cfg.ConfigFile != "" {
		f, err := os.OpenFile(*cfg.ConfigFile, os.O_RDONLY, 0o644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		body, err := io.ReadAll(f)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &cfg); err != nil {
			return nil, err
		}
	}

	if !cfg.Log.Console && !cfg.Log.Otel && len(cfg.Log.File) == 0 {
		cfg.Log.Console = true
	}

	return &cfg, nil
}

func LoadTLSCreds(cfg TLSConfig) (*tls.Config, error) {
	if len(cfg.CertPath) == 0 || len(cfg.KeyPath) == 0 || len(cfg.CAPath) == 0 {
		return nil, nil
	}

	clientCert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
	if err != nil {
		return nil, err
	}

	caCert, err := os.ReadFile(cfg.CAPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		ServerName:   "im-gateway-service",
	}, nil
}
