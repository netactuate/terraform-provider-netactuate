package netactuate

import (
	"fmt"
	"github.com/netactuate/gona/gona"
)

func ensureRouterNotInMagicMesh(c *gona.V3Client, routerID int) error {
	cfg, err := c.GetRouterConfig(routerID)
	if err != nil {
		return err
	}

	if cfg.Metadata.MeshID != nil {
		return fmt.Errorf(
			"Router %d is already enrolled in magic mesh %d; additional VRFs are not allowed for routers in magic mesh",
			routerID,
			*cfg.Metadata.MeshID,
		)
	}

	return nil
}

func ensureRouterHasOnlyDefaultVRF(c *gona.V3Client, routerID int) error {
	cfg, err := c.GetRouterConfig(routerID)
	if err != nil {
		return err
	}

	if hasAdditionalVRFs(cfg) {
		return fmt.Errorf(
			"Router %d has non-default VRFs configured; routers in magic mesh only allow the default VRF",
			routerID,
		)
	}

	return nil
}

func hasAdditionalVRFs(cfg *gona.RouterConfig) bool {
	if cfg == nil || len(cfg.VRF) == 0 {
		return false
	}
	for _, vrf := range cfg.VRF {
		if vrf.VrfID != cfg.DefaultVrfID {
			return true
		}
	}

	return len(cfg.VRF) > 1
}
