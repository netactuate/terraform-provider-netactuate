package netactuate

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/netactuate/gona/gona"
)

func dataSourceStatistics() *schema.Resource {
	return statisticsDataSource(dataSourceStatisticsRead("statistics", func(c *gona.V3Client, metrics []string) ([]gona.StatisticResult, error) {
		return c.QueryStatistics(metrics)
	}))
}

func dataSourceNetworkingStatistics() *schema.Resource {
	return statisticsDataSource(dataSourceStatisticsRead("networking-statistics", func(c *gona.V3Client, metrics []string) ([]gona.StatisticResult, error) {
		return c.QueryNetworkingStatistics(metrics)
	}))
}

func dataSourceAnycastStatistics() *schema.Resource {
	return statisticsDataSource(dataSourceStatisticsRead("anycast-statistics", func(c *gona.V3Client, metrics []string) ([]gona.StatisticResult, error) {
		return c.QueryAnycastStatistics(metrics)
	}))
}

func statisticsDataSource(read schema.ReadContextFunc) *schema.Resource {
	return &schema.Resource{
		ReadContext: read,
		Schema: map[string]*schema.Schema{
			"metrics": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "Metric names to query. The provider converts each name into the nested API metric object shape.",
				Elem: &schema.Schema{
					Type:        schema.TypeString,
					Description: "Metric name.",
				},
			},
			"results": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Statistics results returned by the API. Endpoint-specific extra fields are not surfaced.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Metric name extracted from the API metric object key.",
						},
						"service": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Service value returned for this metric.",
						},
						"samples": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Statistic samples returned for this metric.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"count": {
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: "Sample count returned by the API.",
									},
									"resources": {
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: "Resource count returned by the API.",
									},
									"avg": {
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: "Average value returned by the API.",
									},
									"sum": {
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: "Sum value returned by the API.",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func dataSourceStatisticsRead(idPrefix string, query func(*gona.V3Client, []string) ([]gona.StatisticResult, error)) schema.ReadContextFunc {
	return func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
		c := m.(*ProviderClients).V3

		metrics := expandStringList(d.Get("metrics").([]interface{}))
		// These statistics endpoints are read-only query surfaces even though
		// the API expresses the query as POST.
		results, err := query(c, metrics)
		if err != nil {
			return diag.FromErr(err)
		}

		var diags diag.Diagnostics
		setValue("results", flattenStatisticsResults(results), d, &diags)

		idMetrics := append([]string(nil), metrics...)
		sort.Strings(idMetrics)
		d.SetId(idPrefix + ":" + strings.Join(idMetrics, ","))

		return diags
	}
}

func dataSourceMetricNames() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceMetricNamesRead,
		Schema: map[string]*schema.Schema{
			"time_window": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Time window returned by the metric catalog endpoint.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Time window start timestamp returned by the API.",
						},
						"end": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Time window end timestamp returned by the API.",
						},
						"seconds": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Time window duration in seconds.",
						},
					},
				},
			},
			"metrics": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Metric catalog entries sorted by metric name. API keys beginning with double underscores are excluded.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Metric name from the API response key.",
						},
						"service": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Service value returned for this metric.",
						},
						"resources": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Resource count returned for this metric.",
						},
						"avg":  metricSummarySchema("Average summary for this metric."),
						"last": metricSummarySchema("Last-value summary for this metric."),
						"sum":  metricSummarySchema("Sum summary for this metric."),
					},
				},
			},
		},
	}
}

func metricSummarySchema(description string) *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeList,
		Computed:    true,
		Description: description,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"sum": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "Summary sum value.",
				},
				"avg": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "Summary average value.",
				},
				"min": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "Summary minimum value.",
				},
				"max": {
					Type:        schema.TypeFloat,
					Computed:    true,
					Description: "Summary maximum value.",
				},
			},
		},
	}
}

func dataSourceMetricNamesRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c := m.(*ProviderClients).V3

	// This metric catalog endpoint is a read-only query surface even though
	// the API expresses the query as POST.
	names, err := c.GetMetricNames()
	if err != nil {
		return diag.FromErr(err)
	}

	var diags diag.Diagnostics
	setValue("time_window", []map[string]interface{}{
		{
			"start":   names.TimeWindow.Start,
			"end":     names.TimeWindow.End,
			"seconds": names.TimeWindow.Seconds,
		},
	}, d, &diags)
	setValue("metrics", flattenMetricNames(names.Metrics), d, &diags)
	d.SetId("metric-names")

	return diags
}

func expandStringList(values []interface{}) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.(string)
	}
	return result
}

func flattenStatisticsResults(results []gona.StatisticResult) []map[string]interface{} {
	results = append([]gona.StatisticResult(nil), results...)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Metric < results[j].Metric
	})

	out := make([]map[string]interface{}, len(results))
	for i, result := range results {
		samples := make([]map[string]interface{}, len(result.Data))
		for j, sample := range result.Data {
			samples[j] = map[string]interface{}{
				"count":     sample.Count,
				"resources": sample.Resources,
				"avg":       sample.Avg,
				"sum":       sample.Sum,
			}
		}

		out[i] = map[string]interface{}{
			"metric":  result.Metric,
			"service": result.Service,
			"samples": samples,
		}
	}
	return out
}

func flattenMetricNames(metrics []gona.MetricName) []map[string]interface{} {
	out := make([]map[string]interface{}, len(metrics))
	for i, metric := range metrics {
		out[i] = map[string]interface{}{
			"metric":    metric.Metric,
			"service":   metric.Service,
			"resources": metric.Resources,
			"avg":       flattenMetricSummary(metric.Avg),
			"last":      flattenMetricSummary(metric.Last),
			"sum":       flattenMetricSummary(metric.Sum),
		}
	}
	return out
}

func flattenMetricSummary(summary gona.MetricSummary) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"sum": summary.Sum,
			"avg": summary.Avg,
			"min": summary.Min,
			"max": summary.Max,
		},
	}
}
