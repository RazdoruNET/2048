package modifiers

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strings"
)

// TLSConfig represents dynamic TLS configuration
type TLSConfig struct {
	MinVersion   string   `yaml:"min_version"`
	CipherSuites []string `yaml:"cipher_suites"`
	ALPN         []string `yaml:"alpn"`
	ServerName   string   `yaml:"server_name"`
	Insecure     bool     `yaml:"insecure"`
}

// DefaultTLSConfig returns default TLS configuration for VLESS
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion: "1.3",
		CipherSuites: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		},
		ALPN: []string{"h2", "http/1.1"},
	}
}

// ChromeTLSConfig returns TLS config mimicking Chrome browser
func ChromeTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion: "1.3",
		CipherSuites: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
			"TLS_CHACHA20_POLY1305_SHA256",
		},
		ALPN: []string{"h2", "http/1.1"},
	}
}

// FirefoxTLSConfig returns TLS config mimicking Firefox browser
func FirefoxTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion: "1.3",
		CipherSuites: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_CHACHA20_POLY1305_SHA256",
			"TLS_AES_256_GCM_SHA384",
		},
		ALPN: []string{"h2", "http/1.1"},
	}
}

// ToTLSConfig converts TLSConfig to crypto/tls.Config
func (tc *TLSConfig) ToTLSConfig() (*tls.Config, error) {
	config := &tls.Config{
		ServerName:         tc.ServerName,
		InsecureSkipVerify: tc.Insecure,
		NextProtos:         tc.ALPN,
	}

	// Set minimum version
	if err := tc.setMinVersion(config); err != nil {
		return nil, err
	}

	// Set cipher suites
	if err := tc.setCipherSuites(config); err != nil {
		return nil, err
	}

	return config, nil
}

// setMinVersion configures minimum TLS version
func (tc *TLSConfig) setMinVersion(config *tls.Config) error {
	switch tc.MinVersion {
	case "1.2":
		config.MinVersion = tls.VersionTLS12
	case "1.3":
		config.MinVersion = tls.VersionTLS13
	case "":
		// Use default
		config.MinVersion = tls.VersionTLS12
	default:
		return fmt.Errorf("unsupported TLS version: %s", tc.MinVersion)
	}
	return nil
}

// setCipherSuites configures cipher suites
func (tc *TLSConfig) setCipherSuites(config *tls.Config) error {
	if len(tc.CipherSuites) == 0 {
		return nil
	}

	var suites []uint16
	for _, suiteName := range tc.CipherSuites {
		suite, err := tc.getCipherSuite(suiteName)
		if err != nil {
			return err
		}
		suites = append(suites, suite)
	}

	config.CipherSuites = suites
	return nil
}

// getCipherSuite converts string cipher name to uint16
func (tc *TLSConfig) getCipherSuite(name string) (uint16, error) {
	switch strings.ToUpper(name) {
	case "TLS_RSA_WITH_RC4_128_SHA":
		return tls.TLS_RSA_WITH_RC4_128_SHA, nil
	case "TLS_RSA_WITH_3DES_EDE_CBC_SHA":
		return tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA, nil
	case "TLS_RSA_WITH_AES_128_CBC_SHA":
		return tls.TLS_RSA_WITH_AES_128_CBC_SHA, nil
	case "TLS_RSA_WITH_AES_256_CBC_SHA":
		return tls.TLS_RSA_WITH_AES_256_CBC_SHA, nil
	case "TLS_RSA_WITH_AES_128_CBC_SHA256":
		return tls.TLS_RSA_WITH_AES_128_CBC_SHA256, nil
	case "TLS_RSA_WITH_AES_128_GCM_SHA256":
		return tls.TLS_RSA_WITH_AES_128_GCM_SHA256, nil
	case "TLS_RSA_WITH_AES_256_GCM_SHA384":
		return tls.TLS_RSA_WITH_AES_256_GCM_SHA384, nil
	case "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":
		return tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA, nil
	case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":
		return tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA, nil
	case "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":
		return tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA, nil
	case "TLS_ECDHE_RSA_WITH_RC4_128_SHA":
		return tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA, nil
	case "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":
		return tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA, nil
	case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":
		return tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA, nil
	case "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":
		return tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA, nil
	case "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256":
		return tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256, nil
	case "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":
		return tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, nil
	case "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":
		return tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, nil
	case "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":
		return tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, nil
	case "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":
		return tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, nil
	case "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":
		return tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, nil
	case "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":
		return tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, nil
	case "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":
		return tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305, nil
	case "TLS_AES_128_GCM_SHA256":
		return tls.TLS_AES_128_GCM_SHA256, nil
	case "TLS_AES_256_GCM_SHA384":
		return tls.TLS_AES_256_GCM_SHA384, nil
	case "TLS_CHACHA20_POLY1305_SHA256":
		return tls.TLS_CHACHA20_POLY1305_SHA256, nil
	default:
		return 0, fmt.Errorf("unsupported cipher suite: %s", name)
	}
}

// Validate validates TLS configuration
func (tc *TLSConfig) Validate() error {
	if tc.MinVersion != "" {
		validVersions := []string{"1.2", "1.3"}
		valid := false
		for _, v := range validVersions {
			if tc.MinVersion == v {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid TLS version: %s", tc.MinVersion)
		}
	}

	for _, suite := range tc.CipherSuites {
		if _, err := tc.getCipherSuite(suite); err != nil {
			return fmt.Errorf("invalid cipher suite: %s", suite)
		}
	}

	return nil
}

// Clone creates a copy of TLSConfig
func (tc *TLSConfig) Clone() *TLSConfig {
	if tc == nil {
		return nil
	}

	clone := *tc
	if tc.CipherSuites != nil {
		clone.CipherSuites = make([]string, len(tc.CipherSuites))
		copy(clone.CipherSuites, tc.CipherSuites)
	}
	if tc.ALPN != nil {
		clone.ALPN = make([]string, len(tc.ALPN))
		copy(clone.ALPN, tc.ALPN)
	}

	return &clone
}

// CreateCustomCertPool creates a custom certificate pool for testing
func CreateCustomCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	// Add custom certificates here if needed
	return pool
}
