---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "uptycscspm_role Resource - terraform-provider-uptycscspm"
subcategory: ""
description: |-
  Role Group resource
---

# uptycscspm_role (Resource)

Role Group resource



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `account_id` (String) AWS account ID
- `bucket_name` (String) Cloudtrail Bucket
- `bucket_region` (String) Cloudtrail Bucket Region
- `external_id` (String) External ID
- `integration_name` (String) Integration name
- `policy_document` (String) Uptycs ReadOnly Policy
- `profile_name` (String) Profile name
- `upt_account_id` (String) Uptycs AWS account ID

### Optional

- `org_access_role_name` (String) Organization Account Access Role Name

### Read-Only

- `role` (String) Role ARN


