package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type Cluster struct {
	Name   string
	APIURL string
	CAData []byte
}

type managedClusterList struct {
	Items []managedCluster `json:"items"`
}

type managedCluster struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		ManagedClusterClientConfigs []struct {
			URL     string `json:"url"`
			CABundle string `json:"caBundle"`
		} `json:"managedClusterClientConfigs"`
	} `json:"spec"`
}

func ListManagedClusters(ctx context.Context) ([]Cluster, error) {
	cmd := exec.CommandContext(ctx, "oc", "get", "managedclusters.cluster.open-cluster-management.io", "-o", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("oc get managedclusters: %w", err)
	}
	var mcl managedClusterList
	if err := json.Unmarshal(out, &mcl); err != nil {
		return nil, err
	}
	result := make([]Cluster, 0, len(mcl.Items))
	for _, it := range mcl.Items {
		var api, ca string
		if len(it.Spec.ManagedClusterClientConfigs) > 0 {
			api = it.Spec.ManagedClusterClientConfigs[0].URL
			ca = it.Spec.ManagedClusterClientConfigs[0].CABundle
		}
		result = append(result, Cluster{
			Name:   it.Metadata.Name,
			APIURL: api,
			CAData: []byte(ca),
		})
	}
	return result, nil
}

func GetCluster(ctx context.Context, name string) (Cluster, error) {
	clusters, err := ListManagedClusters(ctx)
	if err != nil {
		return Cluster{}, err
	}
	for _, c := range clusters {
		if c.Name == name {
			return c, nil
		}
	}
	return Cluster{}, fmt.Errorf("Cluster %s nicht gefunden", name)
}
