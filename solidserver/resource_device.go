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
	"strings"
)

func resourcedevice() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcedeviceCreate,
		ReadContext:   resourcedeviceRead,
		UpdateContext: resourcedeviceUpdate,
		DeleteContext: resourcedeviceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcedeviceImportState,
		},

		Description: heredoc.Doc(`
			Device resource allows to create and manage network devices and link them with IP addresses.
			It is pretty useful to link together the many IP addresses a device can rely on.
		`),

		Schema: map[string]*schema.Schema{
			"name": {
				Type:             schema.TypeString,
				Description:      "The name of the device to create.",
				ValidateFunc:     resourcedevicenamevalidateformat,
				DiffSuppressFunc: resourcediffsuppresscase,
				Required:         true,
				ForceNew:         true,
			},
			"class": {
				Type:        schema.TypeString,
				Description: "The class associated to the device.",
				Optional:    true,
				ForceNew:    false,
				Default:     "",
			},
			"class_parameters": {
				Type:        schema.TypeMap,
				Description: "The class parameters associated to device.",
				Optional:    true,
				ForceNew:    false,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// Validate device name format against the hostname regexp
func resourcedevicenamevalidateformat(v interface{}, _ string) ([]string, []error) {
	if match, _ := regexp.MatchString(regexpHostname, v.(string)); match == true {
		return nil, nil
	}

	return nil, []error{fmt.Errorf("Unsupported device name format (it must comply with hostname standard).\n")}
}

func resourcedeviceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("add_flag", "new_only")
	parameters.Add("hostdev_name", strings.ToLower(d.Get("name").(string)))
	parameters.Add("hostdev_class_name", d.Get("class").(string))
	parameters.Add("hostdev_class_parameters", urlfromclassparams(d.Get("class_parameters")).Encode())

	// Sending creation request
	resp, body, err := s.Request("post", "rest/hostdev_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Created device (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to create device: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
			}
		}

		return diag.Errorf("Unable to create device: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcedeviceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("hostdev_id", d.Id())
	parameters.Add("add_flag", "edit_only")
	parameters.Add("hostdev_name", strings.ToLower(d.Get("name").(string)))
	parameters.Add("hostdev_class_name", d.Get("class").(string))
	parameters.Add("hostdev_class_parameters", urlfromclassparams(d.Get("class_parameters")).Encode())

	// Sending the update request
	resp, body, err := s.Request("put", "rest/hostdev_add", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if (resp.StatusCode == 200 || resp.StatusCode == 201) && len(buf) > 0 {
			if oid, oidExist := buf[0]["ret_oid"].(string); oidExist {
				tflog.Debug(ctx, fmt.Sprintf("Updated device (oid): %s\n", oid))
				d.SetId(oid)
				return nil
			}
		}

		// Reporting a failure
		if len(buf) > 0 {
			if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
				return diag.Errorf("Unable to update device: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
			}
		}

		return diag.Errorf("Unable to update device: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcedeviceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("hostdev_id", d.Id())

	// Sending the deletion request
	resp, body, err := s.Request("delete", "rest/hostdev_delete", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode != 200 && resp.StatusCode != 204 {
			// Reporting a failure
			if len(buf) > 0 {
				if errMsg, errExist := buf[0]["errmsg"].(string); errExist {
					return diag.Errorf("Unable to delete device: %s (%s)", strings.ToLower(d.Get("name").(string)), errMsg)
				}
			}

			return diag.Errorf("Unable to delete device: %s", strings.ToLower(d.Get("name").(string)))
		}

		// Log deletion
		tflog.Debug(ctx, fmt.Sprintf("Deleted device (oid): %s\n", d.Id()))

		// Unset local ID
		d.SetId("")

		// Reporting a success
		return nil
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcedeviceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("hostdev_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/hostdev_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", strings.ToLower(buf[0]["hostdev_name"].(string)))
			d.Set("class", buf[0]["hostdev_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["hostdev_class_parameters"].(string))
			computedClassParameters := map[string]string{}

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
				tflog.Debug(ctx, fmt.Sprintf("Unable to find device: %s (%s)\n", strings.ToLower(d.Get("name").(string)), errMsg))
			}
		} else {
			// Log the error
			tflog.Debug(ctx, fmt.Sprintf("Unable to find device (oid): %s\n", d.Id()))
		}

		// Do not unset the local ID to avoid inconsistency

		// Reporting a failure
		return diag.Errorf("Unable to find device: %s\n", strings.ToLower(d.Get("name").(string)))
	}

	// Reporting a failure
	return diag.FromErr(err)
}

func resourcedeviceImportState(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	s := meta.(*SOLIDserver)

	// Building parameters
	parameters := url.Values{}
	parameters.Add("hostdev_id", d.Id())

	// Sending the read request
	resp, body, err := s.Request("get", "rest/hostdev_info", &parameters)

	if err == nil {
		var buf [](map[string]interface{})
		json.Unmarshal([]byte(body), &buf)

		// Checking the answer
		if resp.StatusCode == 200 && len(buf) > 0 {
			d.Set("name", strings.ToLower(buf[0]["hostdev_name"].(string)))
			d.Set("class", buf[0]["hostdev_class_name"].(string))

			// Updating local class_parameters
			currentClassParameters := d.Get("class_parameters").(map[string]interface{})
			retrievedClassParameters, _ := url.ParseQuery(buf[0]["hostdev_class_parameters"].(string))
			computedClassParameters := map[string]string{}

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
				tflog.Debug(ctx, fmt.Sprintf("Unable to import device(oid): %s (%s)\n", d.Id(), errMsg))
			}
		} else {
			tflog.Debug(ctx, fmt.Sprintf("Unable to find and import device (oid): %s\n", d.Id()))
		}

		// Reporting a failure
		return nil, fmt.Errorf("SOLIDServer - Unable to find and import device (oid): %s\n", d.Id())
	}

	// Reporting a failure
	return nil, err
}
