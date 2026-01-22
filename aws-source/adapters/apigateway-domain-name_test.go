package adapters

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/overmindtech/cli/sdp-go"
)

/*
{
   "certificateArn": "string",
   "certificateName": "string",
   "certificateUploadDate": number,
   "distributionDomainName": "string",
   "distributionHostedZoneId": "string",
   "domainName": "string",
   "domainNameStatus": "string",
   "domainNameStatusMessage": "string",
   "endpointConfiguration": {
      "types": [ "string" ],
      "vpcEndpointIds": [ "string" ]
   },
   "mutualTlsAuthentication": {
      "truststoreUri": "string",
      "truststoreVersion": "string",
      "truststoreWarnings": [ "string" ]
   },
   "ownershipVerificationCertificateArn": "string",
   "regionalCertificateArn": "string",
   "regionalCertificateName": "string",
   "regionalDomainName": "string",
   "regionalHostedZoneId": "string",
   "securityPolicy": "string",
   "tags": {
      "string" : "string"
   }
}
*/

func TestDomainNameOutputMapper(t *testing.T) {
	domainName := &types.DomainName{
		CertificateArn:                      PtrString("arn:aws:acm:region:account-id:certificate/certificate-id"),
		CertificateName:                     PtrString("certificate-name"),
		CertificateUploadDate:               PtrTime(time.Now()),
		DistributionDomainName:              PtrString("distribution-domain-name"),
		DistributionHostedZoneId:            PtrString("distribution-hosted-zone-id"),
		DomainName:                          PtrString("domain-name"),
		DomainNameStatus:                    types.DomainNameStatusAvailable,
		DomainNameStatusMessage:             PtrString("status-message"),
		EndpointConfiguration:               &types.EndpointConfiguration{Types: []types.EndpointType{types.EndpointTypeEdge}},
		MutualTlsAuthentication:             &types.MutualTlsAuthentication{TruststoreUri: PtrString("truststore-uri")},
		OwnershipVerificationCertificateArn: PtrString("arn:aws:acm:region:account-id:certificate/ownership-verification-certificate-id"),
		RegionalCertificateArn:              PtrString("arn:aws:acm:region:account-id:certificate/regional-certificate-id"),
		RegionalCertificateName:             PtrString("regional-certificate-name"),
		RegionalDomainName:                  PtrString("regional-domain-name"),
		RegionalHostedZoneId:                PtrString("regional-hosted-zone-id"),
		SecurityPolicy:                      types.SecurityPolicyTls12,
		Tags:                                map[string]string{"key": "value"},
	}

	item, err := domainNameOutputMapper("domain-name", "scope", domainName)
	if err != nil {
		t.Fatal(err)
	}

	if err := item.Validate(); err != nil {
		t.Error(err)
	}

	a, err := ParseARN("arn:aws:acm:region:account-id:certificate/regional-certificate-id")
	if err != nil {
		t.Fatal(err)
	}

	tests := QueryTests{
		{
			ExpectedType:   "acm-certificate",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "arn:aws:acm:region:account-id:certificate/certificate-id",
			ExpectedScope:  FormatScope(a.AccountID, a.Region),
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "distribution-hosted-zone-id",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "route53-hosted-zone",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "regional-hosted-zone-id",
			ExpectedScope:  "scope",
		},
		{
			ExpectedType:   "acm-certificate",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "arn:aws:acm:region:account-id:certificate/regional-certificate-id",
			ExpectedScope:  FormatScope(a.AccountID, a.Region),
		},
		{
			ExpectedType:   "acm-certificate",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "arn:aws:acm:region:account-id:certificate/ownership-verification-certificate-id",
			ExpectedScope:  FormatScope(a.AccountID, a.Region),
		},
		{
			ExpectedType:   "apigateway-domain-name",
			ExpectedMethod: sdp.QueryMethod_GET,
			ExpectedQuery:  "regional-domain-name",
			ExpectedScope:  "scope",
		},
	}

	tests.Execute(t, item)
}

func TestNewAPIGatewayDomainNameAdapter(t *testing.T) {
	config, account, region := GetAutoConfig(t)

	client := apigateway.NewFromConfig(config)

	adapter := NewAPIGatewayDomainNameAdapter(client, account, region, nil)

	test := E2ETest{
		Adapter: adapter,
		Timeout: 10 * time.Second,
	}

	test.Run(t)
}
