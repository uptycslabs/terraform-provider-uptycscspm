package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var errRegex, _ = regexp.Compile("Unable to create uptycscspm role. err=operation error IAM: CreateRole")

func TestAccRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccRoleResourceConfig("noprofile", "123456789012", "012345678912", "uptcloud", "6a9375c1-47c0-470c-9217-d2f9d2d185f1", "uptycs-test-bucket", "us-east-1", "", "OrganizationAccountAccessRole"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("uptycscspm_role.test", "profile_name", "noprofile"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "account_id", "123456789012"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "upt_account_id", "012345678912"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "integration_name", "uptcloud"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "external_id", "6a9375c1-47c0-470c-9217-d2f9d2d185f1"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "role", "arn:aws:iam::123456789012:role/uptcloud"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "bucket_name", "uptycs-test-bucket"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "bucket_region", "us-east-1"),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "policy_document", ""),
					resource.TestCheckResourceAttr("uptycscspm_role.test", "org_access_role_name", "OrganizationAccountAccessRole"),
				),
				// Expect to fail as we cannot contact AWS with fake accounts
				ExpectError: errRegex,
			},
			//// ImportState testing
			//{
			//	ResourceName:      "uptycscspm_role.test",
			//	ImportState:       true,
			//	ImportStateVerify: true,
			//	// This is not normally necessary, but is here because this
			//	// example code does not have an actual upstream service.
			//	// Once the Read method is able to refresh information from
			//	// the upstream service, this can be removed.
			//	ImportStateVerifyIgnore: []string{"configurable_attribute"},
			//},
			//// Update and Read testing
			//{
			//	Config: testAccRoleResourceConfig("default", "123456789012", "012345678912", "uptcloud2", "6a9375c1-47c0-470c-9217-d2f9d2d185f1"),
			//	Check: resource.ComposeAggregateTestCheckFunc(
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "profile_name", "default"),
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "account_id", "123456789012"),
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "upt_account_id", "012345678912"),
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "integration_name", "uptcloud"),
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "external_id", "6a9375c1-47c0-470c-9217-d2f9d2d185f1"),
			//		resource.TestCheckResourceAttr("uptycscspm_role.test", "role", "arn:aws:iam::123456789012:role/uptcloud2"),
			//	),
			//},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccRoleResourceConfig(profile string, account string, uptAccount string, integration string, externalID string, bucketName string, bucketRegion string, policyDocument string, orgAccessRoleName string) string {
	return fmt.Sprintf(`
resource "uptycscspm_role" "test" {
  profile_name = %[1]q
  account_id = %[2]q
  upt_account_id = %[3]q
  integration_name = %[4]q
  external_id = %[5]q
  bucket_name = %[6]q
  bucket_region = %[7]q
  policy_document = %[8]q
  org_access_role_name = %[9]q

}
`, profile, account, uptAccount, integration, externalID, bucketName, bucketRegion, policyDocument, orgAccessRoleName)
}
