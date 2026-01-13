package adapters

import (
	"testing"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
)

var configMapYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-configmap
data:
  DATABASE_URL: "postgres://myuser:mypassword@mydbhost:5432/mydatabase"
  APP_SECRET_KEY: "mysecretkey123"
`

var configMapWithS3ARNYAML = `
apiVersion: v1
kind: ConfigMap
metadata:
  name: configmap-with-s3-arn
data:
  S3_BUCKET_ARN: "arn:aws:s3:::example-bucket-name"
  S3_BUCKET_NAME: "example-bucket-name"
`

func TestConfigMapAdapter(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newConfigMapAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace}, sdpcache.NewNoOpCache())

	st := AdapterTests{
		Adapter:       adapter,
		GetQuery:      "my-configmap",
		GetScope:      sd.String(),
		SetupYAML:     configMapYAML,
		GetQueryTests: QueryTests{},
	}

	st.Execute(t)
}

func TestConfigMapAdapterWithS3ARN(t *testing.T) {
	sd := ScopeDetails{
		ClusterName: CurrentCluster.Name,
		Namespace:   "default",
	}

	adapter := newConfigMapAdapter(CurrentCluster.ClientSet, sd.ClusterName, []string{sd.Namespace}, sdpcache.NewNoOpCache())

	st := AdapterTests{
		Adapter:   adapter,
		GetQuery:  "configmap-with-s3-arn",
		GetScope:  sd.String(),
		SetupYAML: configMapWithS3ARNYAML,
		GetQueryTests: QueryTests{
			{
				ExpectedType:   "s3-bucket",
				ExpectedMethod: sdp.QueryMethod_SEARCH,
				ExpectedQuery:  "arn:aws:s3:::example-bucket-name",
				ExpectedScope:  sdp.WILDCARD,
			},
		},
	}

	st.Execute(t)
}
