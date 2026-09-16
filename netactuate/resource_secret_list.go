package netactuate

import (
	"context"
	"github.com/netactuate/gona/gona"
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
		if gona.IsNotFound(err) {
			// Deleted out of band. vAPI2 reports a missing secret as a 422 on its id
			// field rather than a 404; the SDK maps that to NotFoundError. Without this
			// the resource hard errors on every refresh on a credential resource.
			d.SetId("")
			return nil
		}
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
		if gona.IsNotFound(err) {
			// Already gone, which is what a delete wants. Not an error.
			return nil
		}
		return diag.FromErr(err)
	}

	return nil
}
