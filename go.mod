module github.com/overmindtech/cli

go 1.25.1

replace github.com/anthropics/anthropic-sdk-go => github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.4

// Address an incompatibility between buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go and the kubernetes modules.
// See https://github.com/overmindtech/workspace/pull/1124 and https://github.com/kubernetes/apiserver/issues/116
replace github.com/google/cel-go => github.com/google/cel-go v0.22.1

require (
	atomicgo.dev/keyboard v0.2.9
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1
	buf.build/go/protovalidate v1.1.0
	cloud.google.com/go/aiplatform v1.114.0
	cloud.google.com/go/auth v0.18.1
	cloud.google.com/go/bigquery v1.72.0
	cloud.google.com/go/bigtable v1.41.0
	cloud.google.com/go/compute v1.54.0
	cloud.google.com/go/compute/metadata v0.9.0
	cloud.google.com/go/container v1.46.0
	cloud.google.com/go/dataplex v1.28.0
	cloud.google.com/go/dataproc/v2 v2.15.0
	cloud.google.com/go/filestore v1.10.3
	cloud.google.com/go/functions v1.19.7
	cloud.google.com/go/iam v1.5.3
	cloud.google.com/go/kms v1.25.0
	cloud.google.com/go/logging v1.13.1
	cloud.google.com/go/monitoring v1.24.3
	cloud.google.com/go/networksecurity v0.11.0
	cloud.google.com/go/orgpolicy v1.15.1
	cloud.google.com/go/redis v1.18.3
	cloud.google.com/go/resourcemanager v1.10.7
	cloud.google.com/go/run v1.15.0
	cloud.google.com/go/secretmanager v1.16.0
	cloud.google.com/go/securitycentermanagement v1.1.6
	cloud.google.com/go/spanner v1.87.0
	cloud.google.com/go/storagetransfer v1.13.1
	connectrpc.com/connect v1.18.1 // v1.19.0 was faulty, wait until it is above this version
	connectrpc.com/otelconnect v0.9.0
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.21.0
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.13.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3 v3.0.0-beta.2
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/batch/armbatch/v3 v3.0.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v7 v7.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cosmos/armcosmos v1.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns v1.2.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault/v2 v2.0.1
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/msi/armmsi v1.3.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v8 v8.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/postgresql/armpostgresqlflexibleservers/v5 v5.0.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v2 v2.1.0
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/sql/armsql/v2 v2.0.0-beta.7
	github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage/v3 v3.0.0
	github.com/Masterminds/semver/v3 v3.4.0
	github.com/MrAlias/otel-schema-utils v0.4.0-alpha
	github.com/a-h/templ v0.3.977
	github.com/adrg/strutil v0.3.1
	github.com/akedrou/textdiff v0.1.0
	github.com/anthropics/anthropic-sdk-go v0.2.0-alpha.4
	github.com/antihax/optional v1.0.0
	github.com/auth0/go-auth0/v2 v2.4.0
	github.com/auth0/go-jwt-middleware/v2 v2.3.1
	github.com/aws/aws-sdk-go-v2 v1.41.1
	github.com/aws/aws-sdk-go-v2/config v1.32.7
	github.com/aws/aws-sdk-go-v2/credentials v1.19.7
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.17
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.38.4
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.64.0
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.59.0
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.53.1
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.38.11
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.54.0
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.285.0
	github.com/aws/aws-sdk-go-v2/service/ecs v1.71.0
	github.com/aws/aws-sdk-go-v2/service/efs v1.41.10
	github.com/aws/aws-sdk-go-v2/service/eks v1.77.0
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.33.19
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.6
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.2
	github.com/aws/aws-sdk-go-v2/service/kms v1.49.5
	github.com/aws/aws-sdk-go-v2/service/lambda v1.88.0
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.59.3
	github.com/aws/aws-sdk-go-v2/service/networkmanager v1.41.4
	github.com/aws/aws-sdk-go-v2/service/rds v1.114.0
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.1
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.0
	github.com/aws/aws-sdk-go-v2/service/sesv2 v1.59.1
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.11
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.21
	github.com/aws/aws-sdk-go-v2/service/ssm v1.67.8
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.6
	github.com/aws/smithy-go v1.24.0
	github.com/bombsimon/logrusr/v4 v4.1.0
	github.com/bradleyfalzon/ghinstallation/v2 v2.17.0
	github.com/brianvoe/gofakeit/v7 v7.14.0
	github.com/cenkalti/backoff/v5 v5.0.3
	github.com/charmbracelet/glamour v0.10.0
	github.com/charmbracelet/lipgloss/v2 v2.0.0-beta.3
	github.com/coder/websocket v1.8.14
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/exaring/otelpgx v0.10.0
	github.com/getsentry/sentry-go v0.42.0
	github.com/go-jose/go-jose/v4 v4.1.3
	github.com/gocarina/gocsv v0.0.0-20240520201108-78e41c74b4b1
	github.com/google/btree v1.1.3
	github.com/google/go-github/v80 v80.0.0
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.16.0
	github.com/goombaio/namegenerator v0.0.0-20181006234301-989e774b106e
	github.com/gorilla/mux v1.8.1
	github.com/harness/harness-go-sdk v0.7.4
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/hashicorp/hcl/v2 v2.24.0
	github.com/hashicorp/terraform-config-inspect v0.0.0-20260120201749-785479628bd7
	github.com/invopop/jsonschema v0.13.0
	github.com/jackc/pgx/v5 v5.8.0
	github.com/jedib0t/go-pretty/v6 v6.7.8
	github.com/jxskiss/base62 v1.1.0
	github.com/kaptinlin/jsonrepair v0.2.7
	github.com/manifoldco/promptui v0.9.0
	github.com/mavolin/go-htmx v1.0.0
	github.com/mergestat/timediff v0.0.4
	github.com/micahhausler/aws-iam-policy v0.4.2
	github.com/miekg/dns v1.1.72
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/go-ps v1.0.0
	github.com/muesli/reflow v0.3.0
	github.com/nats-io/jwt/v2 v2.8.0
	github.com/nats-io/nats-server/v2 v2.12.4
	github.com/nats-io/nats.go v1.48.0
	github.com/nats-io/nkeys v0.4.12
	github.com/neo4j/neo4j-go-driver/v6 v6.0.0
	github.com/onsi/ginkgo/v2 v2.28.1
	github.com/onsi/gomega v1.39.1
	github.com/openai/openai-go/v3 v3.17.0
	github.com/openrdap/rdap v0.9.2-0.20240517203139-eb57b3a8dedd
	github.com/overmindtech/pterm v0.0.0-20240919144758-04d94ccb2297
	github.com/pborman/ansi v1.0.0
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/posthog/posthog-go v1.9.1
	github.com/projectdiscovery/subfinder/v2 v2.12.0
	github.com/qhenkart/anthropic-tokenizer-go v0.0.0-20231011194518-5519949e0faf
	github.com/riverqueue/river v0.30.2
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.30.2
	github.com/riverqueue/river/rivertype v0.30.2
	github.com/riverqueue/rivercontrib/otelriver v0.7.0
	github.com/rs/cors v1.11.1
	github.com/samber/slog-logrus/v2 v2.5.2
	github.com/sashabaranov/go-openai v1.41.2
	github.com/serpapi/serpapi-golang v0.0.0-20260126142127-0e41c7993cda
	github.com/sirupsen/logrus v1.9.4
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	github.com/stripe/stripe-go/v84 v84.3.0
	github.com/tiktoken-go/tokenizer v0.7.0
	github.com/ttacon/chalk v0.0.0-20160626202418-22c06c80ed31
	github.com/uptrace/opentelemetry-go-extra/otellogrus v0.3.2
	github.com/wk8/go-ordered-map/v2 v2.1.8
	github.com/xiam/dig v0.0.0-20191116195832-893b5fb5093b
	github.com/zclconf/go-cty v1.17.0
	go.etcd.io/bbolt v1.4.3
	go.opentelemetry.io/contrib/detectors/aws/ec2/v2 v2.0.0-20250901115419-474a7992e57c
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.64.0
	go.opentelemetry.io/otel v1.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.39.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.39.0
	go.opentelemetry.io/otel/sdk v1.39.0
	go.opentelemetry.io/otel/trace v1.39.0
	go.uber.org/automaxprocs v1.6.0
	go.uber.org/goleak v1.3.0
	go.uber.org/mock v0.6.0
	golang.org/x/net v0.49.0
	golang.org/x/oauth2 v0.34.0
	golang.org/x/sync v0.19.0
	golang.org/x/text v0.33.0
	gonum.org/v1/gonum v0.17.0
	google.golang.org/api v0.264.0
	google.golang.org/genproto v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260128011058-8636f8732409
	google.golang.org/grpc v1.78.0
	google.golang.org/protobuf v1.36.11
	gopkg.in/ini.v1 v1.67.1
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v0.35.0
	k8s.io/apimachinery v0.35.0
	k8s.io/client-go v0.35.0
	k8s.io/component-base v0.35.0
	k8s.io/utils v0.0.0-20260108192941-914a6e750570
	modernc.org/sqlite v1.44.3
	riverqueue.com/riverui v0.14.0
	sigs.k8s.io/controller-runtime v0.23.1
	sigs.k8s.io/kind v0.31.0
)

require (
	aead.dev/minisign v0.2.0 // indirect
	al.essio.dev/pkg/shellescape v1.5.1 // indirect
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/longrunning v0.8.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.11.2 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.6.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/Mzack9999/gcache v0.0.0-20230410081825-519e28eab057 // indirect
	github.com/Mzack9999/go-http-digest-auth-client v0.6.1-0.20220414142836-eb8883508809 // indirect
	github.com/PuerkitoBio/rehttp v1.4.0 // indirect
	github.com/STARRY-S/zip v0.2.1 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/akrylysov/pogreb v0.10.1 // indirect
	github.com/alecthomas/chroma/v2 v2.16.0 // indirect
	github.com/alecthomas/kingpin/v2 v2.4.0 // indirect
	github.com/alecthomas/units v0.0.0-20240927000941-0f3dac36c52b // indirect
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/antithesishq/antithesis-sdk-go v0.5.0-default-no-op // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/apache/arrow/go/v15 v15.0.2 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.13 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bodgit/plumbing v1.3.0 // indirect
	github.com/bodgit/sevenzip v1.6.0 // indirect
	github.com/bodgit/windows v1.0.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.3.1 // indirect
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834 // indirect; being pulled by glamour, this will be resolved in https://github.com/charmbracelet/glamour/pull/408
	github.com/charmbracelet/x/ansi v0.8.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.13 // indirect
	github.com/charmbracelet/x/exp/slice v0.0.0-20250417172821-98fd948af1b1 // indirect
	github.com/charmbracelet/x/term v0.2.1 // indirect
	github.com/cheggaaa/pb/v3 v3.1.4 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/cnf/structhash v0.0.0-20201127153200-e1b16c1ebc08 // indirect
	github.com/containerd/console v1.0.4 // indirect
	github.com/corpix/uarand v0.2.0 // indirect
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dsnet/compress v0.0.2-0.20230904184137-39efe44ab707 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gaissmai/bart v0.20.4 // indirect
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
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.0 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/cel-go v0.26.1 // indirect
	github.com/google/flatbuffers v23.5.26+incompatible // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/go-github/v30 v30.1.0 // indirect
	github.com/google/go-github/v75 v75.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/pprof v0.0.0-20260115054156-294ebfa9ad83 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.3 // indirect
	github.com/hako/durafmt v0.0.0-20210316092057-3a2c319c1acd // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/golang-lru/v2 v2.0.7 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgerrcode v0.0.0-20250907135507-afb5586c32a6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lestrrat-go/blackmagic v1.0.3 // indirect
	github.com/lestrrat-go/httpcc v1.0.1 // indirect
	github.com/lestrrat-go/httprc v1.0.6 // indirect
	github.com/lestrrat-go/iter v1.0.2 // indirect
	github.com/lestrrat-go/jwx/v2 v2.1.6 // indirect
	github.com/lestrrat-go/option v1.0.1 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/logrusorgru/aurora v2.0.3+incompatible // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mholt/archives v0.1.0 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/minio/highwayhash v1.0.4-0.20251030100505-070ab1a87a76 // indirect
	github.com/minio/selfupdate v0.6.1-0.20230907112617-f11e74f84ca7 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/nwaples/rardecode/v2 v2.2.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkoukk/tiktoken-go v0.1.7 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/projectdiscovery/blackrock v0.0.1 // indirect
	github.com/projectdiscovery/cdncheck v1.1.24 // indirect
	github.com/projectdiscovery/chaos-client v0.5.2 // indirect
	github.com/projectdiscovery/dnsx v1.2.2 // indirect
	github.com/projectdiscovery/fastdialer v0.4.1 // indirect
	github.com/projectdiscovery/goflags v0.1.74 // indirect
	github.com/projectdiscovery/gologger v1.1.54 // indirect
	github.com/projectdiscovery/hmap v0.0.90 // indirect
	github.com/projectdiscovery/machineid v0.0.0-20240226150047-2e2c51e35983 // indirect
	github.com/projectdiscovery/networkpolicy v0.1.16 // indirect
	github.com/projectdiscovery/ratelimit v0.0.81 // indirect
	github.com/projectdiscovery/retryabledns v1.0.102 // indirect
	github.com/projectdiscovery/retryablehttp-go v1.0.115 // indirect
	github.com/projectdiscovery/utils v0.4.21 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/refraction-networking/utls v1.7.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/riverqueue/apiframe v0.0.0-20251229202423-2b52ce1c482e // indirect
	github.com/riverqueue/river/riverdriver v0.30.2 // indirect
	github.com/riverqueue/river/rivershared v0.30.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/samber/lo v1.47.0 // indirect
	github.com/samber/slog-common v0.18.1 // indirect
	github.com/segmentio/asm v1.2.0 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shirou/gopsutil/v3 v3.23.7 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sorairolake/lzip-go v0.3.5 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/syndtr/goleveldb v1.0.0 // indirect
	github.com/t-tomalak/logrus-easy-formatter v0.0.0-20190827215021-c074f06c5816 // indirect
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
	github.com/weppos/publicsuffix-go v0.30.1 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xiam/to v0.0.0-20191116183551-8328998fc0ed // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	github.com/yuin/goldmark v1.7.10 // indirect
	github.com/yuin/goldmark-emoji v1.0.5 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	github.com/zcalusic/sysinfo v1.0.2 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	github.com/zmap/rc2 v0.0.0-20190804163417-abaa70531248 // indirect
	github.com/zmap/zcrypto v0.0.0-20230422215203-9a665e1e9968 // indirect
	go.devnw.com/structs v1.0.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0 // indirect
	go.opentelemetry.io/otel/log v0.11.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/schema v0.0.12 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	go4.org v0.0.0-20230225012048-214862532bf5 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/exp v0.0.0-20251023183803-a4bb9ffd2546 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/telemetry v0.0.0-20260109210033-bd525da824e2 // indirect
	golang.org/x/term v0.39.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	gomodules.xyz/jsonpatch/v2 v2.5.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260122232226-8e98ce8d340d // indirect
	gopkg.in/djherbis/times.v1 v1.3.0 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/apiextensions-apiserver v0.35.0 // indirect
	k8s.io/apiserver v0.35.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	modernc.org/libc v1.67.6 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.11.0 // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.32.0 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.2-0.20260122202528-d9cc6641c482
	sigs.k8s.io/yaml v1.6.0 // indirect
)
