package netactuate

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/netactuate/gona/gona"
)

func drop2Timeouts() *schema.ResourceTimeout {
	return &schema.ResourceTimeout{Create: schema.DefaultTimeout(30 * time.Minute), Update: schema.DefaultTimeout(30 * time.Minute), Delete: schema.DefaultTimeout(30 * time.Minute)}
}
func drop2IDSchema() *schema.Schema {
	return &schema.Schema{Type: schema.TypeInt, Required: true, ForceNew: true, ValidateDiagFunc: validation.ToDiagFunc(validation.IntAtLeast(1))}
}
func drop2String(computed bool) *schema.Schema {
	return &schema.Schema{Type: schema.TypeString, Computed: computed}
}
func drop2Import(parent string, composite bool, stringChild bool) schema.StateContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
		parts := strings.Split(d.Id(), "/")
		expected := 1
		if composite {
			expected = 2
		}
		if len(parts) != expected {
			return nil, fmt.Errorf("invalid import ID, expected %s%s", parent, map[bool]string{true: "/entryId", false: ""}[composite])
		}
		id, err := strconv.Atoi(parts[0])
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("invalid import ID: %s must be a positive integer", parent)
		}
		if composite {
			if parts[1] == "" || strings.TrimSpace(parts[1]) != parts[1] || strings.ContainsAny(parts[1], "?#%\\") {
				return nil, fmt.Errorf("invalid entry ID")
			}
			if !stringChild {
				child, err := strconv.Atoi(parts[1])
				if err != nil || child <= 0 {
					return nil, fmt.Errorf("entry ID must be a positive integer")
				}
			}
		}
		if err = d.Set(parent, id); err != nil {
			return nil, err
		}
		return []*schema.ResourceData{d}, nil
	}
}
func drop2Parts(d *schema.ResourceData) (int, string, error) {
	p := strings.Split(d.Id(), "/")
	if len(p) != 2 {
		return 0, "", fmt.Errorf("expected parentId/entryId")
	}
	id, e := strconv.Atoi(p[0])
	if e != nil || id <= 0 || p[1] == "" {
		return 0, "", fmt.Errorf("invalid resource ID")
	}
	return id, p[1], nil
}

// Transport errors from net/http can contain the authenticated request URL.
func drop2Error(err error) diag.Diagnostics {
	if err == nil {
		return nil
	}
	var msg string
	var e *url.Error
	if errors.As(err, &e) {
		msg = e.Err.Error()
	} else {
		msg = err.Error()
	}
	return diag.Errorf("%s", msg)
}
func drop2ReadError(d *schema.ResourceData, err error) diag.Diagnostics {
	if gona.IsV3NotFound(err) {
		d.SetId("")
		return nil
	}
	return drop2Error(err)
}
func drop2Wait(ctx context.Context, timeout time.Duration, check func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		done, err := check()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
func drop2Strings(d *schema.ResourceData, key string) []string {
	out := []string{}
	for _, v := range d.Get(key).(*schema.Set).List() {
		out = append(out, v.(string))
	}
	return out
}
