package internal

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
)

const kRootPEM = `
-----BEGIN CERTIFICATE-----
MIICQzCCAcmgAwIBAgIILcX8iNLFS5UwCgYIKoZIzj0EAwMwZzEbMBkGA1UEAwwS
QXBwbGUgUm9vdCBDQSAtIEczMSYwJAYDVQQLDB1BcHBsZSBDZXJ0aWZpY2F0aW9u
IEF1dGhvcml0eTETMBEGA1UECgwKQXBwbGUgSW5jLjELMAkGA1UEBhMCVVMwHhcN
MTQwNDMwMTgxOTA2WhcNMzkwNDMwMTgxOTA2WjBnMRswGQYDVQQDDBJBcHBsZSBS
b290IENBIC0gRzMxJjAkBgNVBAsMHUFwcGxlIENlcnRpZmljYXRpb24gQXV0aG9y
aXR5MRMwEQYDVQQKDApBcHBsZSBJbmMuMQswCQYDVQQGEwJVUzB2MBAGByqGSM49
AgEGBSuBBAAiA2IABJjpLz1AcqTtkyJygRMc3RCV8cWjTnHcFBbZDuWmBSp3ZHtf
TjjTuxxEtX/1H7YyYl3J6YRbTzBPEVoA/VhYDKX1DyxNB0cTddqXl5dvMVztK517
IDvYuVTZXpmkOlEKMaNCMEAwHQYDVR0OBBYEFLuw3qFYM4iapIqZ3r6966/ayySr
MA8GA1UdEwEB/wQFMAMBAf8wDgYDVR0PAQH/BAQDAgEGMAoGCCqGSM49BAMDA2gA
MGUCMQCD6cHEFl4aXTQY2e3v9GwOAEZLuN+yRhHFD/3meoyhpmvOwgPUnPWTxnS4
at+qIxUCMG1mihDK1A3UT82NQz60imOlM27jbdoXt2QfyFMm+YhidDkLF1vLUagM
6BgD56KyKA==
-----END CERTIFICATE-----
`

func DecodeClaims(payload string, claims jwt.Claims) error {
	rootCert, err := DecodeCert(payload, 2)
	if err != nil {
		return err
	}
	intermediateCert, err := DecodeCert(payload, 1)
	if err != nil {
		return err
	}
	if err = VerifyCert(rootCert, intermediateCert); err != nil {
		return err
	}
	if _, err = jwt.ParseWithClaims(payload, claims, func(token *jwt.Token) (interface{}, error) {
		return DecodePublicKey(payload)
	}); err != nil {
		return err
	}
	return nil
}

func DecodePublicKey(payload string) (*ecdsa.PublicKey, error) {
	cert, err := DecodeCert(payload, 0)
	if err != nil {
		return nil, err
	}
	switch pk := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		return pk, nil
	default:
		return nil, errors.New("appstore public key must be of type ecdsa.PublicKey")
	}
}

type Header struct {
	Alg string   `json:"alg"`
	X5C []string `json:"x5c"`
}

func DecodeCert(payload string, index int) (*x509.Certificate, error) {
	if index > 2 {
		return nil, errors.New("invalid index")
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.Split(payload, ".")[0])
	if err != nil {
		return nil, err
	}

	var header *Header
	err = json.Unmarshal(data, &header)
	if err != nil {
		return nil, err
	}

	if len(header.X5C) < index {
		return nil, fmt.Errorf("invalid index")
	}

	certBytes, err := base64.StdEncoding.DecodeString(header.X5C[index])
	if err != nil {
		return nil, err
	}

	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, err
	}
	return cert, nil
}

func VerifyCert(rootCert, intermediateCert *x509.Certificate) error {
	var roots = x509.NewCertPool()
	if !roots.AppendCertsFromPEM([]byte(kRootPEM)) {
		return errors.New("failed to parse root certificate")
	}

	var intermediates = x509.NewCertPool()
	intermediates.AddCert(intermediateCert)

	var opts = x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
	}
	_, err := rootCert.Verify(opts)
	return err
}
