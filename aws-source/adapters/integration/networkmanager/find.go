package networkmanager

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func findGlobalNetworkIDByTags(ctx context.Context, client *networkmanager.Client, requiredTags []types.Tag) (*string, error) {
	result, err := client.DescribeGlobalNetworks(ctx, &networkmanager.DescribeGlobalNetworksInput{})
	if err != nil {
		return nil, err
	}

	for _, globalNetwork := range result.GlobalNetworks {
		if hasTags(globalNetwork.Tags, requiredTags) {
			return globalNetwork.GlobalNetworkId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, globalNetworkSrc))
}

func findSiteIDByTags(ctx context.Context, client *networkmanager.Client, globalNetworkID *string, requiredTags []types.Tag) (*string, error) {
	result, err := client.GetSites(ctx, &networkmanager.GetSitesInput{
		GlobalNetworkId: globalNetworkID,
	})
	if err != nil {
		return nil, err
	}

	for _, site := range result.Sites {
		if hasTags(site.Tags, requiredTags) {
			return site.SiteId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, siteSrc))
}

func findLinkIDByTags(ctx context.Context, client *networkmanager.Client, globalNetworkID, siteID *string, requiredTags []types.Tag) (*string, error) {
	result, err := client.GetLinks(ctx, &networkmanager.GetLinksInput{
		GlobalNetworkId: globalNetworkID,
		SiteId:          siteID,
	})
	if err != nil {
		return nil, err
	}

	for _, link := range result.Links {
		if hasTags(link.Tags, requiredTags) {
			return link.LinkId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, linkSrc))
}

func findDeviceIDByTags(ctx context.Context, client *networkmanager.Client, globalNetworkID, sideID *string, requiredTags []types.Tag) (*string, error) {
	result, err := client.GetDevices(ctx, &networkmanager.GetDevicesInput{
		GlobalNetworkId: globalNetworkID,
		SiteId:          sideID,
	})
	if err != nil {
		return nil, err
	}

	for _, device := range result.Devices {
		if hasTags(device.Tags, requiredTags) {
			return device.DeviceId, nil
		}
	}

	return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, deviceSrc))
}

func findLinkAssociationID(ctx context.Context, client *networkmanager.Client, globalNetworkID, linkID, deviceID *string) (*string, error) {
	result, err := client.GetLinkAssociations(ctx, &networkmanager.GetLinkAssociationsInput{
		GlobalNetworkId: globalNetworkID,
		LinkId:          linkID,
		DeviceId:        deviceID,
	})
	if err != nil {
		return nil, err
	}

	if len(result.LinkAssociations) != 1 {
		if len(result.LinkAssociations) == 0 {
			return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, linkAssociationSrc))
		}

		return nil, fmt.Errorf("expected 1 link association, got %d", len(result.LinkAssociations))
	}

	compositeKey := fmt.Sprintf("%s|%s|%s", *globalNetworkID, *linkID, *deviceID)

	return &compositeKey, nil
}

func findConnectionID(ctx context.Context, client *networkmanager.Client, globalNetworkID, deviceID *string) (*string, error) {
	result, err := client.GetConnections(ctx, &networkmanager.GetConnectionsInput{
		GlobalNetworkId: globalNetworkID,
		DeviceId:        deviceID,
	})
	if err != nil {
		return nil, err
	}

	if len(result.Connections) != 1 {
		if len(result.Connections) == 0 {
			return nil, integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, connectionSrc))
		}

		return nil, fmt.Errorf("expected 1 connection, got %d", len(result.Connections))
	}

	return result.Connections[0].ConnectionId, nil
}
