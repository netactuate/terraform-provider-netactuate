package netactuate

import (
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TestNoDroppedNestedSensitive guards a terraform-plugin-sdk/v2 trap that is
// invisible in code review.
//
// A Computed-only TypeList or TypeSet whose Elem is a *schema.Resource is
// converted by the SDK into a cty object attribute. A cty object type has no
// per-field sensitivity, so a nested `Sensitive: true` on one of its fields is
// silently discarded: the value then renders in cleartext in `terraform plan`
// output, in `terraform show`, and in state. The same element schema declared
// Optional or Required stays a block, where the nested flag survives.
//
// The netactuate_secret_values and netactuate_secret_list_values data sources
// hit exactly this shape and leaked every secret value they returned. The fix
// is to mark the whole list Sensitive (see data_source_secret_list_values.go).
//
// This test walks every schema in the provider (provider config, all resources,
// all data sources, recursively) and fails if any Computed-only list or set
// carries a nested Sensitive field while the list itself is not Sensitive. That
// is precisely the set of places where the flag has been written and will be
// ignored.
func TestNoDroppedNestedSensitive(t *testing.T) {
	p := Provider()

	var violations []string
	walkForDroppedSensitive("provider", p.Schema, &violations)
	for name, r := range p.ResourcesMap {
		walkForDroppedSensitive("resource "+name, r.Schema, &violations)
	}
	for name, d := range p.DataSourcesMap {
		walkForDroppedSensitive("data_source "+name, d.Schema, &violations)
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		for _, v := range violations {
			t.Errorf("dropped nested Sensitive flag: %s", v)
		}
		t.Fatalf("%d attribute(s) mark a nested field Sensitive inside a Computed-only "+
			"list/set that is not itself Sensitive. SDKv2 discards the nested flag and the "+
			"value leaks in cleartext in plan, show and state. Mark the list Sensitive, or "+
			"move the element to a block (Optional/Required).", len(violations))
	}
}

func walkForDroppedSensitive(path string, m map[string]*schema.Schema, violations *[]string) {
	for name, s := range m {
		here := path + "." + name
		res, ok := s.Elem.(*schema.Resource)
		if !ok {
			continue
		}
		computedOnly := s.Computed && !s.Optional && !s.Required
		nested := s.Type == schema.TypeList || s.Type == schema.TypeSet
		if computedOnly && nested && !s.Sensitive {
			if fields := sensitiveFields(res); len(fields) > 0 {
				*violations = append(*violations, fmt.Sprintf(
					"%s (Computed-only %s) has nested Sensitive field(s) %v but the list itself is not Sensitive",
					here, typeLabel(s.Type), fields))
			}
		}
		walkForDroppedSensitive(here, res.Schema, violations)
	}
}

func sensitiveFields(res *schema.Resource) []string {
	var fields []string
	for k, v := range res.Schema {
		if v.Sensitive {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)
	return fields
}

func typeLabel(t schema.ValueType) string {
	switch t {
	case schema.TypeList:
		return "TypeList"
	case schema.TypeSet:
		return "TypeSet"
	default:
		return "list"
	}
}

// TestSecretDataSourcesCoreSchemaSensitive locks the fix at the exact layer the
// bug lived: CoreConfigSchema() is the cty conversion that dropped the nested
// flag. It asserts the leak-prone list attributes are Sensitive in the block
// Terraform actually receives, so a regression is caught even if someone moves
// the flag back down to the element fields.
func TestSecretDataSourcesCoreSchemaSensitive(t *testing.T) {
	cases := []struct {
		dataSource func() *schema.Resource
		attr       string
	}{
		{dataSourceSecretValues, "values"},
		{dataSourceSecretListValues, "values"},
		{dataSourceOIDCClientKeys, "keys"},
		{dataSourceStorageBlockVolumes, "block_volumes"},
	}
	for _, c := range cases {
		block := c.dataSource().CoreConfigSchema()
		attr, ok := block.Attributes[c.attr]
		if !ok {
			t.Errorf("attribute %q not found in core schema", c.attr)
			continue
		}
		if !attr.Sensitive {
			t.Errorf("attribute %q is NOT Sensitive in the core (cty) schema; secret values would render in cleartext", c.attr)
		}
	}
}
