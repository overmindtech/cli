package manual

import (
	"context"
	"errors"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	"github.com/overmindtech/cli/sources"
	"github.com/overmindtech/cli/sources/azure/clients"
	azureshared "github.com/overmindtech/cli/sources/azure/shared"
	"github.com/overmindtech/cli/sources/shared"
	"github.com/overmindtech/cli/sources/stdlib"
)

var NetworkApplicationGatewayLookupByName = shared.NewItemTypeLookup("name", azureshared.NetworkApplicationGateway)

type networkApplicationGatewayWrapper struct {
	client clients.ApplicationGatewaysClient

	*azureshared.ResourceGroupBase
}

func NewNetworkApplicationGateway(client clients.ApplicationGatewaysClient, subscriptionID, resourceGroup string) sources.ListableWrapper {
	return &networkApplicationGatewayWrapper{
		client: client,
		ResourceGroupBase: azureshared.NewResourceGroupBase(
			subscriptionID,
			resourceGroup,
			sdp.AdapterCategory_ADAPTER_CATEGORY_NETWORK,
			azureshared.NetworkApplicationGateway,
		),
	}
}

func (n networkApplicationGatewayWrapper) List(ctx context.Context) ([]*sdp.Item, *sdp.QueryError) {
	pager := n.client.List(n.ResourceGroup(), nil)

	var items []*sdp.Item
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
		}
		for _, applicationGateway := range page.Value {
			if applicationGateway.Name == nil {
				continue
			}
			item, sdpErr := n.azureApplicationGatewayToSDPItem(applicationGateway)
			if sdpErr != nil {
				return nil, sdpErr
			}
			items = append(items, item)
		}
	}
	return items, nil
}

func (n networkApplicationGatewayWrapper) ListStream(ctx context.Context, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	pager := n.client.List(n.ResourceGroup(), nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			stream.SendError(azureshared.QueryError(err, n.DefaultScope(), n.Type()))
			return
		}
		for _, applicationGateway := range page.Value {
			if applicationGateway.Name == nil {
				continue
			}
			item, sdpErr := n.azureApplicationGatewayToSDPItem(applicationGateway)
			if sdpErr != nil {
				stream.SendError(sdpErr)
				continue
			}
			cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)
			stream.SendItem(item)
		}
	}
}
func (n networkApplicationGatewayWrapper) azureApplicationGatewayToSDPItem(applicationGateway *armnetwork.ApplicationGateway) (*sdp.Item, *sdp.QueryError) {
	if applicationGateway.Name == nil {
		return nil, azureshared.QueryError(errors.New("application gateway name is nil"), n.DefaultScope(), n.Type())
	}
	attributes, err := shared.ToAttributesWithExclude(applicationGateway, "tags")
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}
	applicationGatewayName := *applicationGateway.Name
	if applicationGatewayName == "" {
		return nil, azureshared.QueryError(errors.New("application gateway name cannot be empty"), n.DefaultScope(), n.Type())
	}
	sdpItem := &sdp.Item{
		Type:            azureshared.NetworkApplicationGateway.String(),
		UniqueAttribute: "name",
		Attributes:      attributes,
		Scope:           n.DefaultScope(),
		Tags:            azureshared.ConvertAzureTags(applicationGateway.Tags),
	}

	if applicationGateway.Properties == nil {
		return sdpItem, nil
	}

	// Process GatewayIPConfigurations (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-ip-configurations/get
	if applicationGateway.Properties.GatewayIPConfigurations != nil {
		for _, gatewayIPConfig := range applicationGateway.Properties.GatewayIPConfigurations {
			if gatewayIPConfig.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayGatewayIPConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *gatewayIPConfig.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // GatewayIPConfiguration changes affect the Application Gateway's network configuration
						Out: true, // Application Gateway changes (like deletion) affect the gateway IP configuration
					},
				})

				// Link to Subnet from GatewayIPConfiguration
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
				if gatewayIPConfig.Properties != nil && gatewayIPConfig.Properties.Subnet != nil && gatewayIPConfig.Properties.Subnet.ID != nil {
					subnetID := *gatewayIPConfig.Properties.Subnet.ID
					subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
					if len(subnetParams) >= 2 {
						vnetName := subnetParams[0]
						subnetName := subnetParams[1]
						scope := n.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkSubnet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(vnetName, subnetName),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // Subnet changes affect the Application Gateway's network configuration
								Out: false, // Application Gateway changes don't affect the subnet itself
							},
						})

						// Link to VirtualNetwork (extracted from subnet ID)
						scope = n.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkVirtualNetwork.String(),
								Method: sdp.QueryMethod_GET,
								Query:  vnetName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // VirtualNetwork changes affect the Application Gateway's network configuration
								Out: false, // Application Gateway changes don't affect the virtual network itself
							},
						})
					}
				}
			}
		}
	}

	// Process FrontendIPConfigurations (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-frontend-ip-configurations/get
	if applicationGateway.Properties.FrontendIPConfigurations != nil {
		for _, frontendIPConfig := range applicationGateway.Properties.FrontendIPConfigurations {
			if frontendIPConfig.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayFrontendIPConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *frontendIPConfig.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // FrontendIPConfiguration changes affect the Application Gateway's frontend configuration
						Out: true, // Application Gateway changes (like deletion) affect the frontend IP configuration
					},
				})
			}

			if frontendIPConfig.Properties != nil {
				// Link to Public IP Address if referenced
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/public-ip-addresses/get
				if frontendIPConfig.Properties.PublicIPAddress != nil && frontendIPConfig.Properties.PublicIPAddress.ID != nil {
					publicIPName := azureshared.ExtractResourceName(*frontendIPConfig.Properties.PublicIPAddress.ID)
					if publicIPName != "" {
						scope := n.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(*frontendIPConfig.Properties.PublicIPAddress.ID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkPublicIPAddress.String(),
								Method: sdp.QueryMethod_GET,
								Query:  publicIPName,
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // Public IP changes affect the Application Gateway's frontend configuration
								Out: false, // Application Gateway changes don't affect the public IP address itself
							},
						})
					}
				}

				// Link to Subnet if referenced (for private IP)
				// Reference: https://learn.microsoft.com/en-us/rest/api/virtualnetwork/subnets/get
				if frontendIPConfig.Properties.Subnet != nil && frontendIPConfig.Properties.Subnet.ID != nil {
					subnetID := *frontendIPConfig.Properties.Subnet.ID
					subnetParams := azureshared.ExtractPathParamsFromResourceID(subnetID, []string{"virtualNetworks", "subnets"})
					if len(subnetParams) >= 2 {
						vnetName := subnetParams[0]
						subnetName := subnetParams[1]
						scope := n.DefaultScope()
						if extractedScope := azureshared.ExtractScopeFromResourceID(subnetID); extractedScope != "" {
							scope = extractedScope
						}
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   azureshared.NetworkSubnet.String(),
								Method: sdp.QueryMethod_GET,
								Query:  shared.CompositeLookupKey(vnetName, subnetName),
								Scope:  scope,
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true,  // Subnet changes affect the Application Gateway's frontend configuration
								Out: false, // Application Gateway changes don't affect the subnet itself
							},
						})
					}
				}

				// Link to IP address (standard library) if private IP address is assigned
				if frontendIPConfig.Properties.PrivateIPAddress != nil && *frontendIPConfig.Properties.PrivateIPAddress != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkIP.String(),
							Method: sdp.QueryMethod_GET,
							Query:  *frontendIPConfig.Properties.PrivateIPAddress,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // IPs are always linked bidirectionally
							Out: true,
						},
					})
				}
			}
		}
	}

	// Process BackendAddressPools (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-backend-address-pools/get
	if applicationGateway.Properties.BackendAddressPools != nil {
		for _, backendPool := range applicationGateway.Properties.BackendAddressPools {
			if backendPool.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayBackendAddressPool.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *backendPool.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // BackendAddressPool changes affect which backends receive traffic
						Out: true, // Application Gateway changes (like deletion) affect the backend address pool
					},
				})
			}

			// Link to IP addresses in backend addresses
			if backendPool.Properties != nil && backendPool.Properties.BackendAddresses != nil {
				for _, backendAddress := range backendPool.Properties.BackendAddresses {
					if backendAddress.IPAddress != nil && *backendAddress.IPAddress != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkIP.String(),
								Method: sdp.QueryMethod_GET,
								Query:  *backendAddress.IPAddress,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // IPs are always linked bidirectionally
								Out: true,
							},
						})
					}

					// Link to DNS name (standard library) if FQDN is configured
					if backendAddress.Fqdn != nil && *backendAddress.Fqdn != "" {
						sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
							Query: &sdp.Query{
								Type:   stdlib.NetworkDNS.String(),
								Method: sdp.QueryMethod_SEARCH,
								Query:  *backendAddress.Fqdn,
								Scope:  "global",
							},
							BlastPropagation: &sdp.BlastPropagation{
								In:  true, // DNS names are always linked bidirectionally
								Out: true,
							},
						})
					}
				}
			}
		}
	}

	// Process HTTPListeners (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-http-listeners/get
	if applicationGateway.Properties.HTTPListeners != nil {
		for _, httpListener := range applicationGateway.Properties.HTTPListeners {
			if httpListener.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayHTTPListener.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *httpListener.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // HTTPListener changes affect how the Application Gateway receives traffic
						Out: true, // Application Gateway changes (like deletion) affect the HTTP listener
					},
				})
			}

			// Link to DNS names (standard library) if hostnames are configured
			// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-http-listeners/get
			if httpListener.Properties != nil {
				// Single hostname (HostName)
				if httpListener.Properties.HostName != nil && *httpListener.Properties.HostName != "" {
					sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
						Query: &sdp.Query{
							Type:   stdlib.NetworkDNS.String(),
							Method: sdp.QueryMethod_SEARCH,
							Query:  *httpListener.Properties.HostName,
							Scope:  "global",
						},
						BlastPropagation: &sdp.BlastPropagation{
							In:  true, // DNS name changes affect how the Application Gateway receives traffic
							Out: true, // DNS names are always linked bidirectionally
						},
					})
				}

				// Multiple hostnames (HostNames) for multi-site listeners
				if httpListener.Properties.HostNames != nil {
					for _, hostName := range httpListener.Properties.HostNames {
						if hostName != nil && *hostName != "" {
							sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
								Query: &sdp.Query{
									Type:   stdlib.NetworkDNS.String(),
									Method: sdp.QueryMethod_SEARCH,
									Query:  *hostName,
									Scope:  "global",
								},
								BlastPropagation: &sdp.BlastPropagation{
									In:  true, // DNS name changes affect how the Application Gateway receives traffic
									Out: true, // DNS names are always linked bidirectionally
								},
							})
						}
					}
				}
			}
		}
	}

	// Process BackendHTTPSettingsCollection (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-backend-http-settings/get
	if applicationGateway.Properties.BackendHTTPSettingsCollection != nil {
		for _, backendHTTPSettings := range applicationGateway.Properties.BackendHTTPSettingsCollection {
			if backendHTTPSettings.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayBackendHTTPSettings.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *backendHTTPSettings.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // BackendHTTPSettings changes affect how the Application Gateway communicates with backends
						Out: true, // Application Gateway changes (like deletion) affect the backend HTTP settings
					},
				})
			}

			// Link to DNS name (standard library) if hostname override is configured
			// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-backend-http-settings/get
			if backendHTTPSettings.Properties != nil && backendHTTPSettings.Properties.HostName != nil && *backendHTTPSettings.Properties.HostName != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *backendHTTPSettings.Properties.HostName,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // DNS name changes affect backend communication
						Out: true, // DNS names are always linked bidirectionally
					},
				})
			}
		}
	}

	// Process RequestRoutingRules (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-request-routing-rules/get
	if applicationGateway.Properties.RequestRoutingRules != nil {
		for _, rule := range applicationGateway.Properties.RequestRoutingRules {
			if rule.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayRequestRoutingRule.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *rule.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // RequestRoutingRule changes affect how traffic is routed
						Out: true, // Application Gateway changes (like deletion) affect the routing rule
					},
				})
			}
		}
	}

	// Process Probes (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-health-probes/get
	if applicationGateway.Properties.Probes != nil {
		for _, probe := range applicationGateway.Properties.Probes {
			if probe.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayProbe.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *probe.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // Probe changes affect backend health monitoring
						Out: true, // Application Gateway changes (like deletion) affect the probe
					},
				})
			}

			// Link to DNS name (standard library) if probe host is configured
			// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-health-probes/get
			if probe.Properties != nil && probe.Properties.Host != nil && *probe.Properties.Host != "" {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   stdlib.NetworkDNS.String(),
						Method: sdp.QueryMethod_SEARCH,
						Query:  *probe.Properties.Host,
						Scope:  "global",
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // DNS name changes affect health probe targets
						Out: true, // DNS names are always linked bidirectionally
					},
				})
			}
		}
	}

	// Process SSLCertificates (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-ssl-certificates/get
	if applicationGateway.Properties.SSLCertificates != nil {
		for _, sslCert := range applicationGateway.Properties.SSLCertificates {
			if sslCert.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewaySSLCertificate.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *sslCert.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // SSLCertificate changes affect HTTPS listeners
						Out: true, // Application Gateway changes (like deletion) affect the SSL certificate
					},
				})
			}
		}
	}

	// Process URLPathMaps (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-url-path-maps/get
	if applicationGateway.Properties.URLPathMaps != nil {
		for _, urlPathMap := range applicationGateway.Properties.URLPathMaps {
			if urlPathMap.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayURLPathMap.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *urlPathMap.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // URLPathMap changes affect path-based routing
						Out: true, // Application Gateway changes (like deletion) affect the URL path map
					},
				})
			}
		}
	}

	// Process AuthenticationCertificates (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-authentication-certificates/get
	if applicationGateway.Properties.AuthenticationCertificates != nil {
		for _, authCert := range applicationGateway.Properties.AuthenticationCertificates {
			if authCert.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayAuthenticationCertificate.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *authCert.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // AuthenticationCertificate changes affect backend authentication
						Out: true, // Application Gateway changes (like deletion) affect the authentication certificate
					},
				})
			}
		}
	}

	// Process TrustedRootCertificates (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-trusted-root-certificates/get
	if applicationGateway.Properties.TrustedRootCertificates != nil {
		for _, trustedRootCert := range applicationGateway.Properties.TrustedRootCertificates {
			if trustedRootCert.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayTrustedRootCertificate.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *trustedRootCert.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // TrustedRootCertificate changes affect backend server validation
						Out: true, // Application Gateway changes (like deletion) affect the trusted root certificate
					},
				})
			}
		}
	}

	// Process RewriteRuleSets (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-rewrite-rule-sets/get
	if applicationGateway.Properties.RewriteRuleSets != nil {
		for _, rewriteRuleSet := range applicationGateway.Properties.RewriteRuleSets {
			if rewriteRuleSet.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayRewriteRuleSet.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *rewriteRuleSet.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // RewriteRuleSet changes affect request/response modification
						Out: true, // Application Gateway changes (like deletion) affect the rewrite rule set
					},
				})
			}
		}
	}

	// Process RedirectConfigurations (Child Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-redirect-configurations/get
	if applicationGateway.Properties.RedirectConfigurations != nil {
		for _, redirectConfig := range applicationGateway.Properties.RedirectConfigurations {
			if redirectConfig.Name != nil {
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.NetworkApplicationGatewayRedirectConfiguration.String(),
						Method: sdp.QueryMethod_GET,
						Query:  shared.CompositeLookupKey(applicationGatewayName, *redirectConfig.Name),
						Scope:  n.DefaultScope(),
					},
					BlastPropagation: &sdp.BlastPropagation{
						In:  true, // RedirectConfiguration changes affect URL redirection behavior
						Out: true, // Application Gateway changes (like deletion) affect the redirect configuration
					},
				})
			}
		}
	}

	// Link to Web Application Firewall Policy (External Resource)
	// Reference: https://learn.microsoft.com/en-us/rest/api/application-gateway/application-gateway-web-application-firewall-policies/get
	if applicationGateway.Properties.FirewallPolicy != nil && applicationGateway.Properties.FirewallPolicy.ID != nil {
		firewallPolicyName := azureshared.ExtractResourceName(*applicationGateway.Properties.FirewallPolicy.ID)
		if firewallPolicyName != "" {
			scope := n.DefaultScope()
			if extractedScope := azureshared.ExtractScopeFromResourceID(*applicationGateway.Properties.FirewallPolicy.ID); extractedScope != "" {
				scope = extractedScope
			}
			sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
				Query: &sdp.Query{
					Type:   azureshared.NetworkApplicationGatewayWebApplicationFirewallPolicy.String(),
					Method: sdp.QueryMethod_GET,
					Query:  firewallPolicyName,
					Scope:  scope,
				},
				BlastPropagation: &sdp.BlastPropagation{
					In:  true,  // WAF Policy changes affect the Application Gateway's security configuration
					Out: false, // Application Gateway changes don't affect the WAF policy itself
				},
			})
		}
	}

	// Link to User Assigned Managed Identities (external resources)
	// Reference: https://learn.microsoft.com/en-us/rest/api/managedidentity/user-assigned-identities/get?view=rest-managedidentity-2024-11-30&tabs=HTTP
	if applicationGateway.Identity != nil && applicationGateway.Identity.UserAssignedIdentities != nil {
		for identityResourceID := range applicationGateway.Identity.UserAssignedIdentities {
			identityName := azureshared.ExtractResourceName(identityResourceID)
			if identityName != "" {
				// Extract scope from resource ID if it's in a different resource group
				scope := n.DefaultScope()
				if extractedScope := azureshared.ExtractScopeFromResourceID(identityResourceID); extractedScope != "" {
					scope = extractedScope
				}
				sdpItem.LinkedItemQueries = append(sdpItem.LinkedItemQueries, &sdp.LinkedItemQuery{
					Query: &sdp.Query{
						Type:   azureshared.ManagedIdentityUserAssignedIdentity.String(),
						Method: sdp.QueryMethod_GET,
						Query:  identityName,
						Scope:  scope,
					},
					BlastPropagation: &sdp.BlastPropagation{
						// Application Gateway depends on managed identity for authentication (e.g., Key Vault integration for SSL certificates)
						// If identity is deleted/modified, Application Gateway operations may fail
						In:  true,
						Out: false,
					},
				})
			}
		}
	}

	return sdpItem, nil
}

func (n networkApplicationGatewayWrapper) Get(ctx context.Context, queryParts ...string) (*sdp.Item, *sdp.QueryError) {
	if len(queryParts) != 1 {
		return nil, azureshared.QueryError(errors.New("query must be exactly one part and be a application gateway name"), n.DefaultScope(), n.Type())
	}
	applicationGatewayName := queryParts[0]
	if applicationGatewayName == "" {
		return nil, azureshared.QueryError(errors.New("application gateway name cannot be empty"), n.DefaultScope(), n.Type())
	}
	resp, err := n.client.Get(ctx, n.ResourceGroup(), applicationGatewayName, nil)
	if err != nil {
		return nil, azureshared.QueryError(err, n.DefaultScope(), n.Type())
	}
	return n.azureApplicationGatewayToSDPItem(&resp.ApplicationGateway)
}

func (n networkApplicationGatewayWrapper) GetLookups() sources.ItemTypeLookups {
	return sources.ItemTypeLookups{
		NetworkApplicationGatewayLookupByName,
	}
}

func (n networkApplicationGatewayWrapper) PotentialLinks() map[shared.ItemType]bool {
	return shared.NewItemTypesSet(
		// Child resources
		azureshared.NetworkApplicationGatewayGatewayIPConfiguration,
		azureshared.NetworkApplicationGatewayFrontendIPConfiguration,
		azureshared.NetworkApplicationGatewayBackendAddressPool,
		azureshared.NetworkApplicationGatewayHTTPListener,
		azureshared.NetworkApplicationGatewayBackendHTTPSettings,
		azureshared.NetworkApplicationGatewayRequestRoutingRule,
		azureshared.NetworkApplicationGatewayProbe,
		azureshared.NetworkApplicationGatewaySSLCertificate,
		azureshared.NetworkApplicationGatewayURLPathMap,
		azureshared.NetworkApplicationGatewayAuthenticationCertificate,
		azureshared.NetworkApplicationGatewayTrustedRootCertificate,
		azureshared.NetworkApplicationGatewayRewriteRuleSet,
		azureshared.NetworkApplicationGatewayRedirectConfiguration,
		// External resources
		azureshared.NetworkSubnet,
		azureshared.NetworkVirtualNetwork,
		azureshared.NetworkPublicIPAddress,
		azureshared.NetworkApplicationGatewayWebApplicationFirewallPolicy,
		azureshared.ManagedIdentityUserAssignedIdentity,
		// Standard library types
		stdlib.NetworkIP,
		stdlib.NetworkDNS,
	)
}

func (n networkApplicationGatewayWrapper) TerraformMappings() []*sdp.TerraformMapping {
	return []*sdp.TerraformMapping{
		{
			TerraformMethod:   sdp.QueryMethod_GET,
			TerraformQueryMap: "azurerm_application_gateway.name",
		},
	}
}

func (n networkApplicationGatewayWrapper) IAMPermissions() []string {
	return []string{
		"Microsoft.Network/applicationGateways/read",
	}
}

func (n networkApplicationGatewayWrapper) PredefinedRole() string {
	return "Reader"
}
