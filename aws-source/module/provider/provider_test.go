package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	sdp "github.com/overmindtech/cli/go/sdp-go"
	"github.com/overmindtech/cli/go/sdp-go/sdpconnect"
	"golang.org/x/oauth2"
)

// --- mock ManagementService handler ---

type mockMgmtHandler struct {
	sdpconnect.UnimplementedManagementServiceHandler
	mu         sync.Mutex
	sources    map[string]*sdp.Source
	externalID string
}

func newMockMgmtHandler() *mockMgmtHandler {
	return &mockMgmtHandler{
		sources:    make(map[string]*sdp.Source),
		externalID: "test-external-id-12345",
	}
}

func (m *mockMgmtHandler) GetOrCreateAWSExternalId(_ context.Context, _ *connect.Request[sdp.GetOrCreateAWSExternalIdRequest]) (*connect.Response[sdp.GetOrCreateAWSExternalIdResponse], error) {
	return connect.NewResponse(&sdp.GetOrCreateAWSExternalIdResponse{
		AwsExternalId: m.externalID,
	}), nil
}

func (m *mockMgmtHandler) CreateSource(_ context.Context, req *connect.Request[sdp.CreateSourceRequest]) (*connect.Response[sdp.CreateSourceResponse], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := uuid.New()
	source := &sdp.Source{
		Metadata:   &sdp.SourceMetadata{UUID: id[:]},
		Properties: req.Msg.GetProperties(),
	}
	m.sources[id.String()] = source
	return connect.NewResponse(&sdp.CreateSourceResponse{Source: source}), nil
}

func (m *mockMgmtHandler) GetSource(_ context.Context, req *connect.Request[sdp.GetSourceRequest]) (*connect.Response[sdp.GetSourceResponse], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.FromBytes(req.Msg.GetUUID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	source, ok := m.sources[id.String()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&sdp.GetSourceResponse{Source: source}), nil
}

func (m *mockMgmtHandler) UpdateSource(_ context.Context, req *connect.Request[sdp.UpdateSourceRequest]) (*connect.Response[sdp.UpdateSourceResponse], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.FromBytes(req.Msg.GetUUID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	source, ok := m.sources[id.String()]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	source.Properties = req.Msg.GetProperties()
	return connect.NewResponse(&sdp.UpdateSourceResponse{Source: source}), nil
}

func (m *mockMgmtHandler) DeleteSource(_ context.Context, req *connect.Request[sdp.DeleteSourceRequest]) (*connect.Response[sdp.DeleteSourceResponse], error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, err := uuid.FromBytes(req.Msg.GetUUID())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if _, ok := m.sources[id.String()]; !ok {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	delete(m.sources, id.String())
	return connect.NewResponse(&sdp.DeleteSourceResponse{}), nil
}

// --- test provider that bypasses auth ---

// testProvider wraps the real provider but overrides Configure to inject a
// pre-built client backed by the mock server. This avoids needing the
// instance-data endpoint, ApiKeyService, or real JWTs in unit tests.
type testProvider struct {
	overmindProvider
	serverURL string
}

var _ provider.Provider = (*testProvider)(nil)

func (p *testProvider) Configure(ctx context.Context, _ provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	httpClient := oauth2.NewClient(ctx,
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"}))
	mgmtClient := sdpconnect.NewManagementServiceClient(httpClient, p.serverURL)
	resp.DataSourceData = mgmtClient
	resp.ResourceData = mgmtClient
}

func (p *testProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{},
	}
}

func (p *testProvider) Resources(ctx context.Context) []func() resource.Resource {
	return p.overmindProvider.Resources(ctx)
}

func (p *testProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return p.overmindProvider.DataSources(ctx)
}

// --- test helpers ---

func startTestServer(t *testing.T) string {
	t.Helper()
	handler := newMockMgmtHandler()
	path, h := sdpconnect.NewManagementServiceHandler(handler)
	mux := http.NewServeMux()
	mux.Handle(path, h)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL
}

func unitTestProviderFactories(serverURL string) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"overmind": providerserver.NewProtocol6WithError(&testProvider{
			overmindProvider: overmindProvider{version: "test"},
			serverURL:        serverURL,
		}),
	}
}

func accTestProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"overmind": providerserver.NewProtocol6WithError(NewProvider("test")()),
	}
}

// --- unit tests (mock server, always run) ---

func TestAWSSourceResource_CRUD(t *testing.T) {
	serverURL := startTestServer(t)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories(serverURL),
		Steps: []tfresource.TestStep{
			{
				Config: testAccAWSSourceConfig("test-source", "arn:aws:iam::123456789012:role/test", `["us-east-1", "eu-west-1"]`),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttrSet("overmind_aws_source.test", "id"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "name", "test-source"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "aws_role_arn", "arn:aws:iam::123456789012:role/test"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "external_id", "test-external-id-12345"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "aws_regions.#", "2"),
				),
			},
			{
				Config: testAccAWSSourceConfig("updated-source", "arn:aws:iam::123456789012:role/test", `["us-west-2"]`),
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "name", "updated-source"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "aws_regions.#", "1"),
					tfresource.TestCheckResourceAttr("overmind_aws_source.test", "aws_regions.0", "us-west-2"),
				),
			},
			{
				ResourceName:      "overmind_aws_source.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestProviderConfigure_MissingAPIKey(t *testing.T) {
	t.Setenv("OVERMIND_API_KEY", "")
	t.Setenv("OVERMIND_APP_URL", "")

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: accTestProviderFactories(),
		Steps: []tfresource.TestStep{
			{
				Config: `
resource "overmind_aws_source" "test" {
  name         = "x"
  aws_role_arn = "arn"
  aws_regions  = ["us-east-1"]
}
`,
				ExpectError: regexp.MustCompile(`Missing API Key`),
			},
		},
	})
}

func TestAWSExternalIdDataSource_Read(t *testing.T) {
	serverURL := startTestServer(t)

	tfresource.UnitTest(t, tfresource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories(serverURL),
		Steps: []tfresource.TestStep{
			{
				Config: `data "overmind_aws_external_id" "test" {}`,
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(
						"data.overmind_aws_external_id.test", "external_id", "test-external-id-12345"),
				),
			},
		},
	})
}

func testAccAWSSourceConfig(name, roleARN, regions string) string {
	return `resource "overmind_aws_source" "test" {
  name         = "` + name + `"
  aws_role_arn = "` + roleARN + `"
  aws_regions  = ` + regions + `
}`
}
