package solidserver

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"log"
	"net/url"
	"strconv"
	"strings"
)

func resourcednsrr() *schema.Resource {
	return &schema.Resource{
		Create: resourcednsrrCreate,
		Read:   resourcednsrrRead,
		Update: resourcednsrrUpdate,
		Delete: resourcednsrrDelete,
		Exists: resourcednsrrExists,
		Importer: &schema.ResourceImporter{
			State: resourcednsrrImportState,
		},

		Schema: map[string]*schema.Schema{
			"dnsserver": {
				Type:        schema.TypeString,
				Description: "The managed SMART DNS server name, or DNS server name hosting the RR's zone.",
				Required:    true,
				ForceNew:    true,
			},
			"dnsview": {
				Type:        schema.TypeString,
				Description: "The View name of the RR to create.",
				Optional:    true,
				ForceNew:    true,
				Default:     "",
			},
			"dnszone": {
				Type:        schema.TypeString,
				Description: "The Zone name of the RR to create.",
				Optional:    true,
				ForceNew:    true,
				Default:     "",
			},
			"name": {
				Type:        schema.TypeString,
				Description: "The Fully Qualified Domain Name of the RR to create.",
				Required:    true,
				ForceNew:    true,
			},
			"type": {
				Type:         schema.TypeString,
				Description:  "The type of the RR to create (Supported: A, AAAA, CNAME, TXT).",
				ValidateFunc: resourcednsrrvalidatetype,
				Required:     true,
				ForceNew:     true,
			},
			"value": {
				Type:             schema.TypeString,
				Description:      "The value od the RR to create.",
				Computed:         false,
				Required:         true,
				ForceNew:         true,
				DiffSuppressFunc: resourcediffsuppressIPv6Format,
			},
			"ttl": {
				Type:        schema.TypeInt,
				Description: "The DNS Time To Live of the RR to create.",
				Optional:    true,
				Default:     3600,
			},
		},
	}
}

func resourcednsrrvalidatetype(v interface{}, _ string) ([]string, []error) {
	switch strings.ToUpper(v.(string)) {
	case "A":
		return nil, nil
	case "AAAA":
		return nil, nil
	case "PTR":
		return nil, nil
	case "CNAME":
		return nil, nil
	case "DNAME":
		return nil, nil
	case "TXT":
		return nil, nil
	case "NS":
		return nil, nil
	default:
		return nil, []error{fmt.Errorf("Unsupported RR type.")}
	}
}

func resourcednsrrExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}

	// Sending the read request
	//parameters.Add("rr_id", d.Id())
	//log.Printf("[DEBUG] Checking existence of RR (oid): %s\n", d.Id())
	//resp, body, err := s.Request("get", "rest/dns_rr_info", &parameters)

	// Attempt to not rely on the ID that may change due to DNS behavior
	whereClause := "dns_name='" + d.Get("dnsserver").(string) + "' AND rr_full_name='" + d.Get("name").(string) + "' AND rr_type='" + strings.ToUpper(d.Get("type").(string))

	// FIXME - Must convert IPv6 short to long
	if strings.ToUpper(d.Get("type").(string)) == "AAAA" {
		value := shortip6tolongip6(d.Get("value").(string))
		log.Printf("[DEBUG] SOLIDServer - Using Expanded IPv6 format: %s\n", value)
		whereClause += "' AND value1='" + value + "' "
	} else {
		whereClause += "' AND value1='" + d.Get("value").(string) + "' "
	}

	// Attempt to hande changing RR IDs
	if len(d.Get("dnsview").(string)) != 0 {
		whereClause += "AND dnsview_name='" + d.Get("dnsview").(string) + "' "
	} else {
		whereClause += "AND dnsview_name='#' "
	}

	// Add dnszone parameter if it is supplied
	if len(d.Get("dnszone").(string)) != 0 {
		whereClause += "AND dnszone_name='" + d.Get("dnszone").(string) + "' "
	}

	parameters.Add("WHERE", whereClause)
	resp, body, err := s.Request("get", "rest/dns_rr_list", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["rr_id"].(string); oidExist {
				d.SetId(oid)
			}
			return true, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				log.Printf("[DEBUG] SOLIDServer - Unable to find RR (oid): %s (%s)\n", d.Id(), errMsg)
			}
		} else {
			// Log the error
			log.Printf("[DEBUG] SOLIDServer - Unable to find RR (oid): %s\n", d.Id())
		}

		// Unset local ID
		d.SetId("")
	}

	// Reporting a failure
	return false, err
}

func resourcednsrrCreate(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("add_flag", "new_only")
	parameters.Add("dns_name", d.Get("dnsserver").(string))
	parameters.Add("rr_name", d.Get("name").(string))
	parameters.Add("rr_type", strings.ToUpper(d.Get("type").(string)))
	parameters.Add("value1", d.Get("value").(string))
	parameters.Add("rr_ttl", strconv.Itoa(d.Get("ttl").(int)))

	// Add dnsview parameter if it is supplied
	if len(d.Get("dnsview").(string)) != 0 {
		parameters.Add("dnsview_name", strings.ToLower(d.Get("dnsview").(string)))
	}

	// Add dnszone parameter if it is supplied
	if len(d.Get("dnszone").(string)) != 0 {
		parameters.Add("dnszone_name", strings.ToLower(d.Get("dnszone").(string)))
	}

	// Sending the creation request
	resp, body, err := s.Request("post", "rest/dns_rr_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				log.Printf("[DEBUG] SOLIDServer - Created RR (oid): %s\n", oid)
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return fmt.Errorf("SOLIDServer - Unable to create RR: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return fmt.Errorf("SOLIDServer - Unable to create RR: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return err
}

func resourcednsrrUpdate(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("rr_id", d.Id())
	parameters.Add("add_flag", "edit_only")
	parameters.Add("dns_name", d.Get("dnsserver").(string))
	parameters.Add("rr_name", d.Get("name").(string))
	parameters.Add("rr_type", strings.ToUpper(d.Get("type").(string)))
	parameters.Add("value1", d.Get("value").(string))
	parameters.Add("rr_ttl", strconv.Itoa(d.Get("ttl").(int)))

	// Add dnsview parameter if it is supplied
	if len(d.Get("dnsview").(string)) != 0 {
		parameters.Add("dnsview_name", strings.ToLower(d.Get("dnsview").(string)))
	}

	// Add dnszone parameter if it is supplied
	if len(d.Get("dnszone").(string)) != 0 {
		parameters.Add("dnszone_name", strings.ToLower(d.Get("dnszone").(string)))
	}

	// Sending the update request
	resp, body, err := s.Request("put", "rest/dns_rr_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				log.Printf("[DEBUG] SOLIDServer - Updated RR (oid): %s\n", oid)
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return fmt.Errorf("SOLIDServer - Unable to update RR: %s (%s)", d.Get("name").(string), errMsg)
			}
		}

		return fmt.Errorf("SOLIDServer - Unable to update RR: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return err
}

func resourcednsrrDelete(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("rr_id", d.Id())

	// Add dnsview parameter if it is supplied
	if len(d.Get("dnsview").(string)) != 0 {
		parameters.Add("dnsview_name", strings.ToLower(d.Get("dnsview").(string)))
	}

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/dns_rr_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return fmt.Errorf("SOLIDServer - Unable to delete RR: %s (%s)", d.Get("name").(string), errMsg)
				}
			}

			return fmt.Errorf("SOLIDServer - Unable to delete RR: %s", d.Get("name").(string))
		}

		// Log deletion
		log.Printf("[DEBUG] SOLIDServer - Deleted RR (oid): %s\n", d.Id())

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return err
}

func resourcednsrrRead(d *schema.ResourceData, meta interface{}) error {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}

	// Sending the read request
	//parameters.Add("rr_id", d.Id())
	//resp, body, err := s.Request("get", "rest/dns_rr_info", &parameters)

	// Attempt to not rely on the ID that may change due to DNS behavior
	whereClause := "dns_name='" + d.Get("dnsserver").(string) + "' AND rr_full_name='" + d.Get("name").(string) + "' AND rr_type='" + strings.ToUpper(d.Get("type").(string))

	// FIXME - Must convert IPv6 short to long
	if strings.ToUpper(d.Get("type").(string)) == "AAAA" {
		value := shortip6tolongip6(d.Get("value").(string))
		log.Printf("[DEBUG] SOLIDServer - Using Expanded IPv6 format: %s\n", value)
		whereClause += "' AND value1='" + value + "' "
	} else {
		whereClause += "' AND value1='" + d.Get("value").(string) + "' "
	}

	// Attempt to hande changing RR IDs
	if len(d.Get("dnsview").(string)) != 0 {
		whereClause += "AND dnsview_name='" + d.Get("dnsview").(string) + "' "
	} else {
		whereClause += "AND dnsview_name='#' "
	}

	// Add dnszone parameter if it is supplied
	if len(d.Get("dnszone").(string)) != 0 {
		whereClause += "AND dnszone_name='" + d.Get("dnszone").(string) + "' "
	}

	parameters.Add("WHERE", whereClause)
	resp, body, err := s.Request("get", "rest/dns_rr_list", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			if oid, oidExist := buf[0]["rr_id"].(string); oidExist {
				d.SetId(oid)
			}

			ttl, _ := strconv.Atoi(buf[0]["ttl"].(string))

			d.Set("dnsserver", buf[0]["dns_name"].(string))
			d.Set("name", buf[0]["rr_full_name"].(string))
			d.Set("type", buf[0]["rr_type"].(string))

			if strings.ToUpper(buf[0]["rr_type"].(string)) == "AAAA" {
				d.Set("value", longip6toshortip6(buf[0]["value1"].(string)))
			} else {
				d.Set("value", buf[0]["value1"].(string))
			}

			d.Set("ttl", ttl)

			if buf[0]["dnsview_name"].(string) != "#" {
				d.Set("dnsview", buf[0]["dnsview_name"].(string))
			}

			return nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				log.Printf("[DEBUG] SOLIDServer - Unable to find RR: %s (%s)\n", d.Get("name"), errMsg)
			}
		} else {
			// Log the error
			log.Printf("[DEBUG] SOLIDServer - Unable to find RR (oid): %s\n", d.Id())
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return fmt.Errorf("SOLIDServer - Unable to find RR: %s\n", d.Get("name").(string))
	}

	// Reporting a failure
	return err
}

func resourcednsrrImportState(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("rr_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/dns_rr_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			ttl, _ := strconv.Atoi(buf[0]["ttl"].(string))

			d.Set("dnsserver", buf[0]["dns_name"].(string))
			d.Set("name", buf[0]["rr_full_name"].(string))
			d.Set("type", buf[0]["rr_type"].(string))

			if strings.ToUpper(buf[0]["rr_type"].(string)) == "AAAA" {
				d.Set("value", longip6toshortip6(buf[0]["value1"].(string)))
			} else {
				d.Set("value", buf[0]["value1"].(string))
			}

			d.Set("ttl", ttl)

			if buf[0]["dnsview_name"].(string) != "#" {
				d.Set("dnsview", buf[0]["dnsview_name"].(string))
			}

			return []*schema.ResourceData{d}, nil
		}

		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				// Log the error
				log.Printf("[DEBUG] SOLIDServer - Unable to import RR (oid): %s (%s)\n", d.Id(), errMsg)
			}
		} else {
			// Log the error
			log.Printf("[DEBUG] SOLIDServer - Unable to find and import RR (oid): %s\n", d.Id())
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import RR (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}
