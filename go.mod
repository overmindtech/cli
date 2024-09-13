module github.com/overmindtech/cli

go 1.22.5

require (
	connectrpc.com/connect v1.16.2
	github.com/aws/aws-sdk-go-v2 v1.30.5
	github.com/aws/aws-sdk-go-v2/config v1.27.33
	github.com/aws/aws-sdk-go-v2/credentials v1.17.32
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.13
	github.com/charmbracelet/bubbles v0.20.0
	github.com/charmbracelet/bubbletea v1.1.0
	github.com/charmbracelet/glamour v0.8.0
	github.com/charmbracelet/huh v0.6.0
	github.com/charmbracelet/lipgloss v0.13.0
	github.com/getsentry/sentry-go v0.28.1
	github.com/go-jose/go-jose/v4 v4.0.4
	github.com/google/uuid v1.6.0
	github.com/hashicorp/hcl/v2 v2.22.0
	github.com/hashicorp/terraform-config-inspect v0.0.0-20240801114854-6714b46f5fe4
	github.com/hexops/gotextdiff v1.0.3
	github.com/jedib0t/go-pretty/v6 v6.5.9
	github.com/mattn/go-isatty v0.0.20
	github.com/mitchellh/go-homedir v1.1.0
	github.com/muesli/reflow v0.3.0
	github.com/muesli/termenv v0.15.3-0.20240618155329-98d742f6907a
	github.com/overmindtech/aws-source v0.0.0-20240910143452-74bcfddd91cf
	github.com/overmindtech/sdp-go v0.93.1
	github.com/overmindtech/stdlib-source v0.0.0-20240902171206-a7ba824b1451
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.9.0
	github.com/ttacon/chalk v0.0.0-20160626202418-22c06c80ed31
	github.com/uptrace/opentelemetry-go-extra/otellogrus v0.3.1
	github.com/xiam/dig v0.0.0-20191116195832-893b5fb5093b
	github.com/zclconf/go-cty v1.15.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.54.0
	go.opentelemetry.io/otel v1.29.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.29.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.29.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.29.0
	go.opentelemetry.io/otel/sdk v1.29.0
	go.opentelemetry.io/otel/trace v1.29.0
	golang.org/x/net v0.29.0
	golang.org/x/oauth2 v0.23.0
	google.golang.org/protobuf v1.34.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/alecthomas/chroma/v2 v2.14.0 // indirect
	github.com/alecthomas/kingpin/v2 v2.3.2 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/auth0/go-jwt-middleware/v2 v2.2.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.6.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.17 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.3.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/autoscaling v1.43.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudfront v1.38.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.40.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/directconnect v1.27.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.177.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecs v1.45.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/efs v1.31.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/eks v1.48.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancing v1.26.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.35.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/iam v1.35.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.3.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.9.18 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.17.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/kms v1.35.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/lambda v1.58.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/networkfirewall v1.41.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/networkmanager v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/rds v1.82.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.43.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/s3 v1.61.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sns v1.31.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sqs v1.34.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.22.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.26.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.30.7 // indirect
	github.com/aws/smithy-go v1.20.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/catppuccin/go v0.2.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/charmbracelet/x/ansi v0.2.3 // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.0 // indirect
	github.com/coder/websocket v1.8.12 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dlclark/regexp2 v1.11.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/iancoleman/strcase v0.2.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/micahhausler/aws-iam-policy v0.4.2 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/miekg/dns v1.1.62 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/nats-io/jwt/v2 v2.5.8 // indirect
	github.com/nats-io/nats.go v1.37.0 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/openrdap/rdap v0.9.2-0.20240517203139-eb57b3a8dedd // indirect
	github.com/overmindtech/discovery v0.27.7 // indirect
	github.com/overmindtech/sdpcache v1.6.4 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/uptrace/opentelemetry-go-extra/otelutil v0.3.1 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xiam/to v0.0.0-20200126224905-d60d31e03561 // indirect
	github.com/yuin/goldmark v1.7.4 // indirect
	github.com/yuin/goldmark-emoji v1.0.3 // indirect
	go.opentelemetry.io/otel/log v0.3.0 // indirect
	go.opentelemetry.io/otel/metric v1.29.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/exp v0.0.0-20240613232115-7f521ea00fb8 // indirect
	golang.org/x/mod v0.18.0 // indirect
	golang.org/x/sync v0.8.0 // indirect
	golang.org/x/sys v0.25.0 // indirect
	golang.org/x/term v0.24.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240822170219-fc7c04adadcd // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240822170219-fc7c04adadcd // indirect
	google.golang.org/grpc v1.65.0 // indirect
	gopkg.in/go-jose/go-jose.v2 v2.6.3 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)
