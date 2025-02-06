package adapters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/overmindtech/cli/sdp-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ScopeDetails struct {
	ClusterName string
	Namespace   string
}

func (sd ScopeDetails) String() string {
	if sd.Namespace == "" {
		return sd.ClusterName
	}

	return fmt.Sprintf("%v.%v", sd.ClusterName, sd.Namespace)
}

// ParseScope Parses the custer and scope name out of a given SDP scope given
// that the naming convention is {clusterName}.{namespace}. Since all adapters
// know whether they are namespaced or not, we can just pass that in to make
// parsing easier
func ParseScope(itemScope string, namespaced bool) (ScopeDetails, error) {
	sections := strings.Split(itemScope, ".")

	var namespace string
	var clusterEnd int
	var clusterName string

	if namespaced {
		if len(sections) < 2 {
			return ScopeDetails{}, fmt.Errorf("scope %v does not contain a namespace in the format: {clusterName}.{namespace}", itemScope)
		}

		namespace = sections[len(sections)-1]
		clusterEnd = len(sections) - 1
	} else {
		namespace = ""
		clusterEnd = len(sections)
	}

	clusterName = strings.Join(sections[:clusterEnd], ".")

	if clusterName == "" {
		return ScopeDetails{}, fmt.Errorf("cluster name was blank for scope %v", itemScope)
	}

	return ScopeDetails{
		ClusterName: clusterName,
		Namespace:   namespace,
	}, nil
}

// Selector represents a set of key value pairs that we are going to use as a
// selector
type Selector map[string]string

// String converts a set of key value pairs to the string format that a selector
// is expecting
func (l Selector) String() string {
	var conditions []string

	conditions = make([]string, 0)

	for k, v := range l {
		conditions = append(conditions, fmt.Sprintf("%v=%v", k, v))
	}

	return strings.Join(conditions, ",")
}

func ListOptionsToQuery(lo *metav1.ListOptions) string {
	jsonData, err := json.Marshal(lo)

	if err == nil {
		return string(jsonData)
	}

	return ""
}

// LabelSelectorToQuery converts a LabelSelector to JSON so that it can be
// passed to a SEARCH query
func LabelSelectorToQuery(labelSelector *metav1.LabelSelector) string {
	return ListOptionsToQuery(&metav1.ListOptions{
		LabelSelector: Selector(labelSelector.MatchLabels).String(),
	})
}

// QueryToListOptions converts a Search() query string to a ListOptions object that can
// be used to query the API
func QueryToListOptions(query string) (metav1.ListOptions, error) {
	var queryBytes []byte
	var err error
	var listOptions metav1.ListOptions

	queryBytes = []byte(query)

	// Convert from JSON
	if err = json.Unmarshal(queryBytes, &listOptions); err != nil {
		return listOptions, err
	}

	// Override some of the things we don't want people to set
	listOptions.Watch = false

	return listOptions, nil
}

var Metadata = sdp.AdapterMetadataList{}
