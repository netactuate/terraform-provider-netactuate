# Changelog

## Unreleased

### FEATURES

- Add NKE node pool and cluster autoscaler addon resources, node pool listing, and storage availability data sources.

### ENHANCEMENTS

- Support aligned pod CIDRs from /16 to /22 and service CIDRs from /16 to /24 within RFC1918 or CGNAT, cluster network and egress modes, node limits, and worker public and private NIC information.
- Deprecate cluster do_autoscaling in favor of node pool autoscaling. Honor cluster create timeouts in client waits.
