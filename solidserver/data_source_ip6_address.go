package solidserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"net/url"
	"regexp"
	"strconv"
)

func dataSourceip6address() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceip6addressRead,

		Description: heredoc.Doc(`
			IPv6 address data-source allows to retrieve information about reserved IPv6 addresses including meta-data.
		`),

		Schema: map[string]*schema.Schema{
			"space": {
				Type:        schema.TypeString,
				Description: "The name of the space of the IPv6 address.",
				Required:    true,
			},
			"subnet": {
				Type:        schema.TypeString,
				Description: "The name of the subnet of the IPv6 address.",
				Computed:    true,
			},
			"pool": {
				Type:        schema.TypeString,
				Description: "The name of the pool of the IPv6 address.",
				Computed:    true,
			},
			"address": {
				Type:        schema.TypeString,
				Description: "The IPv6 address.",
				Required:    true,
			},
			"device": {
				Type:        schema.TypeString,
				Description: "Device Name associated to the IPv6 address (Require a 'Device Manager' license).",
				Computed:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The short name or FQDN of the IPv6 address.",
				Computed:    true,
			},
			"mac": {
				Type:        schema.TypeString,
				Description: "The MAC Address of the IPv6 address.",
				Computed:    true,
			},
			"prefix": {
				Type:        schema.TypeString,
				Description: "The IPv6 address prefix.",
				Computed:    true,
			},
			"prefix_size": {
				Type:        schema.TypeInt,
				Description: "The prefix_length associated to the IPv6 address.",
				Computed:    true,
			},
			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the IPv6 address.",
				Computed:    true,
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to the IPv6 address.",
				Computed:    true,
			},
		},
	}
}

func dataSourceip6addressRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("WHERE", "site_name='"+d.Get("space").(string)+"' AND ip6_addr='"+ip6tohexip6(d.Get("address").(string))+"'")

	// Sending the read request
	resp, body, err := s.Request("get", "rest/ip6_address6_list", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.SetId(buf[0]["ip6_id"].(string))
			d.Set("space", buf[0]["site_name"].(string))
			d.Set("subnet", buf[0]["subnet6_name"].(string))
			d.Set("pool", buf[0]["pool6_name"].(string))
			d.Set("name", buf[0]["ip6_name"].(string))
			d.Set("device", buf[0]["hostdev_name"].(string))

			prefix_size, _ := strconv.Atoi(buf[0]["subnet6_prefix"].(string))

			d.Set("prefix", hexip6toip6(buf[0]["subnet6_start_ip6_addr"].(string))+"/"+buf[0]["subnet6_prefix"].(string))
			d.Set("prefix_size", prefix_size)

			if macIgnore, _ := regexp.MatchString("^EIP:", buf[0]["ip6_mac_addr"].(string)); !macIgnore {
				d.Set("mac", buf[0]["ip6_mac_addr"].(string))
			} else {
				d.Set("mac", "")
			}

			d.Set("class", buf[0]["ip6_class_name"].(string))

			// Updating local class_parameters
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["ip6_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			for ck := range retrievedClassParameters {
				if ck != "gateway" {
					computedClassParameters[ck] = retrievedClassParameters[ck][0]
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 address: %s (%s)\n", d.Get("name"), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 address (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find IPv6 address: %s", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}
