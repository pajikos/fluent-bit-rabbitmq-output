package main

import (
	"crypto/x509"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func parseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return strconv.ParseBool(value)
	}
}

func loadCACert(caCertFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if ok := caCertPool.AppendCertsFromPEM(caCert); !ok {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return caCertPool, nil
}
