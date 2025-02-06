package apigateway

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/overmindtech/cli/aws-source/adapterhelpers"
)

func deleteRestAPI(ctx context.Context, client *apigateway.Client, restAPIID string) error {
	_, err := client.DeleteRestApi(ctx, &apigateway.DeleteRestApiInput{
		RestApiId: adapterhelpers.PtrString(restAPIID),
	})

	return err
}
