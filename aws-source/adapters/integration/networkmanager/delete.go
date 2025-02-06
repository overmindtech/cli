package networkmanager

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/networkmanager/types"
	"github.com/aws/smithy-go"
	"github.com/overmindtech/cli/aws-source/adapters/integration"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/networkmanager"
)

func deleteGlobalNetwork(ctx context.Context, client *networkmanager.Client, globalNetworkID string) error {
	input := &networkmanager.DeleteGlobalNetworkInput{
		GlobalNetworkId: aws.String(globalNetworkID),
	}

	_, err := client.DeleteGlobalNetwork(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, globalNetworkSrc))
		} else {
			return err
		}
	}

	return nil
}

func deleteSite(ctx context.Context, client *networkmanager.Client, globalNetworkID, siteID *string) error {
	input := &networkmanager.DeleteSiteInput{
		GlobalNetworkId: globalNetworkID,
		SiteId:          siteID,
	}

	_, err := client.DeleteSite(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, siteSrc))
		} else {
			return err
		}
	}

	return nil
}

func deleteLink(ctx context.Context, client *networkmanager.Client, globalNetworkID, linkID *string) error {
	input := &networkmanager.DeleteLinkInput{
		GlobalNetworkId: globalNetworkID,
		LinkId:          linkID,
	}

	_, err := client.DeleteLink(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, linkSrc))
		} else {
			return err
		}
	}

	return nil
}

func deleteDevice(ctx context.Context, client *networkmanager.Client, globalNetworkID, deviceID *string) error {
	input := &networkmanager.DeleteDeviceInput{
		GlobalNetworkId: globalNetworkID,
		DeviceId:        deviceID,
	}

	_, err := client.DeleteDevice(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, deviceSrc))
		} else {
			return err
		}

	}

	return nil
}

func deleteLinkAssociation(ctx context.Context, client *networkmanager.Client, globalNetworkID, deviceID, linkID *string) error {
	input := &networkmanager.DisassociateLinkInput{
		GlobalNetworkId: globalNetworkID,
		DeviceId:        deviceID,
		LinkId:          linkID,
	}

	_, err := client.DisassociateLink(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, linkAssociationSrc))
		} else {
			return err
		}

	}

	return nil
}

func deleteConnection(ctx context.Context, client *networkmanager.Client, globalNetworkID, connectionID *string) error {
	input := &networkmanager.DeleteConnectionInput{
		GlobalNetworkId: globalNetworkID,
		ConnectionId:    connectionID,
	}

	_, err := client.DeleteConnection(ctx, input)
	if err != nil {
		var apiErr smithy.APIError
		notFoundException := types.ResourceNotFoundException{}
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == notFoundException.ErrorCode() {
			return integration.NewNotFoundError(integration.ResourceName(integration.NetworkManager, connectionSrc))
		} else {
			return err
		}
	}

	return nil
}
