package adapters

import (
	"github.com/overmindtech/cli/discovery"
	"k8s.io/client-go/kubernetes"
)

type AdapterLoader func(clientSet *kubernetes.Clientset, cluster string, namespaces []string) discovery.ListableAdapter

var adapterLoaders []AdapterLoader

func registerAdapterLoader(loader AdapterLoader) {
	adapterLoaders = append(adapterLoaders, loader)
}

func LoadAllAdapters(cs *kubernetes.Clientset, cluster string, namespaces []string) []discovery.Adapter {
	adapters := make([]discovery.Adapter, len(adapterLoaders))

	for i, loader := range adapterLoaders {
		adapters[i] = loader(cs, cluster, namespaces)
	}

	return adapters
}
