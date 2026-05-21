package netactuate

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func resourceSSLCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceSSLCertificateCreate,
		ReadContext:   resourceSSLCertificateRead,
		UpdateContext: resourceSSLCertificateUpdate,
		DeleteContext: resourceSSLCertificateDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"ssl_certificate_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The SSL certificate ID assigned by the API",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the SSL certificate",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Description of the SSL certificate",
			},
			"certificate": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "PEM-encoded certificate",
			},
			"private_key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "PEM-encoded private key",
			},
			"fingerprint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "SHA-256 fingerprint of the certificate",
			},
			"domains": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Domains covered by the certificate",
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			"is_active": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the certificate is active",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Certificate status",
			},
			"expiration": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Certificate expiration date",
			},
		},
	}
}

func resourceSSLCertificateCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	req := &gona.CreateSSLCertificateRequest{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Certificate: d.Get("certificate").(string),
		PrivateKey:  d.Get("private_key").(string),
	}

	result, err := c.CreateSSLCertificate(req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.Itoa(result.SSLCertificateID))
	log.Printf("[DEBUG] SSL certificate %d created", result.SSLCertificateID)

	return resourceSSLCertificateRead(ctx, d, m)
}

func resourceSSLCertificateRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	cert, err := c.GetSSLCertificate(id)
	if err != nil {
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SSL certificate %d not found, removing from state", id)
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics

	setValue("ssl_certificate_id", cert.SSLCertificateID, d, &diags)
	setValue("name", cert.Name, d, &diags)
	setValue("description", cert.Description, d, &diags)
	setValue("fingerprint", cert.Fingerprint, d, &diags)
	setValue("domains", cert.Domains, d, &diags)
	setValue("is_active", cert.IsActive, d, &diags)
	setValue("status", cert.Status, d, &diags)

	if cert.Dates != nil {
		setValue("expiration", cert.Dates.Expiration, d, &diags)
	}

	return diags
}

func resourceSSLCertificateUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := &gona.UpdateSSLCertificateRequest{}

	if d.HasChange("name") {
		req.Name = d.Get("name").(string)
	}
	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}
	if d.HasChange("certificate") {
		req.Certificate = d.Get("certificate").(string)
	}
	if d.HasChange("private_key") {
		req.PrivateKey = d.Get("private_key").(string)
	}

	if err := c.UpdateSSLCertificate(id, req); err != nil {
		return diag.FromErr(err)
	}

	return resourceSSLCertificateRead(ctx, d, m)
}

func resourceSSLCertificateDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	id, err := strconv.Atoi(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] Deleting SSL certificate %d", id)

	// The API rejects cert deletion while an HTTP LB group that referenced it
	// is still being torn down. Retry briefly to let the usage reference clear.
	const retries = 10
	const retryInterval = 5 * time.Second
	for i := 0; i < retries; i++ {
		err := c.DeleteSSLCertificate(id)
		if err == nil {
			return nil
		}
		if gona.IsV3NotFound(err) {
			log.Printf("[WARN] SSL certificate %d already deleted", id)
			return nil
		}
		if strings.Contains(err.Error(), "actively used") {
			log.Printf("[DEBUG] SSL certificate %d still in use, retrying (%d/%d)...", id, i+1, retries)
			time.Sleep(retryInterval)
			continue
		}
		return diag.FromErr(err)
	}

	return diag.Errorf("SSL certificate %d still reported as in-use after %d retries", id, retries)
}
