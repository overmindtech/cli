package networkmanager

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func teardown(ctx context.Context, logger *slog.Logger, client *networkmanager.Client) error {
	globalNetworkID, err := findGlobalNetworkIDByTags(ctx, client, resourceTags(globalNetworkSrc, integration.TestID()))
	if err != nil {
		nf := integration.NewNotFoundError(globalNetworkSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Global network not found")
			return nil
		} else {
			return err
		}
	}

	siteID, err := findSiteIDByTags(ctx, client, globalNetworkID, resourceTags(siteSrc, integration.TestID()))
	if err != nil {
		nf := integration.NewNotFoundError(siteSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Site not found")
			return nil
		} else {
			return err
		}
	}

	linkID, err := findLinkIDByTags(ctx, client, globalNetworkID, siteID, resourceTags(linkSrc, integration.TestID()))
	if err != nil {
		nf := integration.NewNotFoundError(linkSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Link not found")
			return nil
		} else {
			return err
		}
	}

	deviceOneID, err := findDeviceIDByTags(ctx, client, globalNetworkID, siteID, resourceTags(deviceSrc, integration.TestID(), deviceOneName))
	if err != nil {
		nf := integration.NewNotFoundError(deviceSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Device not found", "name", deviceOneName)
			return nil
		} else {
			return err
		}
	}

	err = deleteLinkAssociation(ctx, client, globalNetworkID, deviceOneID, linkID)
	if err != nil {
		nf := integration.NewNotFoundError(linkAssociationSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Link association not found.. ignoring")
		} else {
			return err
		}
	}

	connectionID, err := findConnectionID(ctx, client, globalNetworkID, deviceOneID)
	if err != nil {
		nf := integration.NewNotFoundError(connectionSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Connection not found")
			return nil
		} else {
			return err
		}
	}

	err = deleteConnection(ctx, client, globalNetworkID, connectionID)
	if err != nil {
		nf := integration.NewNotFoundError(connectionSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Connection not found.. ignoring", "id", connectionID)
		} else {
			return err
		}
	}

	err = deleteDevice(ctx, client, globalNetworkID, deviceOneID)
	if err != nil {
		nf := integration.NewNotFoundError(deviceSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Device not found.. ignoring", "id", deviceOneID)
		} else {
			return err
		}
	}

	deviceTwoID, err := findDeviceIDByTags(ctx, client, globalNetworkID, siteID, resourceTags(deviceSrc, integration.TestID(), deviceTwoName))
	if err != nil {
		nf := integration.NewNotFoundError(deviceSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Device not found")
			return nil
		} else {
			return err
		}
	}

	err = deleteDevice(ctx, client, globalNetworkID, deviceTwoID)
	if err != nil {
		nf := integration.NewNotFoundError(deviceSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Device not found.. ignoring", "id", deviceTwoID)
		} else {
			return err
		}
	}

	err = deleteLink(ctx, client, globalNetworkID, linkID)
	if err != nil {
		nf := integration.NewNotFoundError(linkSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Link not found.. ignoring", "id", linkID)
		} else {
			return err
		}
	}

	err = deleteSite(ctx, client, globalNetworkID, siteID)
	if err != nil {
		nf := integration.NewNotFoundError(siteSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Site not found.. ignoring", "id", siteID)
		} else {
			return err
		}
	}

	err = deleteGlobalNetwork(ctx, client, *globalNetworkID)
	if err != nil {
		nf := integration.NewNotFoundError(globalNetworkSrc)
		if errors.As(err, &nf) {
			logger.WarnContext(ctx, "Global network not found.. ignoring", "id", globalNetworkID)
		} else {
			return err
		}
	}

	return nil
}
