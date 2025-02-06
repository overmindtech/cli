package adapters

import "testing"

func TestParseScope(t *testing.T) {
	type ParseTest struct {
		Input        string
		ClusterName  string
		Namespace    string
		IsNamespaced bool
		ExpectError  bool
	}

	tests := []ParseTest{
		{
			Input:        "127.0.0.1:61081.default",
			ClusterName:  "127.0.0.1:61081",
			Namespace:    "default",
			IsNamespaced: true,
		},
		{
			Input:        "127.0.0.1:61081.kube-node-lease",
			ClusterName:  "127.0.0.1:61081",
			Namespace:    "kube-node-lease",
			IsNamespaced: true,
		},
		{
			Input:        "127.0.0.1:61081.kube-public",
			ClusterName:  "127.0.0.1:61081",
			Namespace:    "kube-public",
			IsNamespaced: true,
		},
		{
			Input:        "127.0.0.1:61081.kube-system",
			ClusterName:  "127.0.0.1:61081",
			Namespace:    "kube-system",
			IsNamespaced: true,
		},
		{
			Input:        "127.0.0.1:61081",
			ClusterName:  "127.0.0.1:61081",
			Namespace:    "",
			IsNamespaced: false,
		},
		{
			Input:        "cluster1.k8s.company.com:443",
			ClusterName:  "cluster1.k8s.company.com:443",
			Namespace:    "",
			IsNamespaced: false,
		},
		{
			Input:        "cluster1.k8s.company.com",
			ClusterName:  "cluster1.k8s.company.com",
			Namespace:    "",
			IsNamespaced: false,
		},
		{
			Input:        "test",
			ClusterName:  "test",
			Namespace:    "",
			IsNamespaced: false,
		},
		{
			Input:        "prod.default",
			ClusterName:  "prod",
			Namespace:    "default",
			IsNamespaced: true,
		},
		{
			Input:        "prod",
			ClusterName:  "",
			Namespace:    "prod",
			IsNamespaced: true,
			ExpectError:  true,
		},
		{
			Input:        "prod.default.test",
			ClusterName:  "prod.default.test",
			Namespace:    "",
			IsNamespaced: false,
		},
		{
			Input:        "prod.default.test",
			ClusterName:  "prod.default",
			Namespace:    "test",
			IsNamespaced: true,
		},
		{
			Input:        "",
			ClusterName:  "",
			Namespace:    "",
			IsNamespaced: false,
			ExpectError:  true,
		},
		{
			Input:        "",
			ClusterName:  "",
			Namespace:    "",
			IsNamespaced: true,
			ExpectError:  true,
		},
	}

	for _, test := range tests {
		result, err := ParseScope(test.Input, test.IsNamespaced)

		if test.ExpectError {
			if err == nil {
				t.Errorf("Expected error, but got none. Test %v", test)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if test.ClusterName != result.ClusterName {
				t.Errorf("ClusterName did not match, expected %v, got %v", test.ClusterName, result.ClusterName)
			}

			if test.Namespace != result.Namespace {
				t.Errorf("Namespace did not match, expected %v, got %v", test.Namespace, result.Namespace)
			}
		}

	}

}
