package netactuate

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSecretList() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecretListCreate,
		ReadContext:   resourceSecretListRead,
		UpdateContext: resourceSecretListUpdate,
		DeleteContext: resourceSecretListDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the secret list",
			},
		},
	}
}

func resourceSecretListCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	list, err := c.CreateSecretList(d.Get("name").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(list.ID))

	return nil
}

func resourceSecretListRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	list, err := c.GetSecretList(id)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("name", list.Name, d, &diags)

	return diags
}

func resourceSecretListUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if _, err := c.UpdateSecretList(id, d.Get("name").(string)); err != nil {
		return diag.FromErr(err)
	}

	return resourceSecretListRead(ctx, d, m)
}

func resourceSecretListDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteSecretList(id); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
