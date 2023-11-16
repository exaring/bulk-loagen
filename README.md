<img src="pkg/bulkloagen/static/logo.png" alt="Bulk Loagen Logo" width="150" />

# Bulk Loagen

Bulk Loagen will help you generate letters of authorization based on NetBox information.

## Configuration

bulk-loagen is configured through a `config.yaml`. The full configuration can be found in [config/config.go](pkg/config/config.go).

You can use one instance of bulk-loagen for multiple tenants. The tenant is determined automatically based on the slug of tenant specified in the NetBox rack object.

A NetBox Token is necessary to request information from the API. You can speficy this token in the `config.yaml` or via environment variable (`NETBOX_TOKEN`).
