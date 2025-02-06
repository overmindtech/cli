package networkmanager

import (
	"context"
	"errors"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/overmindtech/cli/aws-source/adapters/integration"
)

func createGlobalNetwork(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, testID string) (*string, error) {
	tags := resourceTags(globalNetworkSrc, testID)

	id, err := findGlobalNetworkIDByTags(ctx, client, tags)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating global network")
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Global network already exists")
		return id, nil
	}

	input := &networkmanager.CreateGlobalNetworkInput{
		Description: aws.String("Integration test global network"),
		Tags:        tags,
	}

	response, err := client.CreateGlobalNetwork(ctx, input)
	if err != nil {
		return nil, err
	}

	return response.GlobalNetwork.GlobalNetworkId, nil
}

func createSite(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, testID string, globalNetworkID *string) (*string, error) {
	tags := resourceTags(siteSrc, testID)

	id, err := findSiteIDByTags(ctx, client, globalNetworkID, tags)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating site")
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Site already exists")
		return id, nil
	}

	input := &networkmanager.CreateSiteInput{
		GlobalNetworkId: globalNetworkID,
		Description:     aws.String("Integration test site"),
		Tags:            tags,
	}

	response, err := client.CreateSite(ctx, input)
	if err != nil {
		return nil, err
	}

	return response.Site.SiteId, nil
}

func createLink(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, testID string, globalNetworkID, siteID *string) (*string, error) {
	tags := resourceTags(linkSrc, testID)

	id, err := findLinkIDByTags(ctx, client, globalNetworkID, siteID, tags)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating link")
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Link already exists")
		return id, nil
	}

	input := &networkmanager.CreateLinkInput{
		GlobalNetworkId: globalNetworkID,
		SiteId:          siteID,
		Description:     aws.String("Integration test link"),
		Bandwidth: &types.Bandwidth{
			UploadSpeed:   aws.Int32(50),
			DownloadSpeed: aws.Int32(50),
		},
		Tags: tags,
	}

	response, err := client.CreateLink(ctx, input)
	if err != nil {
		return nil, err
	}

	return response.Link.LinkId, nil
}

func createDevice(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, testID string, globalNetworkID, siteID *string, deviceName string) (*string, error) {
	tags := resourceTags(deviceSrc, testID, deviceName)

	id, err := findDeviceIDByTags(ctx, client, globalNetworkID, siteID, tags)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating device", "name", deviceName)
		} else {
			return nil, err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Device already exists", "name", deviceName)
		return id, nil
	}

	input := &networkmanager.CreateDeviceInput{
		GlobalNetworkId: globalNetworkID,
		SiteId:          siteID,
		Tags:            tags,
	}

	response, err := client.CreateDevice(ctx, input)
	if err != nil {
		return nil, err
	}

	return response.Device.DeviceId, nil
}

func createLinkAssociation(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, globalNetworkID, deviceID, linkID *string) error {
	id, err := findLinkAssociationID(ctx, client, globalNetworkID, linkID, deviceID)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating link association")
		} else {
			return err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Link association already exists")
		return nil
	}

	input := &networkmanager.AssociateLinkInput{
		DeviceId:        deviceID,
		GlobalNetworkId: globalNetworkID,
		LinkId:          linkID,
	}

	_, err = client.AssociateLink(ctx, input)
	if err != nil {
		return err
	}

	return nil
}

func createConnection(ctx context.Context, logger *slog.Logger, client *networkmanager.Client, globalNetworkID, deviceID, connectedDeviceID *string) error {
	id, err := findConnectionID(ctx, client, globalNetworkID, deviceID)
	if err != nil {
		if errors.As(err, new(integration.NotFoundError)) {
			logger.InfoContext(ctx, "Creating connection")
		} else {
			return err
		}
	}

	if id != nil {
		logger.InfoContext(ctx, "Connection already exists")
		return nil
	}

	input := &networkmanager.CreateConnectionInput{
		GlobalNetworkId:   globalNetworkID,
		DeviceId:          deviceID,
		ConnectedDeviceId: connectedDeviceID,
	}

	_, err = client.CreateConnection(ctx, input)
	if err != nil {
		return err
	}

	return nil
}
