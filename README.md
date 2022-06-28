# Terraform Provider Uptycs - for CSPM Integration

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.17+

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

```
terraform {
  required_providers {
    uptycscspm = {
      source  = "github.com/uptycslabs/uptycscspm"
      version = "0.0.1"
    }
  }
}

resource "uptycscspm_role" "test" {
  profile_name = "default"
  account_id = "123456789012"
  integration_name = "UptycsIntegration"
  upt_account_id = "123456789013"
  external_id = "6a9375c1-47c0-470c-9217-d2f9d2d185f1"
}
```