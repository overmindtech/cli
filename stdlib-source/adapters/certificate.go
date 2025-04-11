package adapters

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
)

// CertToName Returns the name of a cert as a string. This is in the format of:
//
// {Subject.CommonName} (SHA-256: {fingerprint})
func CertToName(cert *x509.Certificate) string {
	sum := sha256.Sum256(cert.Raw)
	hexString := toHex(sum[:])

	return fmt.Sprintf(
		"%v (SHA-256: %v)",
		cert.Subject.CommonName,
		hexString,
	)
}

// toHex converts bytes to their uppercase hex representation, separated by colons
func toHex(b []byte) string {
	if len(b) == 0 {
		return ""
	}

	buf := make([]byte, 0, 3*len(b))
	x := buf[1*len(b) : 3*len(b)]
	hex.Encode(x, b)
	for i := 0; i < len(x); i += 2 {
		buf = append(buf, x[i], x[i+1], ':')
	}
	s := strings.TrimSuffix(string(buf), ":")

	return strings.ToUpper(s)
}

// CertificateAdapter This adapter only responds to Search() requests. See the
// docs for the Search() method for more info
type CertificateAdapter struct{}

// Type The type of items that this adapter is capable of finding
func (s *CertificateAdapter) Type() string {
	return "certificate"
}

// Descriptive name for the adapter, used in logging and metadata
func (s *CertificateAdapter) Name() string {
	return "stdlib-certificate"
}

func (s *CertificateAdapter) Metadata() *sdp.AdapterMetadata {
	return certificateMetadata
}

var certificateMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "Certificate",
	Type:            "certificate",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Search:            true,
		SearchDescription: "Takes a full certificate, or certificate bundle as input in PEM encoded format",
	},
	Category: sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

// List of scopes that this adapter is capable of find items for. If the
// adapter supports all scopes the special value `AllScopes` ("*")
// should be used
func (s *CertificateAdapter) Scopes() []string {
	return []string{
		"global", // This is a reserved word meaning that the items should be considered globally unique
	}
}

// Get This adapter does not respond to Get() requests. The logic here is that
// there are many places we might find a certificate, for example after making a
// HTTP connection, sitting on disk, after making a database connection, etc.
// Rather than implement a adapter that knows how to make each of these
// connections, instead we have created this adapter which takes the cert itself
// as an input to Search() and parses it and returns the info
func (s *CertificateAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	return nil, &sdp.QueryError{
		ErrorType:   sdp.QueryError_NOTFOUND,
		ErrorString: "certificate only responds to Search() requests. Consult the documentation",
		Scope:       scope,
	}
}

// List Is not implemented for HTTP as this would require scanning many
// endpoints or something, doesn't really make sense
func (s *CertificateAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	items := make([]*sdp.Item, 0)

	return items, nil
}

// Search This method takes a full certificate, or certificate bundle as input
// (in PEM encoded format), parses them, and returns a items, one for each
// certificate that was found
func (s *CertificateAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	var errors []error
	var items []*sdp.Item

	bundle, err := decodePem(query)

	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
			Scope:       scope,
		}
	}

	// Range over all the parsed certs
	for _, b := range bundle.Certificate {
		var cert *x509.Certificate
		var err error
		var attributes *sdp.ItemAttributes

		cert, err = x509.ParseCertificate(b)

		if err != nil {
			errors = append(errors, err)
			// Skip this cert
			continue
		}

		attributes, err = sdp.ToAttributes(map[string]interface{}{
			"issuer":             cert.Issuer.String(),
			"subject":            cert.Subject.String(),
			"notBefore":          cert.NotBefore.String(),
			"notAfter":           cert.NotAfter.String(),
			"signatureAlgorithm": cert.SignatureAlgorithm.String(),
			"signature":          toHex(cert.Signature),
			"publicKeyAlgorithm": cert.PublicKeyAlgorithm.String(),
			// This needs to be a string as the number could be way too large to
			// fit in JSON or Protobuf
			"serialNumber":     toHex(cert.SerialNumber.Bytes()),
			"keyUsage":         getKeyUsage(cert.KeyUsage),
			"extendedKeyUsage": getExtendedKeyUsage(cert.ExtKeyUsage),
			"version":          cert.Version,
			"basicConstraints": map[string]interface{}{
				"CA":      cert.IsCA,
				"pathLen": cert.MaxPathLen,
			},
			"subjectKeyIdentifier":   toHex(cert.SubjectKeyId),
			"authorityKeyIdentifier": toHex(cert.AuthorityKeyId),
		})

		if err != nil {
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_OTHER,
				ErrorString: err.Error(),
				Scope:       scope,
			}
		}

		if len(cert.OCSPServer) > 0 {
			attributes.Set("ocspServer", strings.Join(cert.OCSPServer, ","))
		}

		if len(cert.IssuingCertificateURL) > 0 {
			attributes.Set("issuingCertificateURL", strings.Join(cert.IssuingCertificateURL, ","))
		}

		if len(cert.CRLDistributionPoints) > 0 {
			attributes.Set("CRLDistributionPoints", cert.CRLDistributionPoints)
		}

		if len(cert.DNSNames) > 0 {
			attributes.Set("dnsNames", cert.DNSNames)
		}

		if len(cert.IPAddresses) > 0 {
			attributes.Set("ipAddresses", cert.IPAddresses)
		}

		if len(cert.URIs) > 0 {
			attributes.Set("uris", cert.URIs)
		}

		if len(cert.PermittedDNSDomains) > 0 {
			attributes.Set("permittedDNSDomains", cert.PermittedDNSDomains)
		}

		if len(cert.ExcludedDNSDomains) > 0 {
			attributes.Set("excludedDNSDomains", cert.ExcludedDNSDomains)
		}

		if len(cert.PermittedIPRanges) > 0 {
			attributes.Set("permittedIPRanges", cert.PermittedIPRanges)
		}

		if len(cert.ExcludedIPRanges) > 0 {
			attributes.Set("excludedIPRanges", cert.ExcludedIPRanges)
		}

		if len(cert.PermittedEmailAddresses) > 0 {
			attributes.Set("permittedEmailAddresses", cert.PermittedEmailAddresses)
		}

		if len(cert.ExcludedEmailAddresses) > 0 {
			attributes.Set("excludedEmailAddresses", cert.ExcludedEmailAddresses)
		}

		if len(cert.PermittedURIDomains) > 0 {
			attributes.Set("permittedURIDomains", cert.PermittedURIDomains)
		}

		if len(cert.ExcludedURIDomains) > 0 {
			attributes.Set("excludedURIDomains", cert.ExcludedURIDomains)
		}

		if len(cert.PolicyIdentifiers) > 0 {
			objectIdentifiers := make([]string, len(cert.PolicyIdentifiers))

			for i := range len(cert.PolicyIdentifiers) {
				objectIdentifiers[i] = cert.PolicyIdentifiers[i].String()
			}

			attributes.Set("policyIdentifiers", objectIdentifiers)
		}

		item := sdp.Item{
			Type:            "certificate",
			UniqueAttribute: "subject",
			Attributes:      attributes,
			Scope:           scope,
		}

		items = append(items, &item)

		// If not self signed, add a link to the issuer
		if cert.Issuer.String() != cert.Subject.String() {
			// Even though this adapter doesn't support Get() requests, this will
			// still work for linking as long as the referenced cert has been
			// included in the bundle since the cache will correctly return the
			// Get() request when it is run
			item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "certificate",
					Method: sdp.QueryMethod_GET,
					Query:  cert.Issuer.String(),
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Changing issuer will affect the child
					In: true,
					// The child can't affect the issuer
					Out: false,
				},
			})
		}
	}

	// If all failed return an error
	if len(errors) == len(bundle.Certificate) {
		return items, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf("parsing all certs failed, errors: %v", errors),
			Scope:       scope,
		}
	}

	return items, nil
}

func decodePem(certInput string) (tls.Certificate, error) {
	var bundle tls.Certificate
	certPEMBlock := []byte(certInput)
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			bundle.Certificate = append(bundle.Certificate, certDERBlock.Bytes)
		}
	}

	if len(bundle.Certificate) == 0 {
		return bundle, errors.New("no certificates could be parsed")
	}

	return bundle, nil
}

// Weight Returns the priority weighting of items returned by this adapter.
// This is used to resolve conflicts where two adapters of the same type
// return an item for a GET request. In this instance only one item can be
// sen on, so the one with the higher weight value will win.
func (s *CertificateAdapter) Weight() int {
	return 100
}

// getKeyUsage Converts the key usage from an integer to an array of valid
// usages. This is done by using a bitwise and to cover the binary number to the
// usage based on its mask e.g. 000010010 (18) would be ContentCommitment and
// KeyAgreement
func getKeyUsage(usage x509.KeyUsage) []string {
	usageStrings := make([]string, 0)

	// Uses the same string values as openssl's
	// https://github.com/openssl/openssl/blob/1c0eede9827b0962f1d752fa4ab5d436fa039da4/crypto/x509/v3_bitst.c#L28-L39
	if (usage & x509.KeyUsageDigitalSignature) == x509.KeyUsageDigitalSignature {
		usageStrings = append(usageStrings, "Digital Signature")
	}
	if (usage & x509.KeyUsageContentCommitment) == x509.KeyUsageContentCommitment {
		usageStrings = append(usageStrings, "Non Repudiation")
	}
	if (usage & x509.KeyUsageKeyEncipherment) == x509.KeyUsageKeyEncipherment {
		usageStrings = append(usageStrings, "Key Encipherment")
	}
	if (usage & x509.KeyUsageDataEncipherment) == x509.KeyUsageDataEncipherment {
		usageStrings = append(usageStrings, "Data Encipherment")
	}
	if (usage & x509.KeyUsageKeyAgreement) == x509.KeyUsageKeyAgreement {
		usageStrings = append(usageStrings, "Key Agreement")
	}
	if (usage & x509.KeyUsageCertSign) == x509.KeyUsageCertSign {
		usageStrings = append(usageStrings, "Certificate Sign")
	}
	if (usage & x509.KeyUsageCRLSign) == x509.KeyUsageCRLSign {
		usageStrings = append(usageStrings, "CRL Sign")
	}
	if (usage & x509.KeyUsageEncipherOnly) == x509.KeyUsageEncipherOnly {
		usageStrings = append(usageStrings, "Encipher Only")
	}
	if (usage & x509.KeyUsageDecipherOnly) == x509.KeyUsageDecipherOnly {
		usageStrings = append(usageStrings, "Decipher Only")
	}

	return usageStrings
}

// getExtendedKeyUsage Gets the list of extended usage, using the same working
// as openssl does as much as possible
//
// See:
// https://github.com/openssl/openssl/blob/b0c1214e1e82bc4c98eadd11d368b4ba9ffa202c/crypto/objects/obj_dat.h
func getExtendedKeyUsage(usage []x509.ExtKeyUsage) []string {
	usageStrings := make([]string, 0)

	for _, use := range usage {
		switch use {
		case x509.ExtKeyUsageAny:
			usageStrings = append(usageStrings, "Any Extended Key Usage")
		case x509.ExtKeyUsageServerAuth:
			usageStrings = append(usageStrings, "TLS Web Server Authentication")
		case x509.ExtKeyUsageClientAuth:
			usageStrings = append(usageStrings, "TLS Web Client Authentication")
		case x509.ExtKeyUsageCodeSigning:
			usageStrings = append(usageStrings, "Code Signing")
		case x509.ExtKeyUsageEmailProtection:
			usageStrings = append(usageStrings, "E-mail Protection")
		case x509.ExtKeyUsageIPSECEndSystem:
			usageStrings = append(usageStrings, "IPSec End System")
		case x509.ExtKeyUsageIPSECTunnel:
			usageStrings = append(usageStrings, "IPSec Tunnel")
		case x509.ExtKeyUsageIPSECUser:
			usageStrings = append(usageStrings, "IPSec User")
		case x509.ExtKeyUsageTimeStamping:
			usageStrings = append(usageStrings, "Time Stamping")
		case x509.ExtKeyUsageOCSPSigning:
			usageStrings = append(usageStrings, "OCSP Signing")
		case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
			usageStrings = append(usageStrings, "Microsoft Server Gated Crypto")
		case x509.ExtKeyUsageNetscapeServerGatedCrypto:
			usageStrings = append(usageStrings, "Netscape Server Gated Crypto")
		case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
			usageStrings = append(usageStrings, "Microsoft Commercial Code Signing")
		case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
			usageStrings = append(usageStrings, "Kernel Mode Code Signing")
		default:
			usageStrings = append(usageStrings, fmt.Sprint(use))
		}
	}

	return usageStrings
}
