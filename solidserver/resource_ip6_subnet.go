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
	"math/big"
	"math/rand"
	"net/url"
	"strconv"
	"time"
)

func resourceip6subnet() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceip6subnetCreate,
		ReadContext:   resourceip6subnetRead,
		UpdateContext: resourceip6subnetUpdate,
		DeleteContext: resourceip6subnetDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceip6subnetImportState,
		},

		Description: heredoc.Doc(`
			IPv6 Subnet resource allows to create and manage IPAM networks that are key to organize the IP space
			Subnet can be blocks or subnets. Blocks reflect the assigned IP ranges (RFC1918 or public prefixes).
			Subnets reflect the internal sub-division of your network.
		`),

		Schema: map[string]*schema.Schema{
			"space": {
				Type:        schema.TypeString,
				Description: "The name of the space into which creating the IPv6 subnet.",
				Required:    true,
				ForceNew:    true,
			},
			"block": {
				Type:        schema.TypeString,
				Description: "The name of the block intyo which creating the IPv6 subnet.",
				Optional:    true,
				ForceNew:    true,
			},
			"request_ip": {
				Type:         schema.TypeString,
				Description:  "The optionally requested subnet IPv6 address.",
				ValidateFunc: validation.IsIPAddress,
				Optional:     true,
				ForceNew:     true,
				Default:      "",
			},
			"prefix_size": {
				Type:        schema.TypeInt,
				Description: "The expected IPv6 subnet's prefix length (ex: 24 for a '/24').",
				Required:    true,
				ForceNew:    true,
			},
			"prefix": {
				Type:        schema.TypeString,
				Description: "The provisionned IPv6 prefix.",
				Computed:    true,
			},
			"address": {
				Type:        schema.TypeString,
				Description: "The provisionned IPv6 network address.",
				Computed:    true,
				ForceNew:    true,
			},
			"gateway_offset": {
				Type:        schema.TypeInt,
				Description: "Offset for creating the gateway. Default is 0 (No gateway).",
				Optional:    true,
				ForceNew:    true,
				Default:     0,
			},
			"gateway": {
				Type:        schema.TypeString,
				Description: "The subnet's computed gateway.",
				Computed:    true,
				ForceNew:    true,
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the IPv6 subnet to create.",
				Required:    true,
				ForceNew:    false,
			},
			"terminal": {
				Type:        schema.TypeBool,
				Description: "The terminal property of the IPv6 subnet.",
				Optional:    true,
				ForceNew:    true,
				Default:     true,
			},
			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the IPv6 subnet.",
				Optional:    true,
				ForceNew:    false,
				Default:     "",
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to the IPv6 subnet.",
				Optional:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func resourceip6subnetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	blockInfo := make(map[string]interface{})
	s := meta.(*SOLIDserver)
	var gateway string = ""

	// Gather required ID(s) from provided information
	siteID, siteErr := ipsiteidbyname(d.Get("space").(string), meta)
	if siteErr != nil {
		// Reporting a failure
		return diag.FromErr(siteErr)
	}

	// If a block is specified, look for free IP subnet within this block
	if len(d.Get("block").(string)) > 0 {
		var blockErr error = nil

		blockInfo, blockErr = ip6subnetinfobyname(siteID, d.Get("block").(string), false, meta)

		if blockErr != nil {
			// Reporting a failure
			return diag.FromErr(blockErr)
		}
	} else {
		// Otherwise, set an empty blockInfo's ID by default
		blockInfo["id"] = ""

		// However, we can't create a block as a terminal subnet
		if d.Get("terminal").(bool) {
			return diag.Errorf("Can't create a terminal IPv6 block subnet: %s", d.Get("name").(string))
		}
	}

	subnetAddresses, subnetErr := ip6subnetfindbysize(siteID, blockInfo["id"].(string), d.Get("request_ip").(string), d.Get("prefix_size").(int), meta)

	if subnetErr != nil {
		// Reporting a failure
		return diag.FromErr(subnetErr)
	}

	for i := 0; i < len(subnetAddresses); i++ {
		// Building parameters
		parameters := url.Values{}
		parameters.Add("site_id", siteID)
		parameters.Add("add_flag", "new_only")
		parameters.Add("subnet6_name", d.Get("name").(string))
		parameters.Add("subnet6_addr", hexip6toip6(subnetAddresses[i]))
		parameters.Add("subnet6_prefix", strconv.Itoa(d.Get("prefix_size").(int)))
		parameters.Add("subnet6_class_name", d.Get("class").(string))

		// If no block specified, create an IP block
		if len(d.Get("block").(string)) == 0 {
			parameters.Add("subnet_level", "0")
		} else {
			parameters.Add("use_reversed_relative_position", "1")
			parameters.Add("relative_position", "0")
		}

		// Specify if subnet is terminal
		if d.Get("terminal").(bool) {
			parameters.Add("is_terminal", "1")
		} else {
			parameters.Add("is_terminal", "0")
		}

		// Building class_parameters
		classParameters := url.Values{}

		// Generate class parameter for the gateway if required
		goffset := d.Get("gateway_offset").(int)

		if goffset != 0 {
			bigStartAddr, _ := new(big.Int).SetString(subnetAddresses[i], 16)

			if goffset > 0 {
				bigOffset := big.NewInt(int64(goffset))
				gateway = hexip6toip6(BigIntToHexStr(bigStartAddr.Add(bigStartAddr, bigOffset)))
			} else {
				bigEndAddr := bigStartAddr.Add(bigStartAddr, prefix6lengthtosize(int64(d.Get("prefix_size").(int))))
				bigOffset := big.NewInt(int64(abs(goffset)))
				gateway = hexip6toip6(BigIntToHexStr(bigEndAddr.Sub(bigEndAddr, bigOffset)))
			}

			classParameters.Add("gateway", gateway)
			tflog.Debug(ctx, fmt.Sprintf("Subnet computed gateway: %s\n", gateway))
		}

		for k, v := range d.Get("class_parameters").(map[string]interface{}) {
			classParameters.Add(k, v.(string))
		}
		parameters.Add("subnet6_class_parameters", classParameters.Encode())

		// Random Delay
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		// Sending the creation request
		resp, body, err := s.Request("post", "rest/ip6_subnet6_add", &parameters)

		if err == nil {
			var buf [](map[string]interface{})
			json.Unmarshal([]byte(body), &buf)

			prefix := hexip6toip6(subnetAddresses[i]) + "/" + strconv.Itoa(d.Get("prefix_size").(int))

			// Checking the answer
			if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
				if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
					tflog.Debug(ctx, fmt.Sprintf("Created IPv6 subnet (oid): %s\n", oid))
					d.SetId(oid)
					d.Set("prefix", prefix)
					d.Set("address", hexip6toip6(subnetAddresses[i]))
					if goffset != 0 {
						d.Set("gateway", gateway)
					}
					return nil
				}
			} else {
				if len(buf) > 0 {
					if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
						tflog.Debug(ctx, fmt.Sprintf("Failed IP subnet registration for IPv6 subnet: %s with prefix: %s (%s)\n", d.Get("name").(string), prefix, errMsg))
					} else {
						tflog.Debug(ctx, fmt.Sprintf("Failed IP subnet registration for IPv6 subnet: %s with prefix: %s\n", d.Get("name").(string), prefix))
					}
				} else {
					tflog.Debug(ctx, fmt.Sprintf("Failed IP subnet registration for IPv6 subnet: %s with prefix: %s\n", d.Get("name").(string), prefix))
				}
			}
		} else {
			// Reporting a failure
			return diag.FromErr(err)
		}
	}

	// Reporting a failure
	return diag.Errorf("Unable to create IPv6 subnet: %s, unable to find a suitable prefix\n", d.Get("name").(string))
}

func resourceip6subnetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("subnet6_id", d.Id())
	parameters.Add("add_flag", "edit_only")
	parameters.Add("subnet6_name", d.Get("name").(string))
	parameters.Add("subnet6_class_name", d.Get("class").(string))

	if d.Get("terminal").(bool) {
		parameters.Add("is_terminal", "1")
	} else {
		parameters.Add("is_terminal", "0")
	}

	// Building class_parameters
	classParameters := url.Values{}

	// Generate class parameter for the gateway if required
	goffset := d.Get("gateway_offset").(int)

	if goffset != 0 {
		classParameters.Add("gateway", d.Get("gateway").(string))
		tflog.Debug(ctx, fmt.Sprintf("Subnet updated gateway: %s\n", d.Get("gateway").(string)))
	}

	for k, v := range d.Get("class_parameters").(map[string]interface{}) {
		classParameters.Add(k, v.(string))
	}

	parameters.Add("subnet6_class_parameters", classParameters.Encode())

	// Sending the update request
	resp, body, err := s.Request("put", "rest/ip6_subnet6_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Updated IPv6 subnet (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to update IPv6 subnet: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return diag.Errorf("Unable to update IPv6 subnet: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceip6subnetgatewayDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	if d.Get("gateway") != nil {
		// Building parameters
		parameters := url.Values{}
		parameters.Add("site_name", d.Get("space").(string))
		parameters.Add("hostaddr", d.Get("gateway").(string))

		// Sending the deletion request
		resp, body, err := s.Request("delete", "rest/ip6_address6_delete", &parameters)

		if err == nil {
			var buf [](map[string]interface{})
			json.Unmarshal([]byte(body), &buf)

			// Checking the answer
			if resp.StatusCode != 200 && resp.StatusCode != 204 {
				// Reporting a failure
				if len(buf) > 0 {
					if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
						tflog.Debug(ctx, fmt.Sprintf("Unable to delete IPv6 subnet's gateway: %s (%s)", d.Get("gateway").(string), errMsg))
					}
				}

				tflog.Debug(ctx, fmt.Sprintf("Unable to delete IPv6 subnet's gateway: %s", d.Get("gateway").(string)))
			}

			// Log deletion
			tflog.Debug(ctx, fmt.Sprintf("Deleted IPv6 subnet's gateway: %s\n", d.Get("gateway").(string)))

			// Reporting a success
			return nil
		}

		// Reporting a failure
		return diag.FromErr(err)
	}

	// Reporting a success (nothing done)
	return nil
}

func resourceip6subnetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Delete related resources such as the Gateway
	if d.Get("gateway_offset") != 0 {
		resourceip6subnetgatewayDelete(ctx, d, meta)
	}

	// Building parameters
	parameters := url.Values{}
	parameters.Add("subnet6_id", d.Id())

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/ip6_subnet6_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return diag.Errorf("Unable to delete IPv6 subnet : %s (%s)", d.Get("name").(string), errMsg)
				}
			}

			return diag.Errorf("Unable to delete IPv6 subnet : %s", d.Get("name").(string))
		}

		// Log deletion
		tflog.Debug(ctx, fmt.Sprintf("Deleted IPv6 subnet (oid): %s\n", d.Id()))

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceip6subnetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("subnet6_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/ip6_block6_subnet6_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("space", buf[0]["site_name"].(string))
			d.Set("block", buf[0]["parent_subnet6_name"].(string))
			d.Set("name", buf[0]["subnet6_name"].(string))
			d.Set("class", buf[0]["subnet6_class_name"].(string))

			if buf[0]["is_terminal"].(string) == "1" {
				d.Set("terminal", true)
			} else {
				d.Set("terminal", false)
			}

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["subnet6_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			if gateway, gatewayExist := retrievedClassParameters["gateway"]; gatewayExist {
				d.Set("gateway", gateway[0])
			}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 subnet: %s (%s)\n", d.Get("name"), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find IPv6 subnet (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find IPv6 subnet: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourceip6subnetImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("subnet6_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/ip6_block6_subnet6_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("space", buf[0]["site_name"].(string))
			d.Set("block", buf[0]["parent_subnet6_name"].(string))
			d.Set("name", buf[0]["subnet6_name"].(string))
			d.Set("class", buf[0]["subnet6_class_name"].(string))

			if buf[0]["is_terminal"].(string) == "1" {
				d.Set("terminal", true)
			} else {
				d.Set("terminal", false)
			}

			// Setting local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["subnet6_class_parameters"].(string))
			computedClassParameters := map[string]string{}

			if gateway, gatewayExist := retrievedClassParameters["gateway"]; gatewayExist {
				d.Set("gateway", gateway[0])
			}

			for ck := range currentClassParameters {
				if rv, rvExist := retrievedClassParameters[ck]; rvExist {
					computedClassParameters[ck] = rv[0]
				} else {
					computedClassParameters[ck] = ""
				}
			}

			d.Set("class_parameters", computedClassParameters)

			return []*schema.ResourceData{d}, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				tflog.Debug(ctx, fmt.Sprintf("Unable to import IPv6 subnet (oid): %s (%s)\n", d.Id(), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find and import IPv6 subnet (oid): %s\n", d.Id()))
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import IPv6 subnet (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}
