<img src="pkg/bulkloagen/static/logo.png" alt="Bulk Loagen Logo" width="150" />

# Bulk Loagen

Bulk Loagen will help you generate letters of authorization based on NetBox information.

1. Search for the device (in NetBox) you want to create a LOA on.
2. Go to the [URL containing your device's NetBox ID](http://localhost:8080/api/v1/devices/{deviceID}).
3. Select the rear port to create a LOA for and specify the partner details.

For convenience you should add custom links in NetBox on the rear ports/devices.

## Assumptions

+ You have your demarcation panel documented as a device in NetBox.
+ You use rear ports to "connect" cross-connects to.
+ You specify the facility name in your site (the name given by the data center provider).
+ You specify facility IDs in your rack (the name given by the data center provider).
+ You specify rack tenants or use the fallback to the default tenant.
+ You specify demarc panel and rear port names with the same scheme as your data center provider.

## Configuration

bulk-loagen is configured through a `config.yaml`. The full configuration can be found in [config/config.go](pkg/config/config.go).

You can use one instance of bulk-loagen for multiple tenants. The tenant is determined automatically based on the slug of tenant specified in the NetBox rack object.

A NetBox Token is necessary to request information from the API. You can speficy this token in the `config.yaml` or via environment variable (`NETBOX_TOKEN`).

## Running/Demo

```sh
# Download necessary files for NetBox demo environment
for f in docker-compose.yml configuration/configuration.py configuration/extra.py configuration/logging.py configuration/plugins.py env/netbox.env env/postgres.env env/redis.env env/redis-cache.env; do
    curl -o demo/"$f" https://raw.githubusercontent.com/netbox-community/netbox-docker/release/"$f"
done

# Start NetBox and Bulk Loagen
docker-compose -p bulk-loagen -f demo/docker-compose.yml -f demo/docker-compose.override.yml up -d

# Add sample data to NetBox
./demo/sample-data.sh
```

A sample LOA can be found in [demo/LOA_PartnerCorp_2023-11-19_DC01.pdf](demo/LOA_PartnerCorp_2023-11-19_DC01.pdf)
