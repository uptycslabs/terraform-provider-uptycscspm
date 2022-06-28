package aws

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"time"
)

const ReadOnlyPolicyName = "UptycsReadOnlyPolicy"
const ReadOnlyPolicy = `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
              "apigateway:GET",
              "codecommit:GetCommit",
              "codecommit:GetRepository",
              "codecommit:GetBranch",
              "codepipeline:ListTagsForResource",
              "codepipeline:GetPipeline",
              "ds:ListTagsForResource",
              "eks:ListNodegroups",
              "eks:DescribeFargateProfile",
              "eks:ListTagsForResource",
              "eks:ListAddons",
              "eks:DescribeAddon",
              "eks:ListFargateProfiles",
              "eks:DescribeNodegroup",
              "eks:DescribeIdentityProviderConfig",
              "eks:ListUpdates",
              "eks:DescribeUpdate",
              "eks:DescribeCluster",
              "eks:ListClusters",
              "eks:ListIdentityProviderConfigs",
              "elasticache:ListTagsForResource",
              "es:ListTags",
              "glacier:GetDataRetrievalPolicy",
              "glacier:ListJobs",
              "glacier:GetVaultAccessPolicy",
              "glacier:ListTagsForVault",
              "glacier:DescribeVault",
              "glacier:GetJobOutput",
              "glacier:GetVaultLock",
              "glacier:ListVaults",
              "glacier:GetVaultNotifications",
              "glacier:DescribeJob",
              "kinesis:DescribeStream",
              "logs:FilterLogEvents",
              "ram:ListResources",
              "ram:GetResourceShares",
              "secretsmanager:DescribeSecret",
              "servicecatalog:SearchProductsAsAdmin",
              "servicecatalog:DescribeProductAsAdmin",
              "servicecatalog:DescribePortfolio",
              "servicecatalog:DescribeServiceAction",
              "servicecatalog:DescribeProvisioningArtifact",
              "sns:ListTagsForResource",
              "sns:ListSubscriptionsByTopic",
              "sns:GetTopicAttributes",
              "sns:ListTopics",
              "sns:GetSubscriptionAttributes",
              "sqs:ListQueues",
              "sqs:GetQueueAttributes",
              "sqs:ListQueueTags",
              "ssm:ListCommandInvocations"
            ],
            "Resource": "*"
        }
    ]
}`

const ViewOnlyAccessArn = "arn:aws:iam::aws:policy/job-function/ViewOnlyAccess"
const SecurityAuditArn = "arn:aws:iam::aws:policy/SecurityAudit"

func getUptycsPolicyDoc(uptAccountId string, externalID string) string {
	return `{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:iam::` + uptAccountId + `:root"
            },
            "Action": "sts:AssumeRole",
            "Condition": {
                "StringEquals": {
                    "sts:ExternalId": "` + externalID + `"
                }
            }
        }
    ]
}`
}

func getAwsConfig(ctx context.Context, profileName string, regionCode string, roleArn string) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(regionCode),
		config.WithSharedConfigProfile(profileName),
	)
	if err != nil {
		return nil, err
	}
	// Create the credentials from AssumeRoleProvider to assume the role
	// referenced by the role ARN.
	stsSvc := sts.NewFromConfig(cfg)
	creds := stscreds.NewAssumeRoleProvider(stsSvc, roleArn, func(options *stscreds.AssumeRoleOptions) {
		options.Duration = time.Duration(60) * time.Minute
		//options.ExternalID = &externalID
	})
	cfg.Credentials = aws.NewCredentialsCache(creds)

	return &cfg, nil
}

func createIntegrationRole(ctx context.Context, svc *iam.Client, integrationName *string, uptAccountId string, externalID string) (string, error) {
	desc := "Uptycs integration role"
	assumeRolePolicyDoc := getUptycsPolicyDoc(uptAccountId, externalID)
	input := iam.CreateRoleInput{
		AssumeRolePolicyDocument: &assumeRolePolicyDoc,
		RoleName:                 integrationName,
		Description:              &desc,
	}
	roleOut, errRole := svc.CreateRole(ctx, &input)
	if errRole != nil {
		return "", errRole
	}
	if roleOut == nil || roleOut.Role == nil || roleOut.Role.Arn == nil {
		return "", fmt.Errorf("invalid CreateRoleOutput for %s", *integrationName)
	}
	return *roleOut.Role.Arn, nil
}

func createReadOnlyInlinePolicy(ctx context.Context, svc *iam.Client, roleName string) (string, error) {
	name := ReadOnlyPolicyName
	doc := ReadOnlyPolicy
	input := iam.PutRolePolicyInput{
		RoleName:       &roleName,
		PolicyName:     &name,
		PolicyDocument: &doc,
	}
	_, errPol := svc.PutRolePolicy(ctx, &input)
	if errPol != nil {
		return "", errPol
	}
	return name, nil
}

func attachPolicyToRole(ctx context.Context, svc *iam.Client, policyArn string, roleName string) error {
	input := iam.AttachRolePolicyInput{
		PolicyArn: &policyArn,
		RoleName:  &roleName,
	}
	_, errAttach := svc.AttachRolePolicy(ctx, &input)
	if errAttach != nil {
		return errAttach
	}
	return nil
}

func deleteIntegrationRole(ctx context.Context, svc *iam.Client, integrationName string) error {
	input := iam.DeleteRoleInput{
		RoleName: &integrationName,
	}
	_, errRole := svc.DeleteRole(ctx, &input)
	if errRole != nil {
		return errRole
	}
	return nil
}

func deleteReadOnlyInlinePolicy(ctx context.Context, svc *iam.Client, integrationName string) error {
	name := ReadOnlyPolicyName
	input := iam.DeleteRolePolicyInput{
		RoleName:   &integrationName,
		PolicyName: &name,
	}
	_, errRole := svc.DeleteRolePolicy(ctx, &input)
	if errRole != nil {
		return errRole
	}
	return nil
}

func detachPolicyToRole(ctx context.Context, svc *iam.Client, policyArn string, roleName string) error {
	input := iam.DetachRolePolicyInput{
		RoleName:  &roleName,
		PolicyArn: &policyArn,
	}
	_, errRole := svc.DetachRolePolicy(ctx, &input)
	if errRole != nil {
		return errRole
	}
	return nil
}

func GetAwsIamClient(ctx context.Context, profileName string, regionCode string, childAccountID string) (*iam.Client, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", childAccountID)
	sess, err := getAwsConfig(ctx, profileName, regionCode, roleArn)
	if err != nil {
		return nil, err
	}
	svc := iam.NewFromConfig(*sess)
	if svc == nil {
		return nil, fmt.Errorf("failed to create client with profile=%s, region=%s, role=%s", profileName, regionCode, roleArn)
	}
	return svc, nil
}

func GetIntegrationRoleName(ctx context.Context, svc *iam.Client, integrationName string) (string, error) {
	input := iam.GetRoleInput{
		RoleName: &integrationName,
	}
	roleOut, errGet := svc.GetRole(ctx, &input)
	if errGet != nil {
		return "", errGet
	}
	if roleOut == nil || roleOut.Role == nil || roleOut.Role.Arn == nil {
		return "", fmt.Errorf("invalid roleOutput for %s", integrationName)
	}
	return *roleOut.Role.Arn, nil
}

func CreateUptycsCspmResources(ctx context.Context, svc *iam.Client, integrationName string, uptAccountID string, externalID string) (string, error) {
	roleName, roleErr := createIntegrationRole(ctx, svc, &integrationName, uptAccountID, externalID)
	if roleErr != nil {
		return "", roleErr
	}
	_, inlinePolErr := createReadOnlyInlinePolicy(ctx, svc, integrationName)
	if inlinePolErr != nil {
		// clean-up already created resources
		_ = deleteIntegrationRole(ctx, svc, integrationName)
		return "", inlinePolErr
	}
	if attachErr := attachPolicyToRole(ctx, svc, ViewOnlyAccessArn, integrationName); attachErr != nil {
		// clean-up already created resources
		_ = deleteReadOnlyInlinePolicy(ctx, svc, integrationName)
		_ = deleteIntegrationRole(ctx, svc, integrationName)
		return "", attachErr
	}
	if attachErr := attachPolicyToRole(ctx, svc, SecurityAuditArn, integrationName); attachErr != nil {
		// clean-up already created resources
		_ = detachPolicyToRole(ctx, svc, ViewOnlyAccessArn, integrationName)
		_ = deleteReadOnlyInlinePolicy(ctx, svc, integrationName)
		_ = deleteIntegrationRole(ctx, svc, integrationName)
		return "", attachErr
	}
	return roleName, nil
}

func DeleteUptycsCspmResources(ctx context.Context, svc *iam.Client, integrationName string) error {
	if detachErr := detachPolicyToRole(ctx, svc, SecurityAuditArn, integrationName); detachErr != nil {
		return detachErr
	}
	if detachErr := detachPolicyToRole(ctx, svc, ViewOnlyAccessArn, integrationName); detachErr != nil {
		return detachErr
	}
	if readOnlyPolErr := deleteReadOnlyInlinePolicy(ctx, svc, integrationName); readOnlyPolErr != nil {
		return readOnlyPolErr
	}
	if roleErr := deleteIntegrationRole(ctx, svc, integrationName); roleErr != nil {
		return roleErr
	}
	return nil
}
