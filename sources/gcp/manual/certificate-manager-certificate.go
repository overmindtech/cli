package manual

import (
	"context"
	"errors"

	certificatemanagerpb "cloud.google.com/go/certificatemanager/apiv1/certificatemanagerpb"
	"google.golang.org/api/iterator"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	"github.com/overmindtech/cli/sources"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var (
	CertificateManagerCertificateLookupByLocation = shared.NewItemTypeLookup("location", gcpshared.CertificateManagerCertificate)
	CertificateManagerCertificateLookupByName     = shared.NewItemTypeLookup("name", gcpshared.CertificateManagerCertificate)
)

type certificateManagerCertificateWrapper struct {
	client gcpshared.CertificateManagerCertificateClient
	*gcpshared.ProjectBase
}

// NewCertificateManagerCertificate creates a new certificateManagerCertificateWrapper.
func NewCertificateManagerCertificate(client gcpshared.CertificateManagerCertificateClient, locations []gcpshared.LocationInfo) sources.SearchableWrapper {
	return &certificateManagerCertificateWrapper{
		client: client,
		ProjectBase: gcpshared.NewProjectBase(
			locations,
			sdp.AdapterCategory_ADAPTER_CATEGORY_SECURITY,
			gcpshared.CertificateManagerCertificate,
		),
	}
}

func (c certificateManagerCertificateWrapper) IAMPermissions() []string {
	return []string{
		"certificatemanager.certs.get",
		"certificatemanager.certs.list",
	}
}

func (c certificateManagerCertificateWrapper) PredefinedRole() string {
	return "roles/certificatemanager.viewer"
}

func (c certificateManagerCertificateWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		gcpshared.CertificateManagerDnsAuthorization,
		gcpshared.CertificateManagerCertificateIssuanceConfig,
		stdlib.NetworkDNS,
	)
}

func (c certificateManagerCertificateWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod: sdp.QueryMethod_SEARCH,
			// https://registry.terraform.io/providers/hashicorp/google/latest/docs/resources/certificate_manager_certificate
			// ID format: projects/{{project}}/locations/{{location}}/certificates/{{name}}
			// The framework automatically intercepts queries starting with "projects/" and converts
			// them to GET operations by extracting the last N path parameters (based on GetLookups count).
			TerraformQueryMap: "google_certificate_manager_certificate.id",
		},
	}
}

func (c certificateManagerCertificateWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		CertificateManagerCertificateLookupByLocation,
		CertificateManagerCertificateLookupByName,
	}
}

// Get retrieves a Certificate Manager Certificate by its unique attribute (location|certificateName).
func (c certificateManagerCertificateWrapper) Get(ctx context.Context, scope string, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		}
	}

	if len(queryParts) != 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Get requires exactly 2 query parts: location and certificate name",
		}
	}

	locationName := queryParts[0]
	certificateName := queryParts[1]

	// Construct the full resource name
	// Format: projects/{project}/locations/{location}/certificates/{certificate}
	name := "projects/" + location.ProjectID + "/locations/" + locationName + "/certificates/" + certificateName

	req := &certificatemanagerpb.GetCertificateRequest{
		Name: name,
	}

	certificate, getErr := c.client.GetCertificate(ctx, req)
	if getErr != nil {
		return nil, gcpshared.QueryError(getErr, scope, c.Type())
	}

	item, sdpErr := c.gcpCertificateToSDPItem(certificate, location)
	if sdpErr != nil {
		return nil, sdpErr
	}

	return item, nil
}

func (c certificateManagerCertificateWrapper) SearchLookups() []sources.ItemTypeLookups {
	return []sources.ItemTypeLookups{
		{
			CertificateManagerCertificateLookupByLocation,
		},
	}
}

// Search searches Certificate Manager Certificates by location.
func (c certificateManagerCertificateWrapper) Search(ctx context.Context, scope string, queryParts ...string) ([]*sdp.Item, *sdp.QueryError) {
	return gcpshared.CollectFromStream(ctx, func(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
		c.SearchStream(ctx, stream, cache, cacheKey, scope, queryParts...)
	})
}

// SearchStream streams certificates matching the search criteria (location).
func (c certificateManagerCertificateWrapper) SearchStream(ctx context.Context, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey, scope string, queryParts ...string) {
	location, err := c.LocationFromScope(scope)
	if err != nil {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: err.Error(),
		})
		return
	}

	if len(queryParts) != 1 {
		stream.SendError(&sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "Search requires 1 query part: location",
		})
		return
	}

	locationName := queryParts[0]

	// Construct the parent path
	// Format: projects/{project}/locations/{location}
	parent := "projects/" + location.ProjectID + "/locations/" + locationName

	req := &certificatemanagerpb.ListCertificatesRequest{
		Parent: parent,
	}

	results := c.client.ListCertificates(ctx, req)

	for {
		cert, iterErr := results.Next()
		if errors.Is(iterErr, iterator.Done) {
			break
		}
		if iterErr != nil {
			stream.SendError(gcpshared.QueryError(iterErr, scope, c.Type()))
			return
		}

		item, sdpErr := c.gcpCertificateToSDPItem(cert, location)
		if sdpErr != nil {
			stream.SendError(sdpErr)
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		stream.SendItem(item)
	}
}

func (c certificateManagerCertificateWrapper) gcpCertificateToSDPItem(certificate *certificatemanagerpb.Certificate, location gcpshared.LocationInfo) (*sdp.Item, *sdp.QueryError) {
	attributes, err := shared.ToAttributesWithExclude(certificate, "labels")
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	// Extract location and certificate name from the resource name
	// Format: projects/{project}/locations/{location}/certificates/{certificate}
	values := gcpshared.ExtractPathParams(certificate.GetName(), "locations", "certificates")
	if len(values) != 2 {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: "invalid certificate name format: " + certificate.GetName(),
		}
	}

	locationName := values[0]
	certificateName := values[1]

	// Set composite unique attribute
	err = attributes.Set("uniqueAttr", shared.CompositeLookupKey(locationName, certificateName))
	if err != nil {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_OTHER,
			ErrorString: err.Error(),
		}
	}

	sdpItem := &sdp.Item{
		Type:            gcpshared.CertificateManagerCertificate.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           location.ToScope(),
		Tags:            certificate.GetLabels(),
	}

	// Link to DNS names from sanDnsNames (covers both managed and self-managed certificates)
	for _, dnsName := range certificate.GetSanDnsnames() {
		if dnsName != "" {
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   stdlib.NetworkDNS.String(),
					Method: sdp.QueryMethod_SEARCH,
					Query:  dnsName,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Certificate depends on DNS resolution
					// DNS changes affect certificate validity
					In:  true,
					Out: true,
				},
			})
		}
	}

	// Link to DNS Authorizations used for managed certificate domain validation
	if managed := certificate.GetManaged(); managed != nil {
		// Link to DNS names from managed.domains
		for _, domain := range managed.GetDomains() {
			if domain != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  domain,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Certificate depends on DNS resolution for domain validation
						// DNS changes affect certificate provisioning
						In:  true,
						Out: true,
					},
				})
			}
		}
		for _, dnsAuthURI := range managed.GetDnsAuthorizations() {
			// Extract location and dnsAuthorization name from URI
			// Format: projects/{project}/locations/{location}/dnsAuthorizations/{dnsAuthorization}
			values := gcpshared.ExtractPathParams(dnsAuthURI, "locations", "dnsAuthorizations")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CertificateManagerDnsAuthorization.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(values[0], values[1]),
						Scope:  location.ProjectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Certificate depends on DNS authorization for domain validation
						// If DNS authorization is deleted, certificate provisioning fails
						// Deleting certificate doesn't affect the DNS authorization
						In:  true,
						Out: false,
					},
				})
			}
		}

		// Link to Certificate Issuance Config for private PKI certificates
		if issuanceConfigURI := managed.GetIssuanceConfig(); issuanceConfigURI != "" {
			// Extract location and issuanceConfig name from URI
			// Format: projects/{project}/locations/{location}/certificateIssuanceConfigs/{certificateIssuanceConfig}
			values := gcpshared.ExtractPathParams(issuanceConfigURI, "locations", "certificateIssuanceConfigs")
			if len(values) == 2 && values[0] != "" && values[1] != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   gcpshared.CertificateManagerCertificateIssuanceConfig.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(values[0], values[1]),
						Scope:  location.ProjectID,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Certificate depends on issuance config for private PKI
						// If issuance config is deleted, certificate provisioning fails
						// Deleting certificate doesn't affect the issuance config
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	// Note: The Certificate resource's UsedBy field (which lists resources using this certificate)
	// is not available in the Go SDK protobuf. The reverse links from CertificateMap,
	// CertificateMapEntry, and TargetHttpsProxy to Certificate will be established
	// when those adapters are created.

	return sdpItem, nil
}
