package solidserver

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceip6ptr() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceip6ptrRead,

		Schema: map[string]*schema.Schema{
			"address": {
				Type:         schema.TypeString,
				Description:  "The IPv6 address to convert into PTR domain name.",
				ValidateFunc: validation.IsIPAddress,
				Required:     true,
			},
			"dname": {
				Type:        schema.TypeString,
				Description: "The PTR record FQDN associated to the IPv6 address.",
				Computed:    true,
			},
		},
	}
}

func dataSourceip6ptrRead(d *schema.ResourceData, meta interface{}) error {
	dname := ip6toptr(d.Get("address").(string))

	if dname != "" {
		d.SetId(strconv.Itoa(rand.Intn(1000000)))
		d.Set("dname", dname)
		return nil
	}

	// Reporting a failure
	return fmt.Errorf("SOLIDServer - Unable to convert the following IPv6 address into PTR domain name: %s\n", d.Get("address").(string))
}
