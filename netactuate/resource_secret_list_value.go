package netactuate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceSecretListValue() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSecretListValueCreate,
		ReadContext:   resourceSecretListValueRead,
		UpdateContext: resourceSecretListValueUpdate,
		DeleteContext: resourceSecretListValueDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceSecretListValueImport,
		},
		Schema: map[string]*schema.Schema{
			"secret_list_id": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "The ID of the parent secret list",
			},
			"secret_key": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The key name for this secret",
			},
			"secret_value": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "The secret value",
			},
		},
	}
}

func resourceSecretListValueCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	listID := d.Get("secret_list_id").(int)
	key := d.Get("secret_key").(string)
	value := d.Get("secret_value").(string)

	val, err := c.CreateSecretListValue(listID, key, value)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(val.ID))

	return nil
}

func resourceSecretListValueRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	listID := d.Get("secret_list_id").(int)
	valueID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	val, err := c.GetSecretListValue(listID, valueID)
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("secret_list_id", val.SecretListID, d, &diags)
	setValue("secret_key", val.SecretKey, d, &diags)
	setValue("secret_value", val.SecretValue, d, &diags)

	return diags
}

func resourceSecretListValueUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	listID := d.Get("secret_list_id").(int)
	valueID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	key := d.Get("secret_key").(string)
	value := d.Get("secret_value").(string)

	if _, err := c.UpdateSecretListValue(listID, valueID, key, value); err != nil {
		return diag.FromErr(err)
	}

	return resourceSecretListValueRead(ctx, d, m)
}

func resourceSecretListValueDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V2

	listID := d.Get("secret_list_id").(int)
	valueID, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	if err := c.DeleteSecretListValue(listID, valueID); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

//Handle composite ID "secret_list_id/value_id"
func resourceSecretListValueImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected import ID format: secret_list_id/value_id, got: %s", d.Id())
	}

	listID, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid secret_list_id %q: %w", parts[0], err)
	}

	valueID, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid value_id %q: %w", parts[1], err)
	}

	d.SetId(strconv.Itoa(valueID))
	if err := d.Set("secret_list_id", listID); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
