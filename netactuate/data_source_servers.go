package netactuate

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceServers() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceServersRead,
		Schema: map[string]*schema.Schema{
			"servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of servers visible to the account",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"fqdn": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server fully qualified domain name",
						},
						"mbpkgid": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Server package ID",
						},
						"os": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server operating system name",
						},
						"os_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Server operating system image ID",
						},
						"ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Primary IPv4 address",
						},
						"ipv6": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Primary IPv6 address",
						},
						"plan_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Plan ID",
						},
						"package": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Package name",
						},
						"contract_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Package billing contract ID",
						},
						"city": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server location name",
						},
						"location_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Server location ID",
						},
						"status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server status",
						},
						"state": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Server power state",
						},
						"installed": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Server installed flag",
						},
						"cloud_pool_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Cloud pool ID, or 0 when the API returns null",
						},
						"vpc_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "VPC ID, or 0 when the API returns null",
						},
						"vpc_reserved_network": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "VPC reserved network",
						},
					},
				},
			},
		},
	}
}

func dataSourceServersRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	servers, err := c.GetServers()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	serversList := make([]map[string]interface{}, len(servers))
	for i, server := range servers {
		cloudPoolID := 0
		if server.CloudPoolID != nil {
			cloudPoolID = *server.CloudPoolID
		}
		vpcID := 0
		if server.VpcID != nil {
			vpcID = *server.VpcID
		}

		serversList[i] = map[string]interface{}{
			"fqdn":                 server.Name,
			"mbpkgid":              server.ID,
			"os":                   server.OS,
			"os_id":                server.OSID,
			"ip":                   server.PrimaryIPv4,
			"ipv6":                 server.PrimaryIPv6,
			"plan_id":              server.PlanID,
			"package":              server.Package,
			"contract_id":          server.PackageBillingContractId,
			"city":                 server.Location,
			"location_id":          server.LocationID,
			"status":               server.ServerStatus,
			"state":                server.PowerStatus,
			"installed":            server.Installed,
			"cloud_pool_id":        cloudPoolID,
			"vpc_id":               vpcID,
			"vpc_reserved_network": server.VpcReservedNetwork,
		}
	}

	setValue("servers", serversList, d, &diags)
	d.SetId("servers")

	return diags
}
