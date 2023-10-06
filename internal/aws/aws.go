package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	storage "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const ReadOnlyPolicyName = "UptycsReadOnlyPolicy"

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

func createReadOnlyInlinePolicy(ctx context.Context, svc *iam.Client, roleName string, policyDocument string) (string, error) {
	name := ReadOnlyPolicyName
	doc := policyDocument
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

func createBucketPolicy(ctx context.Context, svc *iam.Client, roleName string, bucketName string) (string, error) {
	name := roleName + "-CloudtrailBucketPolicy"
	policyDocument := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": "s3:GetObject",
				"Resource": "arn:aws:s3:::%s/*"
			}
		]
	}`
	doc := fmt.Sprintf(policyDocument, bucketName)
	input := iam.CreatePolicyInput{
		PolicyName:     &name,
		PolicyDocument: &doc,
	}
	policy, errPol := svc.CreatePolicy(ctx, &input)
	if errPol != nil {
		return "", errPol
	}
	return *policy.Policy.Arn, nil
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

func deleteBucketPolicy(ctx context.Context, svc *iam.Client, policyArn string) error {
	input := iam.DeletePolicyInput{
		PolicyArn: &policyArn,
	}
	_, errPol := svc.DeletePolicy(ctx, &input)
	if errPol != nil {
		return errPol
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

func GetAwsIamClient(ctx context.Context, profileName string, regionCode string, childAccountID string, roleToAssume string) (*iam.Client, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", childAccountID)
	if roleToAssume != "" {
		roleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", childAccountID, roleToAssume)
	}
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

func getAwsS3Client(ctx context.Context, profileName string, regionCode string, childAccountID string, roleToAssume string) (*storage.Client, error) {
	roleArn := fmt.Sprintf("arn:aws:iam::%s:role/OrganizationAccountAccessRole", childAccountID)
	if roleToAssume != "" {
		roleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", childAccountID, roleToAssume)
	}
	sess, err := getAwsConfig(ctx, profileName, regionCode, roleArn)
	if err != nil {
		return nil, err
	}
	svc := storage.NewFromConfig(*sess)
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

func CreateUptycsCspmResources(
	ctx context.Context,
	svc *iam.Client, integrationName string,
	uptAccountID string,
	externalID string,
	bucketName string,
	bucketRegion string,
	profileName string,
	accountId string,
	policyDocument string,
	roleToAssume string,
	isUpdate bool,
) (string, error) {
	roleArn := ""
	existRoleArn, err := GetIntegrationRoleName(ctx, svc, integrationName)
	if err != nil {
		newRoleArn, roleErr := createIntegrationRole(ctx, svc, &integrationName, uptAccountID, externalID)
		if roleErr != nil {
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", roleErr
		}
		roleArn = newRoleArn
	} else {
		roleArn = existRoleArn
	}

	params := &iam.ListAttachedRolePoliciesInput{
		RoleName: &integrationName,
	}
	attachedPoliciesMap := make(map[string]bool, 0)
	inlinePoliciesMap := make(map[string]bool, 0)
	policiesOuput, ListPolicyErr := svc.ListAttachedRolePolicies(ctx, params)
	if ListPolicyErr == nil {
		for _, attachedPolicy := range policiesOuput.AttachedPolicies {
			attachedPoliciesMap[*attachedPolicy.PolicyArn] = true
		}
	}

	inlinePolicyParams := &iam.ListRolePoliciesInput{
		RoleName: &integrationName,
	}
	inlinePoliciesOp, listInlinePoliciesErr := svc.ListRolePolicies(ctx, inlinePolicyParams)

	if listInlinePoliciesErr == nil {
		for _, inlinePolicy := range inlinePoliciesOp.PolicyNames {
			inlinePoliciesMap[inlinePolicy] = true
		}
	}

	if _, found := inlinePoliciesMap[ReadOnlyPolicyName]; !found {
		_, inlinePolErr := createReadOnlyInlinePolicy(ctx, svc, integrationName, policyDocument)
		if inlinePolErr != nil {
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", inlinePolErr
		}
	}
	if _, found := attachedPoliciesMap[ViewOnlyAccessArn]; !found {
		if attachErr := attachPolicyToRole(ctx, svc, ViewOnlyAccessArn, integrationName); attachErr != nil {
			// clean-up already created resources
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", attachErr
		}
	}

	if _, found := attachedPoliciesMap[SecurityAuditArn]; !found {
		if attachErr := attachPolicyToRole(ctx, svc, SecurityAuditArn, integrationName); attachErr != nil {
			// clean-up already created resources
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", attachErr
		}

	}

	if bucketName != "" {
		//get s3 client
		s3Client, s3ClientErr := getAwsS3Client(ctx, profileName, bucketRegion, accountId, roleToAssume)
		if s3ClientErr != nil {
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", s3ClientErr
		}

		//validate s3 bucket
		input := &storage.HeadBucketInput{
			Bucket: &bucketName,
		}
		_, s3ValidationErr := s3Client.HeadBucket(ctx, input)
		if s3ValidationErr != nil {
			if !isUpdate {
				DeleteUptycsCspmResources(ctx, svc, integrationName)
			}
			return "", s3ValidationErr
		}

		cloudtrailBucketPolicyArn := "arn:aws:iam::" + accountId + ":policy/" + integrationName + "-CloudtrailBucketPolicy"

		if _, found := attachedPoliciesMap[cloudtrailBucketPolicyArn]; !found {
			policyParams := &iam.GetPolicyInput{
				PolicyArn: &cloudtrailBucketPolicyArn,
			}
			if _, policyErr := svc.GetPolicy(ctx, policyParams); policyErr != nil {
				_, policyErr1 := createBucketPolicy(ctx, svc, integrationName, bucketName)
				if policyErr1 != nil {
					if !isUpdate {
						DeleteUptycsCspmResources(ctx, svc, integrationName)
					}
					return "", policyErr
				}
			}

			if attachErr := attachPolicyToRole(ctx, svc, cloudtrailBucketPolicyArn, integrationName); attachErr != nil {
				// clean-up already created resources
				if !isUpdate {
					DeleteUptycsCspmResources(ctx, svc, integrationName)
				}
				return "", attachErr
			}

		}

	}
	return roleArn, nil
}

func DeleteUptycsCspmResources(ctx context.Context, svc *iam.Client, integrationName string) error {
	params := &iam.ListAttachedRolePoliciesInput{
		RoleName: &integrationName,
	}
	policiesOuput, ListPolicyErr := svc.ListAttachedRolePolicies(ctx, params)
	if ListPolicyErr != nil {
		return ListPolicyErr
	}
	cloudtrailBucketPolicyName := integrationName + "-CloudtrailBucketPolicy"

	for _, policy := range policiesOuput.AttachedPolicies {
		switch *policy.PolicyName {
		case cloudtrailBucketPolicyName:
			if detachErr := detachPolicyToRole(ctx, svc, *policy.PolicyArn, integrationName); detachErr != nil {
				return detachErr
			}
			if delPolicyErr := deleteBucketPolicy(ctx, svc, *policy.PolicyArn); delPolicyErr != nil {
				return delPolicyErr
			}
		case "SecurityAudit":
			if detachErr := detachPolicyToRole(ctx, svc, SecurityAuditArn, integrationName); detachErr != nil {
				return detachErr
			}
		case "ViewOnlyAccess":
			if detachErr := detachPolicyToRole(ctx, svc, ViewOnlyAccessArn, integrationName); detachErr != nil {
				return detachErr
			}
		}
	}
	inlinePolicyParams := &iam.ListRolePoliciesInput{
		RoleName: &integrationName,
	}
	inlinePoliciesOp, listInlinePoliciesErr := svc.ListRolePolicies(ctx, inlinePolicyParams)
	if listInlinePoliciesErr != nil {
		return listInlinePoliciesErr
	}
	for _, inlinePolicy := range inlinePoliciesOp.PolicyNames {
		if inlinePolicy == ReadOnlyPolicyName {
			if readOnlyPolErr := deleteReadOnlyInlinePolicy(ctx, svc, integrationName); readOnlyPolErr != nil {
				return readOnlyPolErr
			}
		}

	}
	if roleErr := deleteIntegrationRole(ctx, svc, integrationName); roleErr != nil {
		return roleErr
	}
	return nil

}
