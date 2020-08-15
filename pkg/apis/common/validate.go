package common

import "errors"

// ValidateTLSConfig validates a TLS configuration.
func ValidateTLSConfig(tlsConfig *TLSConfig) error {
	if tlsConfig == nil {
		return nil
	}
	if tlsConfig.ClientKeySecret != nil && tlsConfig.ClientCertSecret != nil && tlsConfig.CACertSecret != nil {
		return nil
	}
	// DEPRECATED.
	if tlsConfig.DeprecatedClientCertPath != "" && tlsConfig.DeprecatedClientKeyPath != "" && tlsConfig.DeprecatedCACertPath != "" {
		return nil
	}
	return errors.New("invalid tls config, please configure caCertSecret, clientCertSecret and clientKeySecret")
}
