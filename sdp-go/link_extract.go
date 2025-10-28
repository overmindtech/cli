package sdp

import (
	"net"
	"net/url"
	"regexp"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// This function tries to extract linked item queries from the attributes of an
// item. It should be on items that we know are likely to contain references
// that we can discover, but are in an unstructured format which we can't
// construct the linked item queries from directly. A good example of this would
// be the env vars for a kubernetes pod, or a config map
//
// This supports extracting the following formats:
//
// - IP addresses
// - HTTP/HTTPS URLs
// - DNS names
func ExtractLinksFromAttributes(attributes *ItemAttributes) []*LinkedItemQuery {
	return extractLinksFromStructValue(attributes.GetAttrStruct())
}

// The same as `ExtractLinksFromAttributes`, but takes any input format and
// converts it to a set of ItemAttributes via the `ToAttributes` function. This
// uses reflection. `ExtractLinksFromAttributes` is more efficient if you have
// the attributes already in the correct format.
func ExtractLinksFrom(anything interface{}) ([]*LinkedItemQuery, error) {
	attributes, err := ToAttributes(map[string]interface{}{
		"": anything,
	})
	if err != nil {
		return nil, err
	}

	return ExtractLinksFromAttributes(attributes), nil
}

func extractLinksFromValue(value *structpb.Value) []*LinkedItemQuery {
	switch value.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil
	case *structpb.Value_NumberValue:
		return nil
	case *structpb.Value_StringValue:
		return extractLinksFromStringValue(value.GetStringValue())
	case *structpb.Value_BoolValue:
		return nil
	case *structpb.Value_StructValue:
		return extractLinksFromStructValue(value.GetStructValue())
	case *structpb.Value_ListValue:
		return extractLinksFromListValue(value.GetListValue())
	}

	return nil
}

func extractLinksFromStructValue(structValue *structpb.Struct) []*LinkedItemQuery {
	queries := make([]*LinkedItemQuery, 0)

	for _, value := range structValue.GetFields() {
		queries = append(queries, extractLinksFromValue(value)...)
	}

	return queries
}

func extractLinksFromListValue(list *structpb.ListValue) []*LinkedItemQuery {
	queries := make([]*LinkedItemQuery, 0)

	for _, value := range list.GetValues() {
		queries = append(queries, extractLinksFromValue(value)...)
	}

	return queries
}

// A regex that matches the ARN format and extracts the service, region, account
// id and resource. Uses a capture group for the full resource portion after
// the account-id (which may include slashes for resource types).
var awsARNRegex = regexp.MustCompile(`^arn:[\w-]+:([\w-]+):([\w-]*):([\w-]*):(.+)`)

// This function does all the heavy lifting for extracting linked item queries
// from strings. It will be called once for every string value in the item so
// needs to be very performant
func extractLinksFromStringValue(val string) []*LinkedItemQuery {
	if ip := net.ParseIP(val); ip != nil {
		return []*LinkedItemQuery{
			{
				Query: &Query{
					Type:   "ip",
					Method: QueryMethod_GET,
					Query:  ip.String(),
					Scope:  "global",
				},
				BlastPropagation: &BlastPropagation{
					In:  true,
					Out: true,
				},
			},
		}
	}

	// This is pretty overzealous when it comes to what it considers a URL, so
	// we need ot do out own validation to make sure that it has actually found
	// what we expected
	if parsed, err := url.Parse(val); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		// If it's a HTTP/HTTPS URL, we can use a HTTP query
		if parsed.Scheme == "http" || parsed.Scheme == "https" {
			return []*LinkedItemQuery{
				{
					Query: &Query{
						Type:   "http",
						Method: QueryMethod_SEARCH,
						Query:  val,
						Scope:  "global",
					},
					BlastPropagation: &BlastPropagation{
						// If we are referencing a HTTP URL, I think it's safe
						// to assume that this is something that the current
						// resource depends on and therefore that the blast
						// radius should propagate inwards. This is a bit of a
						// guess though...
						In:  true,
						Out: false,
					},
				},
			}
		}

		// If it's not a HTTP/HTTPS URL, it'll be an IP or DNS name, so pass
		// back to the main function
		return extractLinksFromStringValue(parsed.Hostname())
	}

	if isLikelyDNSName(val) {
		return []*LinkedItemQuery{
			{
				Query: &Query{
					Type:   "dns",
					Method: QueryMethod_SEARCH,
					Query:  val,
					Scope:  "global",
				},
				BlastPropagation: &BlastPropagation{
					In:  true,
					Out: false,
				},
			},
		}
	}

	// ARNs can't be shorter than 12 characters
	if len(val) >= 12 {
		if matches := awsARNRegex.FindStringSubmatch(val); matches != nil {
			// If it looks like an ARN then we can construct a SEARCH query to try
			// and find it. We can rely on the conventions in the AWS source here

			// Basic validation
			if len(matches) != 5 || matches[1] == "" {
				return nil
			}

			// Parsed ARN parts
			service := matches[1]   // e.g. "ec2", "iam", "s3"
			region := matches[2]    // may be empty for global services (iam, cloudfront)
			accountID := matches[3] // may be empty (e.g. s3, route53)
			resource := matches[4]  // full resource segment (may contain ":" or "/")

			// Extract resource type from the resource field (everything before first "/" or ":" if present)
			resourceType := resource
			if idx := strings.IndexAny(resource, "/:"); idx != -1 {
				resourceType = resource[:idx]
			}

			// Determine scope using a simple rule:
			// - No account → wildcard scope
			// - Account, no region → account-only
			// - Account and region → account.region
			var scope string
			if accountID == "" {
				scope = WILDCARD
			} else if region == "" {
				scope = accountID
			} else {
				scope = accountID + "." + region
			}

			// Determine type using a consistent rule. Default to service-resourceType if available.
			queryType := service
			if resourceType != "" {
				queryType = service + "-" + resourceType
			}
			// Special-case S3 ARNs that omit account and region → treat as bucket references
			if service == "s3" && accountID == "" && region == "" {
				queryType = "s3-bucket"

				// If this is an S3 object ARN (contains /), extract just the bucket
				if strings.Contains(resource, "/") {
					bucketName := strings.SplitN(resource, "/", 2)[0]
					// Construct a bucket-only ARN for the query
					val = "arn:aws:s3:::" + bucketName
				}
			}

			return []*LinkedItemQuery{
				{
					Query: &Query{
						Type:   queryType,
						Method: QueryMethod_SEARCH,
						Query:  val,
						Scope:  scope,
					},
					BlastPropagation: &BlastPropagation{
						In:  true,
						Out: false,
					},
				},
			}
		}
	}

	return nil
}

// Compile a regex pattern to match the general structure of a DNS name. Limits
// each label to 1-63 characters and matches only allowed characters and ensure
// that the name has at least three sections i.e. two dots.
var dnsNameRegex = regexp.MustCompile(`^(?i)([a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?\.){2,}[a-z]{2,}$`)

// This function returns true if the given string is a valid DNS name with at
// least three labels (sections)
func isLikelyDNSName(name string) bool {
	// Quick length check before the regex. The less than 6 is because we're
	// only matching names that have three sections or more, and the shortest
	// three section name is a.b.cd (6 characters, there are no single letter
	// top-level domains)
	if len(name) < 6 || len(name) > 253 {
		return false
	}

	// Check if the name matches the regex pattern.
	return dnsNameRegex.MatchString(name)
}
