package networkmanager

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/aws-source/adapters"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
)

func searchSync(adapter discovery.SearchStreamableAdapter, ctx context.Context, scope, query string, ignoreCache bool) ([]*sdp.Item, error) {
	stream := discovery.NewRecordingQueryResultStream()
	adapter.SearchStream(ctx, scope, query, ignoreCache, stream)

	errs := stream.GetErrors()
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to search: %v", errs)
	}

	return stream.GetItems(), nil
}

func NetworkManager(t *testing.T) {
	ctx := context.Background()

	var err error
	testClient, err := networkManagerClient(ctx)
	if err != nil {
		t.Fatalf("Failed to create NetworkManager client: %v", err)
	}

	testAWSConfig, err := integration.AWSSettings(ctx)
	if err != nil {
		t.Fatalf("Failed to get AWS settings: %v", err)
	}

	accountID := testAWSConfig.AccountID

	t.Logf("Running NetworkManager integration tests")

	globalNetworkSource := adapters.NewNetworkManagerGlobalNetworkAdapter(testClient, accountID)
	if err := globalNetworkSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager global network adapter: %v", err)
	}

	siteSource := adapters.NewNetworkManagerSiteAdapter(testClient, accountID)
	if err := siteSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager site adapter: %v", err)
	}

	linkSource := adapters.NewNetworkManagerLinkAdapter(testClient, accountID)
	if err := linkSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager link adapter: %v", err)
	}

	linkAssociationSource := adapters.NewNetworkManagerLinkAssociationAdapter(testClient, accountID)
	if err := linkAssociationSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager link association adapter: %v", err)
	}

	connectionSource := adapters.NewNetworkManagerConnectionAdapter(testClient, accountID)
	if err := connectionSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager connection adapter: %v", err)
	}

	deviceSource := adapters.NewNetworkManagerDeviceAdapter(testClient, accountID)
	if err := deviceSource.Validate(); err != nil {
		t.Fatalf("failed to validate NetworkManager device adapter: %v", err)
	}

	globalScope := adapterhelpers.FormatScope(accountID, "")

	t.Run("Global Network", func(t *testing.T) {
		stream := discovery.NewRecordingQueryResultStream()
		globalNetworkSource.ListStream(ctx, globalScope, false, stream)

		errs := stream.GetErrors()
		if len(errs) > 0 {
			t.Fatalf("failed to list NetworkManager global networks: %v", errs)
		}

		items := stream.GetItems()
		if len(items) == 0 {
			t.Fatalf("no global networks found")
		}

		globalNetworkUniqueAttribute := items[0].GetUniqueAttribute()

		globalNetworkID, err := integration.GetUniqueAttributeValueByTags(globalNetworkUniqueAttribute, items, integration.ResourceTags(integration.NetworkManager, globalNetworkSrc), false)
		if err != nil {
			t.Fatalf("failed to get global network ID: %v", err)
		}

		// Get global network
		globalNetwork, err := globalNetworkSource.Get(ctx, globalScope, globalNetworkID, true)
		if err != nil {
			t.Fatalf("failed to get NetworkManager global network: %v", err)
		}

		globalNetworkIDFromGet, err := integration.GetUniqueAttributeValueByTags(globalNetworkUniqueAttribute, []*sdp.Item{globalNetwork}, integration.ResourceTags(integration.NetworkManager, globalNetworkSrc), false)
		if err != nil {
			t.Fatalf("failed to get global network ID from get: %v", err)
		}

		if globalNetworkID != globalNetworkIDFromGet {
			t.Fatalf("expected global network ID %s, got %s", globalNetworkID, globalNetworkIDFromGet)
		}

		// Search global network by ARN
		globalNetworkARN, err := globalNetwork.GetAttributes().Get("GlobalNetworkArn")
		if err != nil {
			t.Fatalf("failed to get global network ARN: %v", err)
		}

		if globalScope != globalNetwork.GetScope() {
			t.Fatalf("expected global scope %s, got %s", globalScope, globalNetwork.GetScope())
		}

		items, err = searchSync(globalNetworkSource, ctx, globalScope, globalNetworkARN.(string), true)
		if err != nil {
			t.Fatalf("failed to search NetworkManager global networks: %v", err)
		}

		if len(items) == 0 {
			t.Fatalf("no global networks found")
		}

		globalNetworkIDFromSearch, err := integration.GetUniqueAttributeValueByTags(globalNetworkUniqueAttribute, items, integration.ResourceTags(integration.NetworkManager, globalNetworkSrc), false)
		if err != nil {
			t.Fatalf("failed to get global network ID from search: %v", err)
		}

		if globalNetworkID != globalNetworkIDFromSearch {
			t.Fatalf("expected global network ID %s, got %s", globalNetworkID, globalNetworkIDFromSearch)
		}

		t.Run("Site", func(t *testing.T) {
			// Search sites by the global network ID that they are created on
			sites, err := searchSync(siteSource, ctx, globalScope, globalNetworkID, true)
			if err != nil {
				t.Fatalf("failed to search for site: %v", err)
			}

			if len(sites) == 0 {
				t.Fatalf("no sites found")
			}

			siteUniqueAttribute := sites[0].GetUniqueAttribute()

			// composite site id is in the format of {globalNetworkID}|{siteID}
			compositeSiteID, err := integration.GetUniqueAttributeValueByTags(siteUniqueAttribute, sites, integration.ResourceTags(integration.NetworkManager, siteSrc), false)
			if err != nil {
				t.Fatalf("failed to get site ID from search: %v", err)
			}

			// Get site: query format = globalNetworkID|siteID
			site, err := siteSource.Get(ctx, globalScope, compositeSiteID, true)
			if err != nil {
				t.Fatalf("failed to get site: %v", err)
			}

			siteIDFromGet, err := integration.GetUniqueAttributeValueByTags(siteUniqueAttribute, []*sdp.Item{site}, integration.ResourceTags(integration.NetworkManager, siteSrc), false)
			if err != nil {
				t.Fatalf("failed to get site ID from get: %v", err)
			}

			if compositeSiteID != siteIDFromGet {
				t.Fatalf("expected site ID %s, got %s", compositeSiteID, siteIDFromGet)
			}

			siteID := strings.Split(compositeSiteID, "|")[1]

			t.Run("Link", func(t *testing.T) {
				// Search links by the global network ID that they are created on
				links, err := searchSync(linkSource, ctx, globalScope, globalNetworkID, true)
				if err != nil {
					t.Fatalf("failed to search for link: %v", err)
				}

				if len(links) == 0 {
					t.Fatalf("no links found")
				}

				linkUniqueAttribute := links[0].GetUniqueAttribute()

				compositeLinkID, err := integration.GetUniqueAttributeValueByTags(linkUniqueAttribute, links, integration.ResourceTags(integration.NetworkManager, linkSrc), false)
				if err != nil {
					t.Fatalf("failed to get link ID from search: %v", err)
				}

				// Get link: query format = globalNetworkID|linkID
				link, err := linkSource.Get(ctx, globalScope, compositeLinkID, true)
				if err != nil {
					t.Fatalf("failed to get link: %v", err)
				}

				linkIDFromGet, err := integration.GetUniqueAttributeValueByTags(linkUniqueAttribute, []*sdp.Item{link}, integration.ResourceTags(integration.NetworkManager, linkSrc), false)
				if err != nil {
					t.Fatalf("failed to get link ID from get: %v", err)
				}

				if compositeLinkID != linkIDFromGet {
					t.Fatalf("expected link ID %s, got %s", compositeLinkID, linkIDFromGet)
				}

				linkID := strings.Split(compositeLinkID, "|")[1]

				t.Run("Device", func(t *testing.T) {
					// Search devices by the global network ID and site ID
					// query format = globalNetworkID|siteID
					queryDevice := fmt.Sprintf("%s|%s", globalNetworkID, siteID)
					devices, err := searchSync(deviceSource, ctx, globalScope, queryDevice, true)
					if err != nil {
						t.Fatalf("failed to search for device: %v", err)
					}

					if len(devices) == 0 {
						t.Fatalf("no devices found")
					}

					deviceUniqueAttribute := devices[0].GetUniqueAttribute()

					// composite device id is in the format of: {globalNetworkID}|{deviceID}
					deviceOneCompositeID, err := integration.GetUniqueAttributeValueByTags(deviceUniqueAttribute, devices, integration.ResourceTags(integration.NetworkManager, deviceSrc, deviceOneName), false)
					if err != nil {
						t.Fatalf("failed to get device ID from search: %v", err)
					}

					// Get device: query format = globalNetworkID|deviceID
					device, err := deviceSource.Get(ctx, globalScope, deviceOneCompositeID, true)
					if err != nil {
						t.Fatalf("failed to get device: %v", err)
					}

					deviceOneCompositeIDFromGet, err := integration.GetUniqueAttributeValueByTags(deviceUniqueAttribute, []*sdp.Item{device}, integration.ResourceTags(integration.NetworkManager, deviceSrc, deviceOneName), false)
					if err != nil {
						t.Fatalf("failed to get device ID from get: %v", err)
					}

					if deviceOneCompositeID != deviceOneCompositeIDFromGet {
						t.Fatalf("expected device ID %s, got %s", deviceOneCompositeID, deviceOneCompositeIDFromGet)
					}

					deviceOneID := strings.Split(deviceOneCompositeID, "|")[1]

					// Search devices by the global network ID
					devicesByGlobalNetwork, err := searchSync(deviceSource, ctx, globalScope, globalNetworkID, true)
					if err != nil {
						t.Fatalf("failed to search for device by global network: %v", err)
					}

					integration.AssertEqualItems(t, devices, devicesByGlobalNetwork, deviceUniqueAttribute)

					t.Run("Link Association", func(t *testing.T) {
						// Search link associations by the global network ID, link ID
						queryLALink := fmt.Sprintf("%s|link|%s", globalNetworkID, linkID)
						linkAssociations, err := searchSync(linkAssociationSource, ctx, globalScope, queryLALink, true)
						if err != nil {
							t.Fatalf("failed to search for link association: %v", err)
						}

						if len(linkAssociations) == 0 {
							t.Fatalf("no link associations found")
						}

						linkAssociationUniqueAttribute := linkAssociations[0].GetUniqueAttribute()

						// composite link association id is in the format of: {globalNetworkID}|{linkID}|{deviceID}
						compositeLinkAssociationID, err := integration.GetUniqueAttributeValueByTags(linkAssociationUniqueAttribute, linkAssociations, nil, false)
						if err != nil {
							t.Fatalf("failed to get link association ID from search: %v", err)
						}

						// Get link association: query format = globalNetworkID|linkID|deviceID
						linkAssociation, err := linkAssociationSource.Get(ctx, globalScope, compositeLinkAssociationID, true)
						if err != nil {
							t.Fatalf("failed to get link association: %v", err)
						}

						compositeLinkAssociationIDFromGet, err := integration.GetUniqueAttributeValueByTags(linkAssociationUniqueAttribute, []*sdp.Item{linkAssociation}, nil, false)
						if err != nil {
							t.Fatalf("failed to get link association ID from get: %v", err)
						}

						if compositeLinkAssociationID != compositeLinkAssociationIDFromGet {
							t.Fatalf("expected link association ID %s, got %s", compositeLinkAssociationID, compositeLinkAssociationIDFromGet)
						}

						// Search link associations by the global network ID
						searchLinkAssociationsByGlobalNetwork, err := searchSync(linkAssociationSource, ctx, globalScope, globalNetworkID, true)
						if err != nil {
							t.Fatalf("failed to search for link association by global network: %v", err)
						}

						integration.AssertEqualItems(t, linkAssociations, searchLinkAssociationsByGlobalNetwork, linkAssociationUniqueAttribute)

						// Search link associations by the global network ID and device ID
						queryLADevice := fmt.Sprintf("%s|device|%s", globalNetworkID, deviceOneID)
						linkAssociationsByDevice, err := searchSync(linkAssociationSource, ctx, globalScope, queryLADevice, true)
						if err != nil {
							t.Fatalf("failed to search for link association by device: %v", err)
						}

						integration.AssertEqualItems(t, linkAssociations, linkAssociationsByDevice, linkAssociationUniqueAttribute)
					})

					t.Run("Connection", func(t *testing.T) {
						// Search connections by the global network ID
						connections, err := searchSync(connectionSource, ctx, globalScope, globalNetworkID, true)
						if err != nil {
							t.Fatalf("failed to search for connection: %v", err)
						}

						if len(connections) == 0 {
							t.Fatalf("no connections found")
						}

						connectionUniqueAttribute := connections[0].GetUniqueAttribute()

						// composite connection id is in the format of: {globalNetworkID}|{connectionID}
						compositeConnectionID, err := integration.GetUniqueAttributeValueByTags(connectionUniqueAttribute, connections, nil, false)
						if err != nil {
							t.Fatalf("failed to get connection ID from search: %v", err)
						}

						// Get connection: query format = globalNetworkID|connectionID
						connection, err := connectionSource.Get(ctx, globalScope, compositeConnectionID, true)
						if err != nil {
							t.Fatalf("failed to get connection: %v", err)
						}

						compositeConnectionIDFromGet, err := integration.GetUniqueAttributeValueByTags(connectionUniqueAttribute, []*sdp.Item{connection}, nil, false)
						if err != nil {
							t.Fatalf("failed to get connection ID from get: %v", err)
						}

						if compositeConnectionID != compositeConnectionIDFromGet {
							t.Fatalf("expected connection ID %s, got %s", compositeConnectionID, compositeConnectionIDFromGet)
						}

						// Search connections by global network ID and device ID
						queryCon := fmt.Sprintf("%s|%s", globalNetworkID, deviceOneID)
						connectionsByDevice, err := searchSync(connectionSource, ctx, globalScope, queryCon, true)
						if err != nil {
							t.Fatalf("failed to search for connection by device: %v", err)
						}

						integration.AssertEqualItems(t, connections, connectionsByDevice, connectionUniqueAttribute)
					})
				})
			})
		})
	})
}
