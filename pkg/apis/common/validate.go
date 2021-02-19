package common

import "errors"

// ValidateTLSConfig validates a TLS configuration.
func ValidateTLSConfig(tlsConfig *TLSConfig) error {
	if tlsConfig == nil {
		return nil
	}
	var caCertSet, clientCertSet, clientKeySet bool

	if tlsConfig.CACertSecret != nil || tlsConfig.DeprecatedCACertPath != "" {
		caCertSet = true
	}

	if tlsConfig.ClientCertSecret != nil || tlsConfig.DeprecatedClientCertPath != "" {
		clientCertSet = true
	}

	if tlsConfig.ClientKeySecret != nil || tlsConfig.DeprecatedClientKeyPath != "" {
		clientKeySet = true
	}

	if !caCertSet && !clientCertSet && !clientKeySet {
		return errors.New("invalid tls config, please configure either caCertSecret, or clientCertSecret and clientKeySecret, or both")
	}

	if (clientCertSet || clientKeySet) && (!clientCertSet || !clientKeySet) {
		return errors.New("invalid tls config, both clientCertSecret and clientKeySecret need to be configured")
	}
	return nil
}
