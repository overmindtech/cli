module github.com/overmindtech/cli

go 1.26.0

replace github.com/anthropics/anthropic-sdk-go => github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.4

// Address an incompatibility between buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go and the kubernetes modules.
// See https://github.com/overmindtech/workspace/pull/1124 and https://github.com/kubernetes/apiserver/issues/116
replace github.com/google/cel-go => github.com/google/cel-go v0.22.1

require (
	atomicgo.dev/keyboard v0.2.9
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260209202127-80ab13bee0bf.1
	buf.build/go/protovalidate v1.1.3
	charm.land/lipgloss/v2 v2.0.2
	cloud.google.com/go/aiplatform v1.123.0
	cloud.google.com/go/auth v0.20.0
	cloud.google.com/go/auth/oauth2adapt v0.2.8
	cloud.google.com/go/bigquery v1.75.0
	cloud.google.com/go/bigtable v1.46.0
	cloud.google.com/go/certificatemanager v1.11.0
	cloud.google.com/go/compute v1.59.0
	cloud.google.com/go/compute/metadata v0.9.0
	cloud.google.com/go/container v1.48.0
	cloud.google.com/go/dataplex v1.31.0
	cloud.google.com/go/dataproc/v2 v2.18.0
	cloud.google.com/go/eventarc v1.20.0
	cloud.google.com/go/filestore v1.12.0
	cloud.google.com/go/functions v1.21.0
	cloud.google.com/go/iam v1.8.0
	cloud.google.com/go/kms v1.28.0
	cloud.google.com/go/logging v1.15.0
	cloud.google.com/go/monitoring v1.26.0
	cloud.google.com/go/networksecurity v0.13.0
	cloud.google.com/go/orgpolicy v1.17.0
	cloud.google.com/go/redis v1.20.0
	cloud.google.com/go/resourcemanager v1.12.0
	cloud.google.com/go/run v1.18.0
	cloud.google.com/go/secretmanager v1.18.0
	cloud.google.com/go/securitycentermanagement v1.3.0
	cloud.google.com/go/spanner v1.89.0
	cloud.google.com/go/storage v1.62.0
	cloud.google.com/go/storagetransfer v1.15.0
	connectrpc.com/connect v1.18.1 // v1.19.0 was faulty, wait until it is above this version
	connectrpc.com/otelconnect v0.9.0
	github.com/1password/onepassword-sdk-go v0.4.0
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3 v3.0.0-beta.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v4 v4.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7 v7.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos/v3 v3.4.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/elasticsan/armelasticsan v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2 v2.0.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/maintenance/armmaintenance v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9 v9.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5 v5.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2 v2.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2 v2.0.0-beta.7
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3 v3.0.0
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/MrAlias/otel-schema-utils v0.4.0-alpha
	github.com/a-h/templ v0.3.1001
	github.com/adrg/strutil v0.3.1
	github.com/akedrou/textdiff v0.1.0
	github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.4
	github.com/antihax/optional v1.0.0
	github.com/auth0/go-auth0/v2 v2.8.0
	github.com/auth0/go-jwt-middleware/v3 v3.1.0
	github.com/aws/aws-sdk-go-v2 v1.41.5
	github.com/aws/aws-sdk-go-v2/config v1.32.14
	github.com/aws/aws-sdk-go-v2/credentials v1.19.14
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.21
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.39.1
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.65.0
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.61.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.56.0
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.15
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.57.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.297.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.77.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.14
	github.com/aws/aws-sdk-go-v2/service/eks v1.82.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.23
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.10
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.7
	github.com/aws/aws-sdk-go-v2/service/kms v1.50.4
	github.com/aws/aws-sdk-go-v2/service/lambda v1.89.0
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.59.7
	github.com/aws/aws-sdk-go-v2/service/networkmanager v1.41.8
	github.com/aws/aws-sdk-go-v2/service/rds v1.117.1
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.5
	github.com/aws/aws-sdk-go-v2/service/s3 v1.99.0
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.60.2
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.15
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.25
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.4
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.10
	github.com/aws/smithy-go v1.24.3
	github.com/bombsimon/logrusr/v4 v4.1.0
	github.com/bradleyfalzon/ghinstallation/v2 v2.18.0
	github.com/brianvoe/gofakeit/v7 v7.14.1
	github.com/cenkalti/backoff/v5 v5.0.3
	github.com/charmbracelet/glamour v0.10.0
	github.com/coder/websocket v1.8.14
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/getsentry/sentry-go v0.45.0
	github.com/go-jose/go-jose/v4 v4.1.4
	github.com/gocarina/gocsv v0.0.0-20240520201108-78e41c74b4b1
	github.com/google/btree v1.1.3
	github.com/google/go-github/v84 v84.0.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.21.0
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/gorilla/mux v1.8.1
	github.com/harness/harness-go-sdk v0.7.18
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/hashicorp/terraform-config-inspect v0.0.0-20260224005459-813a97530220
	github.com/hashicorp/terraform-plugin-framework v1.19.0
	github.com/hashicorp/terraform-plugin-go v0.31.0
	github.com/hashicorp/terraform-plugin-testing v1.15.0
	github.com/invopop/jsonschema v0.13.0
	github.com/jackc/pgx/v5 v5.9.1
	github.com/jedib0t/go-pretty/v6 v6.7.9
	github.com/jxskiss/base62 v1.1.0
	github.com/kaptinlin/jsonrepair v0.2.17
	github.com/lithammer/fuzzysearch v1.1.8
	github.com/manifoldco/promptui v0.9.0
	github.com/mavolin/go-htmx v1.0.0
	github.com/mergestat/timediff v0.0.4
	github.com/micahhausler/aws-iam-policy v0.4.4
	github.com/miekg/dns v1.1.72
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-ps v1.0.0
	github.com/modelcontextprotocol/go-sdk v1.5.0
	github.com/muesli/reflow v0.3.0
	github.com/nats-io/jwt/v2 v2.8.1
	github.com/nats-io/nats-server/v2 v2.12.6
	github.com/nats-io/nats.go v1.50.0
	github.com/nats-io/nkeys v0.4.15
	github.com/neo4j/neo4j-go-driver/v6 v6.0.0
	github.com/onsi/ginkgo/v2 v2.28.1
	github.com/onsi/gomega v1.39.1
	github.com/openai/openai-go/v3 v3.31.0
	github.com/openrdap/rdap v0.9.2-0.20240517203139-eb57b3a8dedd
	github.com/overmindtech/otelpgx v0.10.1-0.20260303210427-65bf1016045e
	github.com/overmindtech/pterm v0.0.0-20240919144758-04d94ccb2297
	github.com/pborman/ansi v1.0.0
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/posthog/posthog-go v1.11.2
	github.com/projectdiscovery/subfinder/v2 v2.13.0
	github.com/qhenkart/anthropic-tokenizer-go v0.0.0-20231011194518-5519949e0faf
	github.com/resend/resend-go/v3 v3.3.0
	github.com/riverqueue/river v0.34.0
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.34.0
	github.com/riverqueue/river/rivertype v0.34.0
	github.com/riverqueue/rivercontrib/otelriver v0.7.0
	github.com/rs/cors v1.11.1
	github.com/samber/slog-logrus/v2 v2.5.4
	github.com/sashabaranov/go-openai v1.41.2
	github.com/serpapi/serpapi-golang v0.0.0-20260126142127-0e41c7993cda
	github.com/sirupsen/logrus v1.9.4
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v84 v84.4.1
	github.com/tiktoken-go/tokenizer v0.7.0
	github.com/ttacon/chalk v0.0.0-20160626202418-22c06c80ed31
	github.com/uptrace/opentelemetry-go-extra/otellogrus v0.3.2
	github.com/wk8/go-ordered-map/v2 v2.1.8
	github.com/xiam/dig v0.0.0-20191116195832-893b5fb5093b
	github.com/zclconf/go-cty v1.18.0
	go.etcd.io/bbolt v1.4.3
	go.opentelemetry.io/contrib/detectors/aws/ec2/v2 v2.0.0-20250901115419-474a7992e57c
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.43.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.43.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.43.0
	go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/trace v1.43.0
	go.uber.org/automaxprocs v1.6.0
	go.uber.org/goleak v1.3.0
	go.uber.org/mock v0.6.0
	go.yaml.in/yaml/v3 v3.0.4
	golang.org/x/net v0.53.0
	golang.org/x/oauth2 v0.36.0
	golang.org/x/sync v0.20.0
	golang.org/x/text v0.36.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.275.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260406210006-6f92a3bedf2d
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/ini.v1 v1.67.1
	k8s.io/api v0.35.3
	k8s.io/apimachinery v0.35.3
	k8s.io/client-go v0.35.3
	k8s.io/component-base v0.35.3
	modernc.org/sqlite v1.48.2
	riverqueue.com/riverui v0.15.0
	sigs.k8s.io/controller-runtime v0.23.3
	sigs.k8s.io/kind v0.31.0
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2
)

require (
	aead.dev/minisign v0.2.0 // indirect
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/longrunning v0.9.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/internal/v3 v3.1.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v3 v3.0.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/detectors/gcp v1.31.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/metric v0.55.0 // indirect
	github.com/GoogleCloudPlatform/opentelemetry-operations-go/internal/resourcemapping v0.55.0 // indirect
	github.com/Mzack9999/gcache v0.0.0-20230410081825-519e28eab057 // indirect
	github.com/Mzack9999/go-http-digest-auth-client v0.6.1-0.20220414142836-eb8883508809 // indirect
	github.com/ProtonMail/go-crypto v1.3.0 // indirect
	github.com/PuerkitoBio/rehttp v1.4.0 // indirect
	github.com/STARRY-S/zip v0.2.1 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/akrylysov/pogreb v0.10.1 // indirect
	github.com/alecthomas/chroma/v2 v2.16.0 // indirect
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.6.0-default-no-op // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.8 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.21 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.6 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.22 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.13 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.19 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bodgit/plumbing v1.3.0 // indirect
	github.com/bodgit/sevenzip v1.6.0 // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.2 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.2 // indirect
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834 // indirect; being pulled by glamour, this will be resolved in https://github.com/charmbracelet/glamour/pull/408
	github.com/charmbracelet/ultraviolet v0.0.0-20251205161215-1948445e3318 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250417172821-98fd948af1b1 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/charmbracelet/x/termios v0.1.1 // indirect
	github.com/charmbracelet/x/windows v0.2.2 // indirect
	github.com/cheggaaa/pb/v3 v3.1.4 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/cncf/xds/go v0.0.0-20251210132809-ee656c7534f5 // indirect
	github.com/cnf/structhash v0.0.0-20201127153200-e1b16c1ebc08 // indirect
	github.com/containerd/console v1.0.4 // indirect
	github.com/corpix/uarand v0.2.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20230904184137-39efe44ab707 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/dylibso/observe-sdk/go v0.0.0-20240828172851-9145d8ad07e1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.37.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v1.3.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/extism/go-sdk v1.7.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gaissmai/bart v0.26.0 // indirect
	github.com/go-json-experiment/json v0.0.0-20251027170946-4849db3c2f7e // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.1 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.30.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/cel-go v0.27.0 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-github/v30 v30.1.0 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.14 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.28.0 // indirect
	github.com/hako/durafmt v0.0.0-20210316092057-3a2c319c1acd // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hc-install v0.9.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.25.0 // indirect
	github.com/hashicorp/terraform-json v0.27.2 // indirect
	github.com/hashicorp/terraform-plugin-log v0.10.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.40.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.4.0 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/ianlancetaylor/demangle v0.0.0-20251118225945-96ee0021ea0f // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.5 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.4 // indirect
	github.com/lestrrat-go/dsig v1.0.0 // indirect
	github.com/lestrrat-go/dsig-secp256k1 v1.0.0 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/httprc/v3 v3.0.3 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lestrrat-go/jwx/v3 v3.0.13 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lestrrat-go/option/v2 v2.0.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/logrusorgru/aurora v2.0.3+incompatible // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mholt/archives v0.1.0 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/minio/highwayhash v1.0.4-0.20251030100505-070ab1a87a76 // indirect
	github.com/minio/selfupdate v0.6.1-0.20230907112617-f11e74f84ca7 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/nwaples/rardecode/v2 v2.2.0 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/projectdiscovery/blackrock v0.0.1 // indirect
	github.com/projectdiscovery/cdncheck v1.2.13 // indirect
	github.com/projectdiscovery/chaos-client v0.5.2 // indirect
	github.com/projectdiscovery/dnsx v1.2.3 // indirect
	github.com/projectdiscovery/fastdialer v0.4.19 // indirect
	github.com/projectdiscovery/goflags v0.1.74 // indirect
	github.com/projectdiscovery/gologger v1.1.62 // indirect
	github.com/projectdiscovery/hmap v0.0.98 // indirect
	github.com/projectdiscovery/machineid v0.0.0-20240226150047-2e2c51e35983 // indirect
	github.com/projectdiscovery/networkpolicy v0.1.31 // indirect
	github.com/projectdiscovery/ratelimit v0.0.82 // indirect
	github.com/projectdiscovery/retryabledns v1.0.111 // indirect
	github.com/projectdiscovery/retryablehttp-go v1.1.0 // indirect
	github.com/projectdiscovery/utils v0.7.3 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/refraction-networking/utls v1.8.2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/riverqueue/apiframe v0.0.0-20251229202423-2b52ce1c482e // indirect
	github.com/riverqueue/river/riverdriver v0.34.0 // indirect
	github.com/riverqueue/river/rivershared v0.34.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/slog-common v0.21.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.4 // indirect
	github.com/shirou/gopsutil/v3 v3.23.7 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sorairolake/lzip-go v0.3.5 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spiffe/go-spiffe/v2 v2.6.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/t-tomalak/logrus-easy-formatter v0.0.0-20190827215021-c074f06c5816 // indirect
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/therootcompany/xz v1.0.1 // indirect
	github.com/tidwall/btree v1.6.0 // indirect
	github.com/tidwall/buntdb v1.3.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/grect v0.1.4 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/rtred v0.1.2 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tidwall/tinyqueue v0.1.1 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	github.com/ulikunitz/xz v0.5.15 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelutil v0.3.2 // indirect
	github.com/valyala/fastjson v1.6.7 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/weppos/publicsuffix-go v0.40.3-0.20250408071509-6074bbe7fd39 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xiam/to v0.0.0-20191116183551-8328998fc0ed // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	github.com/yuin/goldmark v1.7.10 // indirect
	github.com/yuin/goldmark-emoji v1.0.5 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zcalusic/sysinfo v1.0.2 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	github.com/zmap/rc2 v0.0.0-20190804163417-abaa70531248 // indirect
	github.com/zmap/zcrypto v0.0.0-20230422215203-9a665e1e9968 // indirect
	go.devnw.com/structs v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/detectors/gcp v1.39.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.67.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.43.0 // indirect
	go.opentelemetry.io/otel/schema v0.0.12 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.43.0 // indirect
	go.opentelemetry.io/proto/otlp v1.10.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/crypto v0.50.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/mod v0.34.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/telemetry v0.0.0-20260311193753-579e4da9a98c // indirect
	golang.org/x/term v0.42.0 // indirect
	golang.org/x/time v0.15.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto v0.0.0-20260319201613-d00831a3d3e7 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260401024825-9d38bb4040a9 // indirect
	gopkg.in/djherbis/times.v1 v1.3.0 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apiextensions-apiserver v0.35.0 // indirect
	k8s.io/apiserver v0.35.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	k8s.io/utils v0.0.0-20260210185600-b8788abfbbc2 // indirect
	modernc.org/libc v1.70.0 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.32.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/yaml v1.6.0 // indirect
)
