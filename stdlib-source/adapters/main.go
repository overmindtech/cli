package adapters

import (
	"context"
	"errors"
	"net/url"
	"reflect"
	"regexp"
	"time"

	"github.com/openrdap/rdap"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/stdlib-source/adapters/test"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	_ "embed"
)

var Metadata = sdp.AdapterMetadataList{}

// Cache duration for RDAP adapters, these things shouldn't change very often
const RdapCacheDuration = 30 * time.Minute

func InitializeEngine(ec *discovery.EngineConfig, reverseDNS bool) (*discovery.Engine, error) {
	e, err := discovery.NewEngine(ec)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Fatal("Error initializing Engine")
	}

	if ec.HeartbeatOptions != nil {
		ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
			// This can't fail, it's always healthy
			return nil
		}
	}

	// Add the base adapters
	adapters := []discovery.Adapter{
		&CertificateAdapter{},
		&DNSAdapter{
			ReverseLookup: reverseDNS,
		},
		&HTTPAdapter{},
		&IPAdapter{},
		&test.TestDogAdapter{},
		&test.TestFoodAdapter{},
		&test.TestGroupAdapter{},
		&test.TestHobbyAdapter{},
		&test.TestLocationAdapter{},
		&test.TestPersonAdapter{},
		&test.TestRegionAdapter{},
		&RdapIPNetworkAdapter{
			ClientFac: newRdapClient,
			Cache:     sdpcache.NewCache(),
			IPCache:   NewIPCache[*rdap.IPNetwork](),
		},
		&RdapASNAdapter{
			ClientFac: newRdapClient,
			Cache:     sdpcache.NewCache(),
		},
		&RdapDomainAdapter{
			ClientFac: newRdapClient,
			Cache:     sdpcache.NewCache(),
		},
		&RdapEntityAdapter{
			ClientFac: newRdapClient,
			Cache:     sdpcache.NewCache(),
		},
		&RdapNameserverAdapter{
			ClientFac: newRdapClient,
			Cache:     sdpcache.NewCache(),
		},
	}

	err = e.AddAdapters(adapters...)

	return e, err
}

// newRdapClient Creates a new RDAP client using otelhttp.DefaultClient. rdap is suspected to not be thread safe, so we create a new client for each request
func newRdapClient() *rdap.Client {
	return &rdap.Client{
		HTTP: otelhttp.DefaultClient,
	}
}

// Wraps an RDAP error in an SDP error, correctly checking for things like 404s
func wrapRdapError(err error) error {
	if err == nil {
		return nil
	}

	var rdapError *rdap.ClientError

	if ok := errors.As(err, &rdapError); ok {
		if rdapError.Type == rdap.ObjectDoesNotExist {
			return &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: err.Error(),
			}
		}
	}

	return err
}

// Extracts SDP queries from a list of entities
func extractEntityLinks(entities []rdap.Entity) []*sdp.LinkedItemQuery {
	queries := make([]*sdp.LinkedItemQuery, 0)

	for _, entity := range entities {
		var selfLink string

		// Loop over the links until you find the self link
		for _, link := range entity.Links {
			if link.Rel == "self" {
				selfLink = link.Href
				break
			}
		}

		if selfLink != "" {
			queries = append(queries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "rdap-entity",
					Method: sdp.QueryMethod_SEARCH,
					Query:  selfLink,
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// The Entity isn't a "real" component, so no matter what
					// changes it won't actually "affect" anything
					In:  false,
					Out: false,
				},
			})
		}
	}

	return queries
}

var rdapUrlRegex = regexp.MustCompile(`^(https?:\/\/.+)\/(ip|nameserver|entity|autnum|domain)\/([^\/]+)$`)

type RDAPUrl struct {
	// The path to the root where queries should be run i.e.
	// https://rdap.apnic.net
	ServerRoot *url.URL
	// The type of query to run i.e. ip, nameserver, entity, autnum, domain
	Type string
	// The query to run i.e. 1.1.1.1
	Query string
}

// Parses an RDAP URL and returns the important components
func parseRdapUrl(rdapUrl string) (*RDAPUrl, error) {
	matches := rdapUrlRegex.FindStringSubmatch(rdapUrl)

	if len(matches) != 4 {
		return nil, errors.New("Invalid RDAP URL")
	}

	serverRoot, err := url.Parse(matches[1])

	if err != nil {
		return nil, err
	}

	return &RDAPUrl{
		ServerRoot: serverRoot,
		Type:       matches[2],
		Query:      matches[3],
	}, nil
}

var RDAPTransforms = sdp.AddDefaultTransforms(sdp.TransformMap{
	reflect.TypeOf(rdap.Link{}): func(i interface{}) interface{} {
		// We only want to return the href for links
		link, ok := i.(rdap.Link)

		if ok {
			return link.Href
		}

		return ""
	},
	reflect.TypeOf(rdap.VCard{}): func(i interface{}) interface{} {
		vcard, ok := i.(rdap.VCard)

		if ok {
			// Convert a vCard to a map as it's much more readable
			vCardDetails := make(map[string]string)

			if name := vcard.Name(); name != "" {
				vCardDetails["Name"] = name
			}
			if pOBox := vcard.POBox(); pOBox != "" {
				vCardDetails["POBox"] = pOBox
			}
			if extendedAddress := vcard.ExtendedAddress(); extendedAddress != "" {
				vCardDetails["ExtendedAddress"] = extendedAddress
			}
			if streetAddress := vcard.StreetAddress(); streetAddress != "" {
				vCardDetails["StreetAddress"] = streetAddress
			}
			if locality := vcard.Locality(); locality != "" {
				vCardDetails["Locality"] = locality
			}
			if region := vcard.Region(); region != "" {
				vCardDetails["Region"] = region
			}
			if postalCode := vcard.PostalCode(); postalCode != "" {
				vCardDetails["PostalCode"] = postalCode
			}
			if country := vcard.Country(); country != "" {
				vCardDetails["Country"] = country
			}
			if tel := vcard.Tel(); tel != "" {
				vCardDetails["Tel"] = tel
			}
			if fax := vcard.Fax(); fax != "" {
				vCardDetails["Fax"] = fax
			}
			if email := vcard.Email(); email != "" {
				vCardDetails["Email"] = email
			}
			if org := vcard.Org(); org != "" {
				vCardDetails["Org"] = org
			}

			return vCardDetails
		}

		return nil
	},
	reflect.TypeOf(&rdap.DecodeData{}): func(i interface{}) interface{} {
		// Exclude these
		return nil
	},
})
