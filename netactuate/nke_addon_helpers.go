package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

const (
	nkeDNSAddonType     = "netactuate-dns"
	nkeStorageAddonType = "storage"
)

func nkeAddonStateID(clusterID int, addonType string) string {
	return fmt.Sprintf("%d/%s", clusterID, addonType)
}

func parseNKEAddonStateID(id string) (int, string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return 0, "", fmt.Errorf("expected addon ID in the form cluster_id/addon_type")
	}
	clusterID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", err
	}
	return clusterID, parts[1], nil
}

func validateNKEAddonPlan(ctx context.Context, diff *schema.ResourceDiff, meta interface{}, addonType string) error {
	clusterID := diff.Get("cluster_id").(int)
	if clusterID == 0 || meta == nil {
		return nil
	}

	c := meta.(*ProviderClients).V3
	catalog, err := c.ListAddonCatalog()
	if err != nil {
		return fmt.Errorf("fetch NKE addon catalog: %w", err)
	}

	var entry *gona.NKEAddonCatalogEntry
	for i := range catalog {
		if catalog[i].AddonType == addonType {
			entry = &catalog[i]
			break
		}
	}
	if entry == nil {
		return fmt.Errorf("NKE addon %q was not found in the addon catalog", addonType)
	}

	cluster, err := c.GetNKECluster(clusterID)
	if err != nil {
		return fmt.Errorf("fetch NKE cluster %d for addon validation: %w", clusterID, err)
	}

	if entry.RequiresVpc && cluster.VpcID == 0 {
		return fmt.Errorf("NKE addon %q requires a VPC cluster", addonType)
	}
	if entry.MinKubernetesVersion != "" && compareKubernetesVersions(cluster.Version.Active, entry.MinKubernetesVersion) < 0 {
		return fmt.Errorf("NKE addon %q requires Kubernetes version %s or newer, cluster %d is running %s",
			addonType, entry.MinKubernetesVersion, clusterID, cluster.Version.Active)
	}

	return nil
}

func compareKubernetesVersions(a, b string) int {
	av := kubernetesVersionParts(a)
	bv := kubernetesVersionParts(b)
	for i := 0; i < len(av) || i < len(bv); i++ {
		var ai, bi int
		if i < len(av) {
			ai = av[i]
		}
		if i < len(bv) {
			bi = bv[i]
		}
		if ai < bi {
			return -1
		}
		if ai > bi {
			return 1
		}
	}
	return 0
}

func kubernetesVersionParts(v string) []int {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if idx := strings.IndexAny(v, "+-"); idx >= 0 {
		v = v[:idx]
	}
	pieces := strings.Split(v, ".")
	parts := make([]int, 0, len(pieces))
	for _, piece := range pieces {
		if piece == "" {
			parts = append(parts, 0)
			continue
		}
		n, err := strconv.Atoi(piece)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, n)
	}
	return parts
}
