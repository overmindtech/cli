package adapters

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/miekg/dns"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// DNSAdapter struct on which all methods are registered
type DNSAdapter struct {
	// List of DNS server to use in order ot preference. They should be in the
	// format "ip:port"
	Servers []string

	// Whether to perform reverse lookups on IP addresses
	ReverseLookup bool

	client dns.Client

	cache       *sdpcache.Cache // The sdpcache of this adapter
	cacheInitMu sync.Mutex      // Mutex to ensure cache is only initialised once
}

const dnsCacheDuration = 5 * time.Minute

func (s *DNSAdapter) ensureCache() {
	s.cacheInitMu.Lock()
	defer s.cacheInitMu.Unlock()

	if s.cache == nil {
		s.cache = sdpcache.NewCache()
	}
}

func (s *DNSAdapter) Cache() *sdpcache.Cache {
	s.ensureCache()
	return s.cache
}

var DefaultServers = []string{
	"169.254.169.253:53", // Route 53 default resolver. See https://docs.aws.amazon.com/vpc/latest/userguide/AmazonDNS-concepts.html#AmazonDNS
	"1.1.1.1:53",
	"8.8.8.8:53",
	"8.8.4.4:53",
}

const ItemType = "dns"
const UniqueAttribute = "name"

var ErrNoServersAvailable = errors.New("no dns servers available")

// Type is the type of items that this returns
func (d *DNSAdapter) Type() string {
	return "dns"
}

// Name Returns the name of the backend
func (d *DNSAdapter) Name() string {
	return "stdlib-dns"
}

// Weighting of duplicate adapters
func (d *DNSAdapter) Weight() int {
	return 100
}

func (d *DNSAdapter) GetServers() []string {
	if len(d.Servers) == 0 {
		return DefaultServers
	}
	return d.Servers
}

func (d *DNSAdapter) Metadata() *sdp.AdapterMetadata {
	return dnsMetadata
}

var dnsMetadata = Metadata.Register(&sdp.AdapterMetadata{
	DescriptiveName: "DNS Entry",
	Type:            "dns",
	SupportedQueryMethods: &sdp.AdapterSupportedQueryMethods{
		Get:               true,
		Search:            true,
		GetDescription:    "A DNS A or AAAA entry to look up",
		SearchDescription: "A DNS name (or IP for reverse DNS), this will perform a recursive search and return all results. It is recommended that you always use the SEARCH method",
	},
	PotentialLinks: []string{"dns", "ip", "rdap-domain"},
	Category:       sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
})

// List of scopes that this adapter is capable of find items for
func (d *DNSAdapter) Scopes() []string {
	return []string{
		// DNS entries *should* be globally unique
		"global",
	}
}

// Gets a single item. This expects a DNS name
func (d *DNSAdapter) Get(ctx context.Context, scope string, query string, ignoreCache bool) (*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "DNS queries only supported in global scope",
			Scope:       scope,
		}
	}

	// Check for IP addresses and do nothing
	if net.ParseIP(query) != nil {
		return &sdp.Item{}, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOTFOUND,
			ErrorString: fmt.Sprintf("%v is already an IP address, no DNS entry will be found", query),
		}
	}

	d.ensureCache()
	cacheHit, ck, cachedItems, qErr := d.cache.Lookup(ctx, d.Name(), sdp.QueryMethod_GET, scope, d.Type(), query, ignoreCache)
	if qErr != nil {
		return nil, qErr
	}
	if cacheHit {
		if len(cachedItems) > 0 {
			return cachedItems[0], nil
		} else {
			return nil, nil
		}
	}

	// This won't work for CNAMEs since the linked query logic needs to be
	// different and we're only querying for A and AAAA. Realistically people
	// should be using Search() now anyway
	items, err := d.MakeQuery(ctx, query)

	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     "global",
		}
	}

	d.cache.StoreItem(items[0], dnsCacheDuration, ck)
	return items[0], nil
}

// List calls back to the ListFunction to find all items
func (d *DNSAdapter) List(ctx context.Context, scope string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "DNS queries only supported in global scope",
			Scope:       scope,
		}
	}

	return make([]*sdp.Item, 0), nil
}

type DNSRecord struct {
	Name   string
	Target string
	Type   string
}

func (d *DNSAdapter) Search(ctx context.Context, scope string, query string, ignoreCache bool) ([]*sdp.Item, error) {
	if scope != "global" {
		return nil, &sdp.QueryError{
			ErrorType:   sdp.QueryError_NOSCOPE,
			ErrorString: "DNS queries only supported in global scope",
			Scope:       scope,
		}
	}

	if net.ParseIP(query) != nil {
		if d.ReverseLookup {
			// If it's an IP then we want to run a reverse lookup
			return d.MakeReverseQuery(ctx, query)
		} else {
			// If disabled, return nothing
			return []*sdp.Item{}, nil
		}
	}

	ck := sdpcache.CacheKeyFromParts(d.Name(), sdp.QueryMethod_SEARCH, scope, d.Type(), query)

	items, err := d.MakeQuery(ctx, query)
	if err != nil {
		d.cache.StoreError(err, dnsCacheDuration, ck)
		return nil, err
	}

	for _, item := range items {
		d.cache.StoreItem(item, dnsCacheDuration, ck)
	}

	return items, nil
}

// retryDNSQuery handles retrying DNS queries with backoff and server rotation
func (d *DNSAdapter) retryDNSQuery(ctx context.Context, queryFn func(context.Context, string) ([]*sdp.Item, error)) ([]*sdp.Item, error) {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 100 * time.Millisecond
	b.MaxInterval = 500 * time.Millisecond
	b.MaxElapsedTime = 30 * time.Second

	var items []*sdp.Item
	var i int
	var server string

	operation := func() error {
		if i >= len(d.GetServers()) {
			i = 0
		}

		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		server = d.GetServers()[i]

		var err error
		items, err = queryFn(ctx, server)
		if err != nil {
			i++ // Move to next server on error

			if errors.Is(err, context.DeadlineExceeded) ||
				strings.Contains(err.Error(), "timeout") ||
				strings.Contains(err.Error(), "temporary failure") {
				return err // Retry on timeout
			}
			return backoff.Permanent(err)
		}

		return nil
	}

	err := backoff.Retry(operation, b)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("ovm.dns.server", server),
	)
	if err != nil {
		return nil, err
	}

	return items, nil
}

// Updated MakeQuery
func (d *DNSAdapter) MakeQuery(ctx context.Context, query string) ([]*sdp.Item, error) {
	return d.retryDNSQuery(ctx, func(ctx context.Context, server string) ([]*sdp.Item, error) {
		return d.makeQueryImpl(ctx, query, server)
	})
}

// Updated MakeReverseQuery
func (d *DNSAdapter) MakeReverseQuery(ctx context.Context, query string) ([]*sdp.Item, error) {
	return d.retryDNSQuery(ctx, func(ctx context.Context, server string) ([]*sdp.Item, error) {
		return d.makeReverseQueryImpl(ctx, query, server)
	})
}

func (d *DNSAdapter) makeReverseQueryImpl(ctx context.Context, query string, server string) ([]*sdp.Item, error) {
	arpa, err := dns.ReverseAddr(query)

	if err != nil {
		return nil, err
	}

	// Create the query
	msg := dns.Msg{
		Question: []dns.Question{
			{
				Name:   arpa,
				Qclass: dns.ClassINET,
				Qtype:  dns.TypePTR,
			},
		},
		MsgHdr: dns.MsgHdr{
			Opcode:           dns.OpcodeQuery,
			RecursionDesired: true,
		},
	}

	r, _, err := d.client.ExchangeContext(ctx, &msg, server)

	if err != nil {
		return nil, err
	}

	items := make([]*sdp.Item, 0)

	for _, rr := range r.Answer {
		if ptr, ok := rr.(*dns.PTR); ok {
			newItems, err := d.MakeQuery(ctx, ptr.Ptr)

			if err != nil {
				return nil, err
			}

			items = append(items, newItems...)
		}
	}

	return items, nil
}

// trimDnsSuffix Trims the trailing dot from a name to make it more user friendly
func trimDnsSuffix(name string) string {
	if strings.HasSuffix(name, ".") {
		return name[:len(name)-1]
	}

	return name
}

func (d *DNSAdapter) makeQueryImpl(ctx context.Context, query string, server string) ([]*sdp.Item, error) {
	// Create the query
	msg := dns.Msg{
		Question: []dns.Question{
			{
				Name:   dns.Fqdn(query),
				Qclass: dns.ClassINET,
				Qtype:  dns.TypeA,
			},
		},
		MsgHdr: dns.MsgHdr{
			Opcode:           dns.OpcodeQuery,
			RecursionDesired: true,
		},
	}

	r, _, err := d.client.ExchangeContext(ctx, &msg, server)

	if err != nil {
		return nil, err
	}

	// Also query for AAAA
	msg.Question[0].Qtype = dns.TypeAAAA
	r2, _, err := d.client.ExchangeContext(ctx, &msg, server)

	if err != nil {
		return nil, err
	}

	answers := make([]dns.RR, 0)
	answers = append(answers, r.Answer...)
	answers = append(answers, r2.Answer...)

	if len(answers) == 0 {
		// This means nothing was found
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_NOTFOUND,
			Scope:     "global",
		}
	}

	ag := GroupAnswers(answers)

	items := make([]*sdp.Item, 0)

	var item *sdp.Item
	var attrs *sdp.ItemAttributes

	// Iterate over the groups and convert
	for _, r := range ag.CNAME {
		if cname, ok := r.(*dns.CNAME); ok {
			// Strip trailing dot as while it's *technically* correct, it's
			// annoying to have to deal with
			name := trimDnsSuffix(cname.Hdr.Name)
			target := trimDnsSuffix(cname.Target)

			attrs, err = sdp.ToAttributes(map[string]interface{}{
				"name":   name,
				"type":   "CNAME",
				"ttl":    cname.Hdr.Ttl,
				"target": target,
			})

			if err != nil {
				return nil, err
			}

			item = &sdp.Item{
				Type:            ItemType,
				UniqueAttribute: UniqueAttribute,
				Scope:           "global",
				Attributes:      attrs,
				// TODO(LIQs): update this method to return the data as edges; fixup all callers
				LinkedItems: []*sdp.LinkedItem{
					{
						Item: &sdp.Reference{
							Type:                 ItemType,
							UniqueAttributeValue: target,
							Scope:                "global",
						},
					},
				},
				LinkedItemQueries: []*sdp.LinkedItemQuery{
					{
						Query: &sdp.Query{
							Type:   "rdap-domain",
							Method: sdp.QueryMethod_SEARCH,
							Query:  name,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true,
							Out: false,
						},
					},
				},
			}

			items = append(items, item)
		}
	}

	// Convert A & AAAA records by group
	for name, rs := range ag.Address {
		// Strip trailing dot as while it's *technically* correct, it's
		// annoying to have to deal with
		name = trimDnsSuffix(name)

		item, err := AToItem(name, rs)

		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	return items, nil
}

type AnswerGroup struct {
	CNAME   map[string]dns.RR
	Address map[string][]dns.RR
}

// GroupAnswers Groups the DNS answers so they they can be turned into
// individual items. This is required because some types (such as A records) can
// return man records for the same name and these need to be grouped to avoid
// duplicate items
func GroupAnswers(answers []dns.RR) *AnswerGroup {
	ag := AnswerGroup{
		CNAME:   make(map[string]dns.RR),
		Address: make(map[string][]dns.RR),
	}

	for _, answer := range answers {
		if hdr := answer.Header(); hdr != nil {
			switch hdr.Rrtype {
			case dns.TypeCNAME:
				// We should only get one CNAME per request, but since we have
				// done A and AAAA requests we could have duplicates, use a map
				// to avoid this
				ag.CNAME[hdr.Name] = answer
			case dns.TypeA, dns.TypeAAAA:
				// Create the map entry if required
				if _, ok := ag.Address[hdr.Name]; !ok {
					ag.Address[hdr.Name] = make([]dns.RR, 0)
				}

				ag.Address[hdr.Name] = append(ag.Address[hdr.Name], answer)
			}
		}
	}

	return &ag
}

// AToItem Converts a set of A or AAAA records to an item
func AToItem(name string, records []dns.RR) (*sdp.Item, error) {
	recordAttrs := make([]map[string]interface{}, 0)
	liq := make([]*sdp.LinkedItemQuery, 0)

	for _, r := range records {
		if hdr := r.Header(); hdr != nil {
			var ip net.IP
			var typ string

			if a, ok := r.(*dns.A); ok {
				typ = "A"
				ip = a.A
			} else if aaaa, ok := r.(*dns.AAAA); ok {
				typ = "AAAA"
				ip = aaaa.AAAA
			}

			recordAttrs = append(recordAttrs, map[string]interface{}{
				"ttl":  hdr.Ttl,
				"type": typ,
				"ip":   ip.String(),
			})

			liq = append(liq, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   "ip",
					Method: sdp.QueryMethod_GET,
					Query:  ip.String(),
					Scope:  "global",
				},
				BlastPropagation: &sdp.BlastPropagation{
					// Tightly coupled
					In:  true,
					Out: true,
				},
			})
		}
	}

	// Sort records to ensure they are consistent
	sort.Slice(recordAttrs, func(i, j int) bool {
		return fmt.Sprint(i) < fmt.Sprint(j)
	})

	attrs, err := sdp.ToAttributes(map[string]interface{}{
		"name":    name,
		"type":    "address",
		"records": recordAttrs,
	})

	if err != nil {
		return nil, err
	}

	item := sdp.Item{
		Type:              ItemType,
		UniqueAttribute:   UniqueAttribute,
		Scope:             "global",
		Attributes:        attrs,
		LinkedItemQueries: liq,
	}

	item.LinkedItemQueries = append(item.LinkedItemQueries, &sdp.LinkedItemQuery{
		Query: &sdp.Query{
			Type:   "rdap-domain",
			Method: sdp.QueryMethod_SEARCH,
			Query:  name,
			Scope:  "global",
		},
		BlastPropagation: &sdp.BlastPropagation{
			// Changes to the domain will affect the DNS entry
			In: true,
			// Changes to the DNS entry won't affect the domain
			Out: false,
		},
	})

	return &item, nil
}
