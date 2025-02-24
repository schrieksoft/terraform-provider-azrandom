// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/crypto/ssh"
)

// keyGenerator extracts data from the given *schema.ResourceData,
// and generates a new public/private key-pair according to the
// selected algorithm.
type keyGenerator func(prvKeyConf *cryptographicKeyModelV0) (crypto.PrivateKey, error)

// keyParser parses a private key from the given []byte,
// according to the selected algorithm.
type keyParser func([]byte) (crypto.PrivateKey, error)

// HMACSHA256Key is an implementation of crypto.PrivateKey
type HMACSHA256Key []byte

func (k HMACSHA256Key) Public() crypto.PublicKey {
	return nil // HMAC-SHA256 doesn't have a separate public key
}

func (k HMACSHA256Key) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	if opts != nil {
		return nil, errors.New("unsupported options")
	}
	mac := hmac.New(sha256.New, k)
	mac.Write(digest)
	return mac.Sum(nil), nil
}

func (k HMACSHA256Key) Bytes() []byte {
	return []byte(k)
}

var keyGenerators = map[Algorithm]keyGenerator{
	RSA: func(prvKeyConf *cryptographicKeyModelV0) (crypto.PrivateKey, error) {
		if prvKeyConf.RSABits.IsUnknown() || prvKeyConf.RSABits.IsNull() {
			return nil, fmt.Errorf("RSA bits curve not provided")
		}

		return rsa.GenerateKey(rand.Reader, int(prvKeyConf.RSABits.ValueInt64()))
	},
	ECDSA: func(prvKeyConf *cryptographicKeyModelV0) (crypto.PrivateKey, error) {
		if prvKeyConf.ECDSACurve.IsUnknown() || prvKeyConf.ECDSACurve.IsNull() {
			return nil, fmt.Errorf("ECDSA curve not provided")
		}

		curve := ECDSACurve(prvKeyConf.ECDSACurve.ValueString())
		switch curve {
		case P224:
			return ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
		case P256:
			return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		case P384:
			return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		case P521:
			return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		default:
			return nil, fmt.Errorf("invalid ECDSA curve; supported values are: %v", supportedECDSACurves())
		}
	},
	ED25519: func(_ *cryptographicKeyModelV0) (crypto.PrivateKey, error) {
		_, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ED25519 key: %s", err)
		}
		return key, err
	},
	HMAC: func(prvKeyConf *cryptographicKeyModelV0) (crypto.PrivateKey, error) {
		if prvKeyConf.HMACHashFunction.IsUnknown() || prvKeyConf.HMACHashFunction.IsNull() {
			return nil, fmt.Errorf("HMAC hash function not provided")
		}

		hash_function := HMACHashFunction(prvKeyConf.HMACHashFunction.ValueString())
		switch hash_function {
		case SHA256:
			{
				key := make([]byte, 32) // 32 bytes for HMAC-SHA256 (256 bits)
				_, err := rand.Read(key)
				if err != nil {
					return nil, fmt.Errorf("Error generating random key: %s", err)
				}
				return HMACSHA256Key(key), nil
			}

		default:
			return nil, fmt.Errorf("invalid ECDSA curve; supported values are: %v", supportedECDSACurves())
		}
	},
}

// privateKeyToPublicKey takes a crypto.PrivateKey and extracts the corresponding crypto.PublicKey,
// after having figured out its type.
func privateKeyToPublicKey(prvKey crypto.PrivateKey) (crypto.PublicKey, error) {
	signer, ok := prvKey.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("unsupported private key type: %T", prvKey)
	}

	return signer.Public(), nil
}

type PublicKeyBundle struct {
	PublicKeyPem               string
	PublicKeySSH               string
	PublicKeyFingerPrintMD5    string
	PublicKeyFingerPrintSHA256 string
}

func getPublicKeyBundle(ctx context.Context, prvKey crypto.PrivateKey) (PublicKeyBundle, error) {

	var pubKeyBundle PublicKeyBundle

	if _, ok := prvKey.(HMACSHA256Key); ok {
		// HMAC keys are symmetric keys, therefore do no have public keys
		return pubKeyBundle, nil
	}

	pubKey, err := privateKeyToPublicKey(prvKey)
	if err != nil {
		return pubKeyBundle, errors.New("Failed to get public key from private key" + err.Error())
	}
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return pubKeyBundle, errors.New("Failed to marshal public key" + err.Error())
	}

	pubKeyPemBlock := &pem.Block{
		Type:  PreamblePublicKey.String(),
		Bytes: pubKeyBytes,
	}

	// NOTE: ECDSA keys with elliptic curve P-224 are not supported by `x/crypto/ssh`,
	// so this will return an error: in that case, we set the below fields to empty strings
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	var pubKeySSH, pubKeySSHFingerprintMD5, pubKeySSHFingerprintSHA256 string
	if err == nil {
		sshPubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)

		pubKeySSH = string(sshPubKeyBytes)
		pubKeySSHFingerprintMD5 = ssh.FingerprintLegacyMD5(sshPubKey)
		pubKeySSHFingerprintSHA256 = ssh.FingerprintSHA256(sshPubKey)
	}

	pubKeyBundle.PublicKeyPem = string(pem.EncodeToMemory(pubKeyPemBlock))
	pubKeyBundle.PublicKeySSH = pubKeySSH
	pubKeyBundle.PublicKeyFingerPrintMD5 = pubKeySSHFingerprintMD5
	pubKeyBundle.PublicKeyFingerPrintSHA256 = pubKeySSHFingerprintSHA256

	return pubKeyBundle, nil
}

// hashForState computes the hexadecimal representation of the SHA1 checksum of a string.
// This is used by most resources/data-sources here to compute their Unique Identifier (ID).
func hashForState(value string) string {
	if value == "" {
		return ""
	}
	hash := sha1.Sum([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(hash[:])
}

// Algorithm represents a type of private key algorithm.
type Algorithm string

const (
	RSA     Algorithm = "RSA"
	ECDSA   Algorithm = "ECDSA"
	ED25519 Algorithm = "ED25519"
	HMAC    Algorithm = "HMAC"
)

func (a Algorithm) String() string {
	return string(a)
}

// supportedAlgorithms returns a slice of Algorithm currently supported by this provider.
func supportedAlgorithms() []Algorithm {
	return []Algorithm{
		RSA,
		ECDSA,
		ED25519,
		HMAC,
	}
}

// supportedAlgorithmsStr returns the same content of supportedAlgorithms but as a slice of string.
func supportedAlgorithmsStr() []string {
	supported := supportedAlgorithms()
	supportedStr := make([]string, len(supported))
	for i := range supported {
		supportedStr[i] = supported[i].String()
	}
	return supportedStr
}

// ECDSACurve represents a type of ECDSA elliptic curve.
type ECDSACurve string

const (
	P224 ECDSACurve = "P224"
	P256 ECDSACurve = "P256"
	P384 ECDSACurve = "P384"
	P521 ECDSACurve = "P521"
)

func (e ECDSACurve) String() string {
	return string(e)
}

// supportedECDSACurves returns an array of ECDSACurve currently supported by this provider.
func supportedECDSACurves() []ECDSACurve {
	return []ECDSACurve{
		P224,
		P256,
		P384,
		P521,
	}
}

// supportedECDSACurvesStr returns the same content of supportedECDSACurves but as a slice of string.
func supportedECDSACurvesStr() []string {
	supported := supportedECDSACurves()
	supportedStr := make([]string, len(supported))
	for i := range supported {
		supportedStr[i] = supported[i].String()
	}
	return supportedStr
}

type HMACHashFunction string

func (e HMACHashFunction) String() string {
	return string(e)
}

const (
	SHA256 HMACHashFunction = "SHA256"
)

// supportedHMACHashFunctions returns an array of HMACHashFunction currently supported by this provider.
func supportedHMACHashFunctions() []HMACHashFunction {
	return []HMACHashFunction{
		SHA256,
	}
}

// supportedHMACHashFunctionsStr returns the same content of supportedHMACHashFunctions but as a slice of string.
func supportedHMACHashFunctionsStr() []string {
	supported := supportedHMACHashFunctions()
	supportedStr := make([]string, len(supported))
	for i := range supported {
		supportedStr[i] = supported[i].String()
	}
	return supportedStr
}

// PEMPreamble represents the heading used in a PEM-formatted for the "encapsulation boundaries",
// that is used to delimit the "encapsulated text portion" of cryptographic documents.
//
// See https://datatracker.ietf.org/doc/html/rfc1421 and https://datatracker.ietf.org/doc/html/rfc7468.
type PEMPreamble string

const (
	PreamblePublicKey PEMPreamble = "PUBLIC KEY"

	PreamblePrivateKeyPKCS8   PEMPreamble = "PRIVATE KEY"
	PreamblePrivateKeyHMAC    PEMPreamble = "SYMMETRIC KEY"
	PreamblePrivateKeyRSA     PEMPreamble = "RSA PRIVATE KEY"
	PreamblePrivateKeyEC      PEMPreamble = "EC PRIVATE KEY"
	PreamblePrivateKeyOpenSSH PEMPreamble = "OPENSSH PRIVATE KEY"

	PreambleCertificate        PEMPreamble = "CERTIFICATE"
	PreambleCertificateRequest PEMPreamble = "CERTIFICATE REQUEST"
)

func (p PEMPreamble) String() string {
	return string(p)
}

// pemBlockToPEMPreamble takes a pem.Block and returns the related PEMPreamble, if supported.
func pemBlockToPEMPreamble(block *pem.Block) (PEMPreamble, error) {
	switch block.Type {
	case PreamblePublicKey.String():
		return PreamblePublicKey, nil
	case PreamblePrivateKeyPKCS8.String():
		return PreamblePrivateKeyPKCS8, nil
	case PreamblePrivateKeyRSA.String():
		return PreamblePrivateKeyRSA, nil
	case PreamblePrivateKeyHMAC.String():
		return PreamblePrivateKeyHMAC, nil
	case PreamblePrivateKeyEC.String():
		return PreamblePrivateKeyEC, nil
	case PreambleCertificate.String():
		return PreambleCertificate, nil
	case PreambleCertificateRequest.String():
		return PreambleCertificateRequest, nil
	default:
		return "", fmt.Errorf("unsupported PEM preamble/type: %s", block.Type)
	}
}

// ProxyScheme represents url schemes supported when providing proxy configuration to this provider.
type ProxyScheme string

const (
	HTTPProxy   ProxyScheme = "http"
	HTTPSProxy  ProxyScheme = "https"
	SOCKS5Proxy ProxyScheme = "socks5"
)

func (p ProxyScheme) String() string {
	return string(p)
}

// supportedProxySchemes returns an array of ProxyScheme currently supported by this provider.
func supportedProxySchemes() []ProxyScheme {
	return []ProxyScheme{
		HTTPProxy,
		HTTPSProxy,
		SOCKS5Proxy,
	}
}

// supportedProxySchemesStr returns the same content of supportedProxySchemes but as a slice of string.
func supportedProxySchemesStr() []string {
	supported := supportedProxySchemes()
	supportedStr := make([]string, len(supported))
	for i := range supported {
		supportedStr[i] = string(supported[i])
	}
	return supportedStr
}

// URLScheme represents url schemes supported by resources and data-sources of this provider.
type URLScheme string

const (
	HTTPSScheme URLScheme = "https"
	TLSScheme   URLScheme = "tls"
)

func (p URLScheme) String() string {
	return string(p)
}

// supportedURLSchemes returns an array of URLScheme currently supported by this provider.
func supportedURLSchemes() []URLScheme {
	return []URLScheme{
		HTTPSScheme,
		TLSScheme,
	}
}

// supportedURLSchemesStr returns the same content of supportedURLSchemes but as a slice of string.
func supportedURLSchemesStr() []string {
	supported := supportedURLSchemes()
	supportedStr := make([]string, len(supported))
	for i := range supported {
		supportedStr[i] = string(supported[i])
	}
	return supportedStr
}

func createKey(ctx context.Context, plan cryptographicKeyModelV0) (crypto.PrivateKey, *pem.Block, error) {
	keyAlgoName := Algorithm(plan.Algorithm.ValueString())

	var emptyKey crypto.PrivateKey
	var emptyBlock *pem.Block
	// Identify the correct (Private) Key Generator
	var keyGen keyGenerator
	var ok bool
	if keyGen, ok = keyGenerators[keyAlgoName]; !ok {
		return emptyKey, emptyBlock, errors.New("Invalid Key Algorithm" + fmt.Sprintf("Key Algorithm %q is not supported", keyAlgoName))
	}

	// Generate the new Key
	tflog.Debug(ctx, "Generating private key for algorithm", map[string]interface{}{
		"algorithm": keyAlgoName,
	})
	prvKey, err := keyGen(&plan)
	if err != nil {
		return emptyKey, emptyBlock, errors.New("Unable to generate Key from configuration" + err.Error())
	}

	// Marshal the Key in PEM block
	tflog.Debug(ctx, "Marshalling private key to PEM")
	var prvKeyPemBlock *pem.Block

	switch k := prvKey.(type) {
	case *rsa.PrivateKey:
		prvKeyPemBlock = &pem.Block{
			Type:  PreamblePrivateKeyRSA.String(),
			Bytes: x509.MarshalPKCS1PrivateKey(k),
		}
	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return emptyKey, emptyBlock, errors.New("Unable to encode key to PEM" + err.Error())
		}

		prvKeyPemBlock = &pem.Block{
			Type:  PreamblePrivateKeyEC.String(),
			Bytes: keyBytes,
		}
	case ed25519.PrivateKey:
		prvKeyBytes, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return emptyKey, emptyBlock, errors.New("Unable to encode key to PEM" + err.Error())
		}

		prvKeyPemBlock = &pem.Block{
			Type:  PreamblePrivateKeyPKCS8.String(),
			Bytes: prvKeyBytes,
		}
	case HMACSHA256Key:
		if hmacKey, ok := prvKey.(HMACSHA256Key); ok {
			prvKeyPemBlock = &pem.Block{
				Type:  PreamblePrivateKeyHMAC.String(),
				Bytes: hmacKey.Bytes(),
			}
		} else {
			return emptyKey, emptyBlock, errors.New("Unable to encode key to PEM" + err.Error())
		}
	default:
		return emptyKey, emptyBlock, errors.New("Unsupported private key type. Key type not supported")
	}

	return prvKey, prvKeyPemBlock, nil

}
