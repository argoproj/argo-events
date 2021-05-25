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

func ValidateSASLConfig(saslConfig *SASLConfig) error {
	if saslConfig == nil {
		return nil
	}

	switch saslConfig.Mechanism {
	case "", "PLAIN", "OAUTHBEARER", "SCRAM-SHA-256", "SCRAM-SHA-512", "GSSAPI":
	default:
		return errors.New("invalid sasl config. Possible values for SASL Mechanism are `OAUTHBEARER`, `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512` and `GSSAPI`")
	}

	// user and password must both be set
	if saslConfig.UserSecret == nil || saslConfig.PasswordSecret == nil {
		return errors.New("invalid sasl config, both userSecret and passwordSecret must be defined")
	}

	return nil
}
