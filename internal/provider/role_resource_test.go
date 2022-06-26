package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRoleResourceConfig("one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptycscspm_role.test", "account_id", "123456789012"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "upt_account_id", "012345678912"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "integration_name", "uptcloud"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "external_id", "6a9375c1-47c0-470c-9217-d2f9d2d185f1"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "role", "arn:aws:iam::123456789012:role/uptcloud"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "uptycscspm_role.test",
				ImportState:       true,
				ImportStateVerify: true,
				// This is not normally necessary, but is here because this
				// example code does not have an actual upstream service.
				// Once the Read method is able to refresh information from
				// the upstream service, this can be removed.
				ImportStateVerifyIgnore: []string{"configurable_attribute"},
			},
			// Update and Read testing
			{
				Config: testAccRoleResourceConfig("two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptycscspm_role.test", "account_id", "123456789013"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRoleResourceConfig(configurableAttribute string) string {
	return fmt.Sprintf(`
resource "uptycscspm_role" "test" {
  configurable_attribute = %[1]q
}
`, configurableAttribute)
}
