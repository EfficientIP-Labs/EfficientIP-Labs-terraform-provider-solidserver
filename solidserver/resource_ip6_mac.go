package solidserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/url"
	"regexp"
	"strings"
)

func resourceip6mac() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceip6macCreate,
		ReadContext:   resourceip6macRead,
		DeleteContext: resourceip6macDelete,

		Schema: map[string]*schema.Schema{
			"space": {
				Type:        schema.TypeString,
				Description: "The name of the space into which mapping the IP and the MAC address.",
				Required:    true,
				ForceNew:    true,
			},
			"address": {
				Type:         schema.TypeString,
				Description:  "The IPv6 address to map with the MAC address.",
				ValidateFunc: validation.IsIPAddress,
				Required:     true,
				ForceNew:     true,
			},
			"mac": {
				Type:             schema.TypeString,
				Description:      "The MAC Address o map with the IPv6 address.",
				ValidateFunc:     validation.StringMatch(regexp.MustCompile("^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"), "Unsupported MAC address format."),
				DiffSuppressFunc: resourcediffsuppresscase,
				Required:         true,
				ForceNew:         true,
			},
		},
	}
}

func resourceip6macCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("site_name", d.Get("space").(string))
	parameters.Add("add_flag", "edit_only")
	parameters.Add("hostaddr", d.Get("address").(string))
	parameters.Add("ip6_mac_addr", strings.ToLower(d.Get("mac").(string)))
	parameters.Add("keep_class_parameters", "1")

	// Sending the creation request
	resp, body, err := s.Request("put", "rest/ip6_address6_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Created IP MAC association (oid) %s\n", oid))
				d.SetId(oid)
				return nil
			}
		} else {
			return diag.Errorf("Failed to create IP MAC association between %s and %s\n", d.Get("address").(string), d.Get("mac").(string))
		}
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceip6macDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("site_name", d.Get("space").(string))
	parameters.Add("add_flag", "edit_only")
	parameters.Add("hostaddr", d.Get("address").(string))
	parameters.Add("ip6_mac_addr", "")
	parameters.Add("keep_class_parameters", "1")

	// Sending the creation request
	resp, body, err := s.Request("put", "rest/ip6_address6_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Deleted IP MAC association (oid) %s\n", oid))
				d.SetId("")
				return nil
			}
		} else {
			return diag.Errorf("Failed to delete IP MAC association between %s and %s\n", d.Get("address").(string), d.Get("mac").(string))
		}
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceip6macRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("ip6_id", d.Id())

	tflog.Debug(ctx, fmt.Sprintf("Reading information about IPv6 address (oid): %s; associated to the mac: %s\n", d.Id(), d.Get("mac").(string)))

	// Sending the read request
	resp, body, err := s.Request("get", "rest/ip6_address6_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if ip6Mac, ip6MacExist := buf[0]["ip6_mac_addr"].(string); ip6MacExist {
				if strings.ToLower(ip6Mac) == strings.ToLower(d.Get("mac").(string)) {
					return nil
				}
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find the IPv6 address (oid): %s; associated to the mac (%s)\n", d.Id(), d.Get("mac").(string)))
			}
		} else {
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					// Log the error
					tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 address (oid): %s (%s)\n", d.Id(), errMsg))
				}
			} else {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 address (oid): %s\n", d.Id()))
			}
		}

		// Unset local ID
		d.SetId("")
	}

	return diag.FromErr(err)
}
