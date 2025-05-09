package linux

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

func userResource() *schema.Resource {
	return &schema.Resource{
		Create: userResourceCreate,
		Read:   userResourceRead,
		Update: userResourceUpdate,
		Delete: userResourceDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"shell": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/bin/bash",
			},
			"home": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return new == "" || old == new
				},
			},
			"create_home": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == "" || old == new
				},
			},
			"uid": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"gid": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return k == "groups.#" || new == ""
				},
			},
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"system": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					return old == "" || old == new
				},
			},
		},
	}
}

func userResourceCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	name := d.Get("name").(string)
	uid := d.Get("uid").(int)
	gid := d.Get("gid").(int)
	system := d.Get("system").(bool)
	comment := d.Get("comment").(string)
	home := d.Get("home").(string)
	create_home := d.Get("create_home").(bool)
	shell := d.Get("shell").(string)
	groups := d.Get("groups").(*schema.Set).List()
	groupsList := make([]string, len(groups))
	for i, group := range groups {
		groupsList[i] = group.(string)
	}

	err := createUser(client, name, uid, gid, system, comment, home, create_home, shell, groupsList)
	if err != nil {
		return errors.Wrap(err, "Couldn't create user")
	}

	uid, err = getUserId(client, name)
	if err != nil {
		return errors.Wrap(err, "Couldn't get uid")
	}

	d.Set("uid", uid)

	d.SetId(fmt.Sprintf("%v", uid))
	return userResourceRead(d, m)
}

func createUser(client *Client, name string, uid int, gid int, system bool, comment string, home string, create_home bool, shell string, groups []string) error {
	command := "/usr/sbin/useradd"

	if len(home) > 0 {
		command = fmt.Sprintf("%s --home-dir %s", command, home)
	} else {
		command = fmt.Sprintf("%s --home-dir /home/%s", command, name)
	}
	if create_home {
		command = fmt.Sprintf("%s --create-home", command)
	}
	if len(comment) > 0 {
		command = fmt.Sprintf("%s --comment '%s'", command, comment)
	}
	if len(shell) > 0 {
		command = fmt.Sprintf("%s --shell %s", command, shell)
	}
	if uid > 0 {
		command = fmt.Sprintf("%s --uid %d", command, uid)
	}
	if gid > 0 {
		command = fmt.Sprintf("%s --gid %d", command, gid)
	}
	if len(groups) > 0 {
		command = fmt.Sprintf("%s --groups %s", command, strings.Join(groups, ","))
	}
	if system {
		command = fmt.Sprintf("%s --system", command)
	}
	command = fmt.Sprintf("%s %s", command, name)
	_, _, err := runCommand(client, true, command, "")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	return nil
}

func getUserId(client *Client, name string) (int, error) {
	command := fmt.Sprintf("id --user %s", name)
	stdout, _, err := runCommand(client, false, command, "")
	if err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	if stdout == "" {
		return 0, fmt.Errorf("User not found with name %v", name)
	}
	uid, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		return 0, err
	}
	return uid, nil
}

func getUserFromID(client *Client, uid int) ([]string, error) {
	command := fmt.Sprintf("getent passwd %d", uid)
	stdout, _, err := runCommand(client, false, command, "")
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	if stdout == "" {
		return nil, fmt.Errorf("User not found with id %v", uid)
	}
	data := strings.Split(strings.TrimSpace(stdout), ":")
	return data, nil
}

func getUserFromName(client *Client, name string) ([]string, error) {
	command := fmt.Sprintf("getent passwd %s", name)
	stdout, _, err := runCommand(client, false, command, "")
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	if stdout == "" {
		return nil, fmt.Errorf("User not found with name %v", name)
	}
	data := strings.Split(strings.TrimSpace(stdout), ":")
	return data, nil
}

func getGroupIdForUser(_ *Client, details []string) (int, error) {
	uid, err := strconv.Atoi(strings.TrimSpace(details[3]))
	if err != nil {
		return 0, err
	}
	return uid, nil
}

func getUserGroups(client *Client, name string) ([]string, error) {
	command := fmt.Sprintf("id --name --groups %s", name)
	stdout, _, err := runCommand(client, false, command, "")
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	if stdout == "" {
		return nil, fmt.Errorf("User not found with name %v", name)
	}
	return strings.Split(strings.TrimSpace(stdout), " "), nil
}

func userResourceRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	uid, err := strconv.Atoi(d.Id())
	if err != nil {
		return errors.Wrap(err, "ID stored is not int")
	}
	details, err := getUserFromID(client, uid)
	if err != nil {
		d.SetId("")
		return nil
	}
	d.Set("name", details[0])
	gid, err := getGroupIdForUser(client, details)
	if err != nil {
		return errors.Wrap(err, "Couldn't find group for user")
	}
	d.Set("gid", gid)
	d.Set("comment", details[4])
	d.Set("home", details[5])
	d.Set("shell", details[6])
	groups, err := getUserGroups(client, details[0])
	if err != nil {
		return errors.Wrap(err, "Couldn't find group for user")
	}
	d.Set("groups", groups)
	return nil
}

func userResourceUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	uid, err := strconv.Atoi(d.Id())
	if err != nil {
		return errors.Wrap(err, "ID stored is not int")
	}
	old, err := getUserFromID(client, uid)
	if err != nil {
		return errors.Wrap(err, "Failed to get user name")
	}
	command := "/usr/sbin/usermod"

	if d.HasChange("name") {
		command = fmt.Sprintf("%s --login %s", command, d.Get("name").(string))
	}
	if d.HasChange("gid") {
		command = fmt.Sprintf("%s --gid %d", command, d.Get("gid").(int))
	}
	if d.HasChange("home") {
		command = fmt.Sprintf("%s --move-home --home '%s'", command, d.Get("home").(string))
	}
	if d.HasChange("shell") {
		command = fmt.Sprintf("%s --shell %s", command, d.Get("shell").(string))
	}
	if d.HasChange("comment") {
		command = fmt.Sprintf("%s --comment '%s'", command, d.Get("comment").(string))
	}
	if d.HasChange("groups") {
		groups := d.Get("groups").(*schema.Set).List()
		groupsList := make([]string, len(groups))
		for i, group := range groups {
			groupsList[i] = group.(string)
		}
		command = fmt.Sprintf("%s --groups %s", command, strings.Join(groupsList, ","))
	}

	command = fmt.Sprintf("%s %s", command, old[0])
	_, _, err = runCommand(client, true, command, "")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}

	return userResourceRead(d, m)
}

func userResourceDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*Client)
	uid, err := strconv.Atoi(d.Id())
	if err != nil {
		return errors.Wrap(err, "ID stored is not int")
	}
	details, err := getUserFromID(client, uid)
	if err != nil {
		return errors.Wrap(err, "Failed to get user name")
	}

	command := fmt.Sprintf("/usr/sbin/userdel %s", details[0])
	_, _, err = runCommand(client, true, command, "")
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Command failed: %s", command))
	}
	return nil
}
