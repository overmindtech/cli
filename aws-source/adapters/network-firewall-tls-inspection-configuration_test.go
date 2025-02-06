package adapters

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/networkfirewall"
	"github.com/aws/aws-sdk-go-v2/service/networkfirewall/types"

	"github.com/overmindtech/cli/aws-source/adapterhelpers"
	"github.com/overmindtech/cli/sdp-go"
)

func (c testNetworkFirewallClient) DescribeTLSInspectionConfiguration(ctx context.Context, params *networkfirewall.DescribeTLSInspectionConfigurationInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.DescribeTLSInspectionConfigurationOutput, error) {
	now := time.Now()
	return &networkfirewall.DescribeTLSInspectionConfigurationOutput{
		TLSInspectionConfigurationResponse: &types.TLSInspectionConfigurationResponse{
			TLSInspectionConfigurationArn:  adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:tls-inspection-configuration/aws-network-firewall-DefaultTLSInspectionConfiguration-1J3Z3W2ZQXV3"),
			TLSInspectionConfigurationId:   adapterhelpers.PtrString("test"),
			TLSInspectionConfigurationName: adapterhelpers.PtrString("test"),
			CertificateAuthority: &types.TlsCertificateData{
				CertificateArn:    adapterhelpers.PtrString("arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"), // link
				CertificateSerial: adapterhelpers.PtrString("test"),
				Status:            adapterhelpers.PtrString("OK"),
				StatusMessage:     adapterhelpers.PtrString("test"),
			},
			Certificates: []types.TlsCertificateData{
				{
					CertificateArn:    adapterhelpers.PtrString("arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"), // link
					CertificateSerial: adapterhelpers.PtrString("test"),
					Status:            adapterhelpers.PtrString("OK"),
					StatusMessage:     adapterhelpers.PtrString("test"),
				},
			},
			Description: adapterhelpers.PtrString("test"),
			EncryptionConfiguration: &types.EncryptionConfiguration{
				Type:  types.EncryptionTypeAwsOwnedKmsKey,
				KeyId: adapterhelpers.PtrString("arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"), // link (this can be an ARN or ID)
			},
			LastModifiedTime:                 &now,
			NumberOfAssociations:             adapterhelpers.PtrInt32(1),
			TLSInspectionConfigurationStatus: types.ResourceStatusActive, // health
			Tags: []types.Tag{
				{
					Key:   adapterhelpers.PtrString("test"),
					Value: adapterhelpers.PtrString("test"),
				},
			},
		},
		TLSInspectionConfiguration: &types.TLSInspectionConfiguration{
			ServerCertificateConfigurations: []types.ServerCertificateConfiguration{
				{
					CertificateAuthorityArn: adapterhelpers.PtrString("arn:aws:acm:us-east-1:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012"), // link
					CheckCertificateRevocationStatus: &types.CheckCertificateRevocationStatusActions{
						RevokedStatusAction: types.RevocationCheckActionPass,
						UnknownStatusAction: types.RevocationCheckActionPass,
					},
					Scopes: []types.ServerCertificateScope{
						{
							DestinationPorts: []types.PortRange{
								{
									FromPort: 1,
									ToPort:   1,
								},
							},
							Destinations: []types.Address{
								{
									AddressDefinition: adapterhelpers.PtrString("test"),
								},
							},
							Protocols: []int32{1},
							SourcePorts: []types.PortRange{
								{
									FromPort: 1,
									ToPort:   1,
								},
							},
							Sources: []types.Address{
								{
									AddressDefinition: adapterhelpers.PtrString("test"),
								},
							},
						},
					},
					ServerCertificates: []types.ServerCertificate{
						{
							ResourceArn: adapterhelpers.PtrString("arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012"), // link
						},
					},
				},
			},
		},
	}, nil
}

func (c testNetworkFirewallClient) ListTLSInspectionConfigurations(ctx context.Context, params *networkfirewall.ListTLSInspectionConfigurationsInput, optFns ...func(*networkfirewall.Options)) (*networkfirewall.ListTLSInspectionConfigurationsOutput, error) {
	return &networkfirewall.ListTLSInspectionConfigurationsOutput{
		TLSInspectionConfigurations: []types.TLSInspectionConfigurationMetadata{
			{
				Arn: adapterhelpers.PtrString("arn:aws:network-firewall:us-east-1:123456789012:tls-inspection-configuration/aws-network-firewall-DefaultTLSInspectionConfiguration-1J3Z3W2ZQXV3"),
			},
		},
	}, nil
}

func TestTLSInspectionConfigurationGetFunc(t *testing.T) {
	item, err := tlsInspectionConfigurationGetFunc(context.Background(), testNetworkFirewallClient{}, "test", &networkfirewall.DescribeTLSInspectionConfigurationInput{})

	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Fatal(err)
	}

	tests := adapterhelpers.QueryTests{
		{
			ExpectedType:   "acm-pca-certificate-authority-certificate",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "acm-certificate",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:acm:us-east-1:123456789012:certificate/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "acm-pca-certificate-authority",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:acm:us-east-1:123456789012:certificate-authority/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
		{
			ExpectedType:   "kms-key",
			ExpectedMethod: sdp.QueryMethod_SEARCH,
			ExpectedQuery:  "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012",
			ExpectedScope:  "123456789012.us-east-1",
		},
	}

	tests.Execute(t, item)
}
