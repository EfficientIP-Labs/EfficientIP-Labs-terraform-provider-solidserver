package solidserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"net/url"
)

func resourceipalias() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceipaliasCreate,
		ReadContext:   resourceipaliasRead,
		//UpdateContext: resourceipaliasUpdate,
		DeleteContext: resourceipaliasDelete,

		Schema: map[string]*schema.Schema{
			"space": {
				Type:        schema.TypeString,
				Description: "The name of the space to which the address belong to.",
				Required:    true,
				ForceNew:    true,
			},
			"address": {
				Type:         schema.TypeString,
				Description:  "The IP address for which the alias will be associated to.",
				ValidateFunc: validation.IsIPAddress,
				Required:     true,
				ForceNew:     true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The FQDN of the IP address alias to create.",
				Required:    true,
				ForceNew:    true,
			},
			"type": {
				Type:         schema.TypeString,
				Description:  "The type of the Alias to create (Supported: A, CNAME; Default: CNAME).",
				ValidateFunc: validation.StringInSlice([]string{"A", "CNAME"}, true),
				Default:      "CNAME",
				Optional:     true,
				ForceNew:     true,
			},
		},
	}
}

func resourceipaliasCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Gather required ID(s) from provided information
	siteID, err := ipsiteidbyname(d.Get("space").(string), meta)
	if err != nil {
		// Reporting a failure
		return diag.FromErr(err)
	}

	addressID, err := ipaddressidbyip(siteID, d.Get("address").(string), meta)
	if err != nil {
		// Reporting a failure
		return diag.FromErr(err)
	}

	// Building parameters
	parameters := url.Values{}
	parameters.Add("ip_id", addressID)
	parameters.Add("ip_name", d.Get("name").(string))
	parameters.Add("ip_name_type", d.Get("type").(string))

	// Sending the creation request
	resp, body, err := s.Request("post", "rest/ip_alias_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Created IP alias (oid): %s\n", oid))
				d.SetId(oid)

				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to create IP alias: %s - %s associated to IP address (OID): %s (%s)\n", d.Get("name").(string), d.Get("type"), addressID, errMsg)
			}
		}

		return diag.Errorf("Unable to create IP alias: %s - %s associated to IP address (OID): %s\n", d.Get("name").(string), d.Get("type"), addressID)
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceipaliasDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("ip_name_id", d.Id())

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/ip_alias_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return diag.Errorf("Unable to delete IP alias : %s - %s (%s)\n", d.Get("name"), d.Get("type"), errMsg)
				}
			}

			return diag.Errorf("Unable to delete IP alias : %s - %s\n", d.Get("name"), d.Get("type"))
		}

		// Log deletion
		tflog.Debug(ctx, fmt.Sprintf("Deleted IP alias with oid: %s\n", d.Id()))

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceipaliasRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Gather required ID(s) from provided information
	siteID, err := ipsiteidbyname(d.Get("space").(string), meta)
	if err != nil {
		// Reporting a failure
		return diag.FromErr(err)
	}

	addressID, err := ipaddressidbyip(siteID, d.Get("address").(string), meta)
	if err != nil {
		// Reporting a failure
		return diag.FromErr(err)
	}

	// Building parameters
	parameters := url.Values{}
	parameters.Add("ip_id", addressID)
	parameters.Add("WHERE", "ip_name_id='"+d.Id()+"'")

	// Sending the read request
	resp, body, err := s.Request("get", "rest/ip_alias_list", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", buf[0]["alias_name"].(string))
			d.Set("type", buf[0]["ip_name_type"].(string))

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find IP alias: %s (%s)\n", d.Get("name"), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find IP alias (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find IP alias: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}
