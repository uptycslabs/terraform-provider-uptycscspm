package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	awsinternal "github.com/uptycslabs/terraform-provider-uptycscspm/internal/aws"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ tfsdk.ResourceType = roleResourceType{}
var _ tfsdk.Resource = roleResource{}
var _ tfsdk.ResourceWithImportState = roleResource{}

type roleResourceType struct{}

func (t roleResourceType) GetSchema(ctx context.Context) (tfsdk.Schema, diag.Diagnostics) {
	_ = ctx
	return tfsdk.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Role Group resource",

		Attributes: map[string]tfsdk.Attribute{
			"profile_name": {
				MarkdownDescription: "Profile name",
				Required:            true,
				Type:                types.StringType,
			},
			"account_id": {
				MarkdownDescription: "AWS account ID",
				Required:            true,
				Type:                types.StringType,
			},
			"integration_name": {
				MarkdownDescription: "Integration name",
				Required:            true,
				Type:                types.StringType,
			},
			"upt_account_id": {
				MarkdownDescription: "Uptycs AWS account ID",
				Required:            true,
				Type:                types.StringType,
			},
			"external_id": {
				MarkdownDescription: "External ID",
				Required:            true,
				Type:                types.StringType,
			},
			"role": {
				MarkdownDescription: "Role ARN",
				Computed:            true,
				Type:                types.StringType,
			},
			"bucket_name": {
				MarkdownDescription: "Cloudtrail Bucket",
				Required:            true,
				Type:                types.StringType,
			},
			"bucket_region": {
				MarkdownDescription: "Cloudtrail Bucket Region",
				Required:            true,
				Type:                types.StringType,
			},
			"policy_document": {
				MarkdownDescription: "Uptycs ReadOnly Policy",
				Required:            true,
				Type:                types.StringType,
			},
			"org_access_role_name": {
				MarkdownDescription: "Organization Account Access Role Name",
				Optional:            true,
				Type:                types.StringType,
			},
		},
	}, nil
}

func (t roleResourceType) NewResource(ctx context.Context, in tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	_ = ctx
	provider, diags := convertProviderType(in)

	return roleResource{
		provider: provider,
	}, diags
}

type exampleResourceData struct {
	ProfileName       types.String `tfsdk:"profile_name"`
	AccountID         types.String `tfsdk:"account_id"`
	IntegrationName   types.String `tfsdk:"integration_name"`
	UptAccountID      types.String `tfsdk:"upt_account_id"`
	ExternalID        types.String `tfsdk:"external_id"`
	Role              types.String `tfsdk:"role"`
	BucketName        types.String `tfsdk:"bucket_name"`
	BucketRegion      types.String `tfsdk:"bucket_region"`
	PolicyDocument    types.String `tfsdk:"policy_document"`
	OrgAccessRoleName types.String `tfsdk:"org_access_role_name"`
}

type roleResource struct {
	provider provider
}

func (r roleResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	var data exampleResourceData

	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.CreateExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	// For the purposes of this example code, hardcoding a response value to
	// save into the Terraform state.
	svc, errSvc := awsinternal.GetAwsIamClient(ctx, data.ProfileName.Value, "aws-global", data.AccountID.Value, data.OrgAccessRoleName.Value)
	if errSvc != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get client for %s with profile %s. err=%s", data.AccountID.Value, data.ProfileName.Value, errSvc.Error()))
		return
	}
	role, errCreate := awsinternal.CreateUptycsCspmResources(ctx,
		svc,
		data.IntegrationName.Value,
		data.UptAccountID.Value,
		data.ExternalID.Value,
		data.BucketName.Value,
		data.BucketRegion.Value,
		data.ProfileName.Value,
		data.AccountID.Value,
		data.PolicyDocument.Value,
		data.OrgAccessRoleName.Value,
		false)
	if errCreate != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create uptycscspm role. err=%s", errCreate))
		return
	}
	data.Role = types.String{Value: role}

	// write logs using the tflog package
	// see https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log/tflog
	// for more information
	tflog.Trace(ctx, "created a resource")

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r roleResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	var data exampleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.ReadExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read example, got error: %s", err))
	//     return
	// }

	orgSvc, orgErrSvc := awsinternal.GetOrgClient(ctx, data.ProfileName.Value)
	if orgErrSvc != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get org client with profile %s. err=%s", data.ProfileName.Value, orgErrSvc.Error()))
		return
	}

	accountExists, err := awsinternal.IsAccountExistsInOrg(ctx, orgSvc, data.AccountID.Value)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get account list from organization. err=%s", orgErrSvc.Error()))
		return
	}

	if !accountExists {
		resp.State.RemoveResource(ctx)
		return
	}

	svc, errSvc := awsinternal.GetAwsIamClient(ctx, data.ProfileName.Value, "aws-global", data.AccountID.Value, data.OrgAccessRoleName.Value)
	if errSvc != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get client for %s with profile %s. err=%s", data.AccountID.Value, data.ProfileName.Value, errSvc.Error()))
		return
	}
	role, errRole := awsinternal.GetIntegrationRoleName(ctx, svc, data.IntegrationName.Value)
	if errRole != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get uptycscspm role. err=%s", errRole))
		return
	}
	data.Role = types.String{Value: role}

	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r roleResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	var data exampleResourceData

	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.UpdateExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update example, got error: %s", err))
	//     return
	// }
	svc, errSvc := awsinternal.GetAwsIamClient(ctx, data.ProfileName.Value, "aws-global", data.AccountID.Value, data.OrgAccessRoleName.Value)
	if errSvc != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get client for %s with profile %s. err=%s", data.AccountID.Value, data.ProfileName.Value, errSvc.Error()))
		return
	}
	errDel := awsinternal.DeleteUptycsCspmResources(ctx, svc, data.IntegrationName.Value)
	if errDel != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update uptycscspm role. err=%s", errDel))
		return
	}
	role, errCreate := awsinternal.CreateUptycsCspmResources(ctx,
		svc,
		data.IntegrationName.Value,
		data.UptAccountID.Value,
		data.ExternalID.Value,
		data.BucketName.Value,
		data.BucketRegion.Value,
		data.ProfileName.Value,
		data.AccountID.Value,
		data.PolicyDocument.Value,
		data.OrgAccessRoleName.Value,
		true)
	if errCreate != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to re-create uptycscspm role. err=%s", errCreate))
		return
	}
	data.Role = types.String{Value: role}
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
}

func (r roleResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	var data exampleResourceData

	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// example, err := d.provider.client.DeleteExample(...)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete example, got error: %s", err))
	//     return
	// }
	svc, errSvc := awsinternal.GetAwsIamClient(ctx, data.ProfileName.Value, "aws-global", data.AccountID.Value, data.OrgAccessRoleName.Value)
	if errSvc != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get client for %s with profile %s. err=%s", data.AccountID.Value, data.ProfileName.Value, errSvc.Error()))
		return
	}
	errDel := awsinternal.DeleteUptycsCspmResources(ctx, svc, data.IntegrationName.Value)
	if errDel != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create uptycscspm role. err=%s", errDel))
		return
	}
}

func (r roleResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}
