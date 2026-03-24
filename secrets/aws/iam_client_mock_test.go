package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/hashicorp/go-secure-stdlib/awsutil/v2"
)

type iamClientMock struct {
	awsutil.IAMClient
}

var _ IAMAPI = &iamClientMock{}

func (i *iamClientMock) AddUserToGroup(ctx context.Context, params *iam.AddUserToGroupInput, optFns ...func(*iam.Options)) (*iam.AddUserToGroupOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) AttachUserPolicy(ctx context.Context, params *iam.AttachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.AttachUserPolicyOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) CreateUser(ctx context.Context, params *iam.CreateUserInput, optFns ...func(*iam.Options)) (*iam.CreateUserOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) DeleteUser(ctx context.Context, params *iam.DeleteUserInput, optFns ...func(*iam.Options)) (*iam.DeleteUserOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) DeleteUserPolicy(ctx context.Context, params *iam.DeleteUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DeleteUserPolicyOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) DetachUserPolicy(ctx context.Context, params *iam.DetachUserPolicyInput, optFns ...func(*iam.Options)) (*iam.DetachUserPolicyOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) GetGroupPolicy(ctx context.Context, params *iam.GetGroupPolicyInput, optFns ...func(*iam.Options)) (*iam.GetGroupPolicyOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) ListAttachedGroupPolicies(ctx context.Context, params *iam.ListAttachedGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedGroupPoliciesOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) ListAttachedUserPolicies(ctx context.Context, params *iam.ListAttachedUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListAttachedUserPoliciesOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) ListGroupPolicies(ctx context.Context, params *iam.ListGroupPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListGroupPoliciesOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) ListGroupsForUser(ctx context.Context, params *iam.ListGroupsForUserInput, optFns ...func(*iam.Options)) (*iam.ListGroupsForUserOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) ListUserPolicies(ctx context.Context, params *iam.ListUserPoliciesInput, optFns ...func(*iam.Options)) (*iam.ListUserPoliciesOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) PutUserPolicy(ctx context.Context, params *iam.PutUserPolicyInput, optFns ...func(*iam.Options)) (*iam.PutUserPolicyOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) RemoveUserFromGroup(ctx context.Context, params *iam.RemoveUserFromGroupInput, optFns ...func(*iam.Options)) (*iam.RemoveUserFromGroupOutput, error) {
	panic("unimplemented")
}

func (i *iamClientMock) TagUser(ctx context.Context, params *iam.TagUserInput, optFns ...func(*iam.Options)) (*iam.TagUserOutput, error) {
	panic("unimplemented")
}
