package config

type Config struct {
	ListenAddr   string `name:"listen-addr" default:":8080"`
	NetBoxHost   string `env:"NETBOX_HOST" name:"netbox-host" help:"host of the NetBox server" required:""`
	NetBoxScheme string `env:"NETBOX_SCHEME" name:"netbox-scheme" help:"http/https" default:"https"`
	NetBoxToken  string `env:"NETBOX_TOKEN" name:"netbox-token" help:"API token for NetBox" required:""`

	Tenants map[string]Tenant `name:"tenants" required:""`
}

type Tenant struct {
	Name       string `name:"name" required:""`
	Short      string `name:"short" required:""`
	Street     string `name:"street" required:""`
	City       string `name:"city" required:""`
	NOC        string `name:"noc" required:""`
	Email      string `name:"email" required:""`
	Phone      string `name:"phone" default:""`
	ExpiryDays int    `name:"expiry" default:"60"`
}
