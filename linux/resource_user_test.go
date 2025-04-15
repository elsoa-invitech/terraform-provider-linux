package linux

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccUserCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: userConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					testAccCheckUID("testuser", func(uid int) error { return nil }),
				),
			},
		},
	})
}

func TestAccSystemUserCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: systemUserConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					resource.TestCheckResourceAttr("linux_user.testuser", "system", "true"),
					testAccCheckUID("testuser", func(uid int) error {
						if uid > 1000 {
							return fmt.Errorf("System user uid should be less than 1000")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserWithUIDCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: userWithUIDConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1024"),
					testAccCheckUID("testuser", func(uid int) error {
						if uid != 1024 {
							return fmt.Errorf("UID should be 1024")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserWithGroupsCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: userWithShellConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "home", "/home/testuser2"),
					//					resource.TestCheckResourceAttrSet("linux_user.testuser", "groups"),
					resource.TestCheckResourceAttr("linux_user.testuser", "shell", "/bin/false"),
					testAccCheckUID("testuser", func(uid int) error {
						if uid != 1024 {
							return fmt.Errorf("UID should be 1024")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserWithGroupCreation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: userWithGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1048"),
					testAccCheckUID("testuser", func(uid int) error {
						if uid != 1024 {
							return fmt.Errorf("UID should be 1024")
						}
						return nil
					}),
					testAccCheckGID("testgroup", func(gid int) error {
						if gid != 1048 {
							return fmt.Errorf("GID should be 1048")
						}
						return nil
					}),
					testAccCheckGIDForUser("testuser", func(gid int) error {
						if gid != 1048 {
							return fmt.Errorf("GID should be 1048")
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccUserUpdation(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		CheckDestroy: func(*terraform.State) error {
			client := testAccProvider.Meta().(*Client)
			return deleteGroup(client, "testuser") // changing testuser's name leaves this group dangling
		},
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: userWithUIDConfig,
			},
			resource.TestStep{
				Config: userWithGroupConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1048"),
					testAccCheckUID("testuser", func(uid int) error {
						if uid != 1024 {
							return fmt.Errorf("UID should be 1024")
						}
						return nil
					}),
					testAccCheckGID("testgroup", func(gid int) error {
						if gid != 1048 {
							return fmt.Errorf("GID should be 1048")
						}
						return nil
					}),
					testAccCheckGIDForUser("testuser", func(gid int) error {
						if gid != 1048 {
							return fmt.Errorf("GID should be 1048")
						}
						return nil
					}),
				),
			},
			resource.TestStep{
				Config: userWithGroupNameUpdatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser_alt"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1024"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1048"),
				),
			},
			resource.TestStep{
				Config: userWithGroupNameUIDUpdatedConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("linux_user.testuser", "name", "testuser_alt"),
					resource.TestCheckResourceAttr("linux_user.testuser", "uid", "1025"),
					resource.TestCheckResourceAttr("linux_user.testuser", "gid", "1048"),
				),
			},
		},
	})
}

func testAccCheckUID(username string, check func(int) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		uid, err := getUserId(client, username)
		if err != nil {
			return err
		}
		return check(uid)
	}
}

func testAccCheckGIDForUser(username string, check func(int) error) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*Client)
		details, err := getUserFromName(client, username)
		if err != nil {
			return err
		}
		gid, err := getGroupIdForUser(client, details)
		if err != nil {
			return err
		}
		return check(gid)
	}
}

const userConfig = `
resource "linux_user" "testuser" {
	name = "testuser"
}
`
const systemUserConfig = `
resource "linux_user" "testuser" {
	name = "testuser"
	system = true
}
`
const userWithUIDConfig = `
resource "linux_user" "testuser" {
	name = "testuser"
	uid = 1024
}
`
const userWithShellConfig = `
resource "linux_user" "testuser" {
	name = "testuser"
	uid = 1024
	home = "/home/testuser2"
	shell = "/bin/false"
	groups = ["backup"]
}
`

const userWithGroupConfig = `
resource "linux_group" "testgroup" {
	name = "testgroup"
	gid = 1048
}
resource "linux_user" "testuser" {
	name = "testuser"
	uid = 1024
	gid = linux_group.testgroup.gid
}
`
const userWithGroupNameUpdatedConfig = `
resource "linux_group" "testgroup" {
	name = "testgroup"
	gid = 1048
}
resource "linux_user" "testuser" {
	name = "testuser_alt"
	uid = 1024
	gid = linux_group.testgroup.gid
}
`
const userWithGroupNameUIDUpdatedConfig = `
resource "linux_group" "testgroup" {
	name = "testgroup"
	gid = 1048
}
resource "linux_user" "testuser" {
	name = "testuser_alt"
	uid = 1025
	gid = linux_group.testgroup.gid
}
`
