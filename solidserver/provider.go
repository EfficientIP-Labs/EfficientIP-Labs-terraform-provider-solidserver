package solidserver

import (
	"context"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"regexp"
)

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOLIDServer_HOST", nil),
				Description: "SOLIDServer Hostname or IP address",
			},
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOLIDServer_USERNAME", nil),
				Description: "SOLIDServer API user's ID",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOLIDServer_PASSWORD", nil),
				Description: "SOLIDServer API user's password",
			},
			"sslverify": {
				Type:        schema.TypeBool,
				Required:    false,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOLIDServer_SSLVERIFY", true),
				Description: "Enable/Disable ssl verify (Default : enabled)",
			},
			"additional_trust_certs_file": {
				Type:        schema.TypeString,
				Required:    false,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("SOLIDServer_ADDITIONALTRUSTCERTSFILE", nil),
				Description: "PEM formatted file with additional certificates to trust for TLS connection",
			},
			"solidserverversion": {
				Type:         schema.TypeString,
				Required:     false,
				Optional:     true,
				DefaultFunc:  schema.EnvDefaultFunc("SOLIDServer_VERSION", ""),
				ValidateFunc: validation.StringMatch(regexp.MustCompile(`^([0-9]\.[0-9]\.[0-9]\.([pP][0-9]+[a-z]?)?)?$`), "Invalid Version Number"),
				Description:  "SOLIDServer Version in case API user does not have admin permissions",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"solidserver_ip_space":         dataSourceipspace(),
			"solidserver_ip_subnet":        dataSourceipsubnet(),
			"solidserver_ip_subnet_query":  dataSourceipsubnetquery(),
			"solidserver_ip6_subnet":       dataSourceip6subnet(),
			"solidserver_ip6_subnet_query": dataSourceip6subnetquery(),
			"solidserver_ip_pool":          dataSourceippool(),
			"solidserver_ip6_pool":         dataSourceip6pool(),
			"solidserver_ip_address":       dataSourceipaddress(),
			"solidserver_ip6_address":      dataSourceip6address(),
			"solidserver_ip_ptr":           dataSourceipptr(),
			"solidserver_ip6_ptr":          dataSourceip6ptr(),
			"solidserver_dns_smart":        dataSourcednssmart(),
			"solidserver_dns_server":       dataSourcednsserver(),
			"solidserver_dns_view":         dataSourcednsview(),
			"solidserver_dns_zone":         dataSourcednszone(),
			"solidserver_usergroup":        dataSourceusergroup(),
			"solidserver_cdb":              dataSourcecdb(),
			"solidserver_cdb_data":         dataSourcecdbdata(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"solidserver_ip_space":         resourceipspace(),
			"solidserver_ip_subnet":        resourceipsubnet(),
			"solidserver_ip6_subnet":       resourceip6subnet(),
			"solidserver_ip_pool":          resourceippool(),
			"solidserver_ip6_pool":         resourceip6pool(),
			"solidserver_ip_address":       resourceipaddress(),
			"solidserver_ip6_address":      resourceip6address(),
			"solidserver_ip_alias":         resourceipalias(),
			"solidserver_ip6_alias":        resourceip6alias(),
			"solidserver_ip_mac":           resourceipmac(),
			"solidserver_ip6_mac":          resourceip6mac(),
			"solidserver_device":           resourcedevice(),
			"solidserver_vlan_domain":      resourcevlandomain(),
			"solidserver_vlan":             resourcevlan(),
			"solidserver_dns_smart":        resourcednssmart(),
			"solidserver_dns_server":       resourcednsserver(),
			"solidserver_dns_view":         resourcednsview(),
			"solidserver_dns_zone":         resourcednszone(),
			"solidserver_dns_forward_zone": resourcednsforwardzone(),
			"solidserver_dns_rr":           resourcednsrr(),
			"solidserver_app_application":  resourceapplication(),
			"solidserver_app_pool":         resourceapplicationpool(),
			"solidserver_app_node":         resourceapplicationnode(),
			"solidserver_user":             resourceuser(),
			"solidserver_usergroup":        resourceusergroup(),
			"solidserver_cdb":              resourcecdb(),
			"solidserver_cdb_data":         resourcecdbdata(),
		},
		ConfigureContextFunc: ProviderConfigure,
	}
}

func ProviderConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	s, err := NewSOLIDserver(
		ctx,
		d.Get("host").(string),
		d.Get("username").(string),
		d.Get("password").(string),
		d.Get("sslverify").(bool),
		d.Get("additional_trust_certs_file").(string),
		d.Get("solidserverversion").(string),
	)
	return s, err
}
