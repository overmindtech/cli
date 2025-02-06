package networkmanager

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

const (
	globalNetworkSrc   = "global-network"
	siteSrc            = "site"
	linkSrc            = "link"
	deviceSrc          = "device"
	linkAssociationSrc = "link-association"
	connectionSrc      = "connection"
)

const (
	deviceOneName = "device-1"
	deviceTwoName = "device-2"
)

func setup(ctx context.Context, logger *slog.Logger, networkmanagerClient *networkmanager.Client) error {
	testID := integration.TestID()

	// Create a global network
	globalNetworkID, err := createGlobalNetwork(ctx, logger, networkmanagerClient, testID)
	if err != nil {
		return err
	}

	// Create a site in the global network
	siteID, err := createSite(ctx, logger, networkmanagerClient, testID, globalNetworkID)
	if err != nil {
		return err
	}

	// Create a link in the global network for the site
	linkID, err := createLink(ctx, logger, networkmanagerClient, testID, globalNetworkID, siteID)
	if err != nil {
		return err
	}

	// Create a device in the global network for the site
	deviceOneID, err := createDevice(ctx, logger, networkmanagerClient, testID, globalNetworkID, siteID, deviceOneName)
	if err != nil {
		return err
	}

	// Create a link association in the global network for the device
	err = createLinkAssociation(ctx, logger, networkmanagerClient, globalNetworkID, deviceOneID, linkID)
	if err != nil {
		return err
	}

	// Create another device in the global network for the site
	deviceTwoID, err := createDevice(ctx, logger, networkmanagerClient, testID, globalNetworkID, siteID, deviceTwoName)
	if err != nil {
		return err
	}

	// Create a connection between the devices
	err = createConnection(ctx, logger, networkmanagerClient, globalNetworkID, deviceOneID, deviceTwoID)
	if err != nil {
		return err
	}

	return nil
}
