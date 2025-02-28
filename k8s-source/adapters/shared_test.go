package adapters

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/kind/pkg/apis/config/v1alpha4"
	"sigs.k8s.io/kind/pkg/cluster"
)

const TestNamespace = "k8s-source-testing"

const TestNamespaceYAML = `
apiVersion: v1
kind: Namespace
metadata:
  name: k8s-source-testing
`

type TestCluster struct {
	Name       string
	Kubeconfig string
	ClientSet  *kubernetes.Clientset
	provider   *cluster.Provider
	T          *testing.T
}

func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}

func (t *TestCluster) ConnectExisting(name string) error {
	kubeconfig := homedir.HomeDir() + "/.kube/config"

	var rc *rest.Config
	var err error

	// Load kubernetes config
	rc, err = buildConfigWithContextFromFlags("kind-"+name, kubeconfig)
	if err != nil {
		return err
	}

	var clientSet *kubernetes.Clientset

	// Create clientset
	clientSet, err = kubernetes.NewForConfig(rc)
	if err != nil {
		return err
	}

	// Validate that we can connect to the cluster
	_, err = clientSet.CoreV1().Namespaces().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return err
	}

	t.Name = name
	t.Kubeconfig = kubeconfig
	t.ClientSet = clientSet

	return nil
}

func (t *TestCluster) Start() error {
	clusterName := "local-tests"

	log.Println("🔍 Trying to connect to existing cluster")
	err := t.ConnectExisting(clusterName)

	if err != nil {
		// If there is an error then create out own cluster
		log.Println("🤞 Creating Kubernetes cluster using Kind")

		clusterConfig := new(v1alpha4.Cluster)

		// Read environment variables to check for kube version
		if version, ok := os.LookupEnv("KUBE_VERSION"); ok {
			log.Printf("⚙️ Setting custom Kubernetes version: %v\n", version)

			clusterConfig.Nodes = []v1alpha4.Node{
				{
					Role:  v1alpha4.ControlPlaneRole,
					Image: fmt.Sprintf("kindest/node:%v", version),
				},
			}
		}

		t.provider = cluster.NewProvider()
		err = t.provider.Create(clusterName, cluster.CreateWithV1Alpha4Config(clusterConfig))

		if err != nil {
			return err
		}

		// Connect to the cluster we just created
		err = t.ConnectExisting(clusterName)

		if err != nil {
			return err
		}

		err = t.provider.ExportKubeConfig(t.Name, t.Kubeconfig, false)

		if err != nil {
			return err
		}
	}

	log.Printf("🐚 Ensuring test namespace %v exists\n", TestNamespace)
	err = t.Apply(TestNamespaceYAML)

	if err != nil {
		return err
	}

	return nil
}

// func (t *TestCluster) ApplyBaselineConfig() error {
// 	return t.Apply(ClusterBaseline)
// }

// Apply Runs of `kubectl apply -f` for a given string of YAML
func (t *TestCluster) Apply(yaml string) error {
	return t.kubectl("apply", yaml)
}

// Delete Runs of `kubectl delete -f` for a given string of YAML
func (t *TestCluster) Delete(yaml string) error {
	return t.kubectl("delete", yaml)
}

func (t *TestCluster) kubectl(method string, yaml string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Create temp file to write config to
	config, err := os.CreateTemp("", "*-conf.yaml")
	if err != nil {
		return err
	}

	_, err = config.WriteString(yaml)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", method, "-f", config.Name())
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = filepath.Dir(config.Name())

	// Inherit from the ENV
	cmd.Env = os.Environ()

	// Set KUBECONFIG location
	cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%v", t.Kubeconfig))

	// Run the command
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("%w\nstdout: %v\nstderr: %v", err, stdout.String(), stderr.String())
	}

	if e := stderr.String(); e != "" {
		return errors.New(e)
	}

	return nil
}

func (t *TestCluster) Stop() error {
	if t.provider != nil {
		log.Println("🏁 Destroying cluster")

		return t.provider.Delete(t.Name, t.Kubeconfig)
	}

	return nil
}

var CurrentCluster TestCluster

func TestMain(m *testing.M) {
	CurrentCluster = TestCluster{}

	err := CurrentCluster.Start()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	// log.Println("🎁 Creating resources in cluster for testing")
	// err = CurrentCluster.ApplyBaselineConfig()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	log.Println("✅ Running tests")
	code := m.Run()

	err = CurrentCluster.Stop()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	os.Exit(code)
}
