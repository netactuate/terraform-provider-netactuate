package netactuate

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)
//Kubeconfig need only to get kubeconfig for connection, not affect k8s struct
func dataSourceNKEKubeconfig() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceNKEKubeconfigRead,
		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "The ID of the NKE cluster",
			},
			"expiration_seconds": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     31536000,
				Description: "Token lifetime in seconds (default 31536000 = 1 year, max 3153600000)",
			},
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The kubeconfig YAML for accessing the cluster",
			},
		},
	}
}

func dataSourceNKEKubeconfigRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	clusterID := d.Get("cluster_id").(int)

	expirationSeconds := d.Get("expiration_seconds").(int)
	kubeconfig, err := c.GenerateNKEKubeconfig(clusterID, expirationSeconds)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("kubeconfig", kubeconfig, d, &diags)
	d.SetId(fmt.Sprintf("nke-kubeconfig-%d", clusterID))

	return diags
}
