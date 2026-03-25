package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	mock_aws "github.com/openbao/openbao-plugins/secrets/aws/internal/mock"
	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/openbao/openbao/sdk/v2/queue"
	"go.uber.org/mock/gomock"
)

// TestRotation verifies that the rotation code and priority queue correctly selects and rotates credentials
// for static secrets.
func TestRotation(t *testing.T) {
	bgCTX := context.Background()

	type credToInsert struct {
		config staticRoleEntry // role configuration from a normal createRole request
		age    time.Duration   // how old the cred should be - if this is longer than the config.RotationPeriod,
		// the cred is 'pre-expired'

		changed bool // whether we expect the cred to change - this is technically redundant to a comparison between
		// rotationPeriod and age.
	}

	// due to a limitation with the mockIAM implementation, any cred you want to rotate must have
	// username jane-doe and userid unique-id, since we can only pre-can one exact response to GetUser
	cases := []struct {
		name  string
		creds []credToInsert
	}{
		{
			name: "refresh one",
			creds: []credToInsert{
				{
					config: staticRoleEntry{
						Name:           "test",
						Username:       "jane-doe",
						ID:             "unique-id",
						RotationPeriod: 2 * time.Second,
					},
					age:     5 * time.Second,
					changed: true,
				},
			},
		},
		{
			name: "refresh none",
			creds: []credToInsert{
				{
					config: staticRoleEntry{
						Name:           "test",
						Username:       "jane-doe",
						ID:             "unique-id",
						RotationPeriod: 1 * time.Minute,
					},
					age:     5 * time.Second,
					changed: false,
				},
			},
		},
		{
			name: "refresh one of two",
			creds: []credToInsert{
				{
					config: staticRoleEntry{
						Name:           "toast",
						Username:       "john-doe",
						ID:             "other-id",
						RotationPeriod: 1 * time.Minute,
					},
					age:     5 * time.Second,
					changed: false,
				},
				{
					config: staticRoleEntry{
						Name:           "test",
						Username:       "jane-doe",
						ID:             "unique-id",
						RotationPeriod: 1 * time.Second,
					},
					age:     5 * time.Second,
					changed: true,
				},
			},
		},
		{
			name:  "no creds to rotate",
			creds: []credToInsert{},
		},
	}

	ak := "long-access-key-id"
	oldSecret := "abcdefghijklmnopqrstuvwxyz"
	newSecret := "zyxwvutsrqponmlkjihgfedcba"

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			config := logical.TestBackendConfig()
			config.StorageView = &logical.InmemStorage{}

			b := Backend(config)

			// insert all our creds
			for i, cred := range c.creds {

				// all the creds will be the same for every user, but that's okay
				// since what we care about is whether they changed on a single-user basis.
				mock := mock_aws.NewMockIAMAPI(gomock.NewController(t))
				mock.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&iam.GetUserOutput{
					User: &iamtypes.User{
						UserId:   aws.String(cred.config.ID),
						UserName: aws.String(cred.config.Username),
					},
				}, nil)
				mock.EXPECT().ListAccessKeys(gomock.Any(), gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []iamtypes.AccessKeyMetadata{
						{},
					},
				}, nil)
				mock.EXPECT().CreateAccessKey(gomock.Any(), gomock.Any()).Return(&iam.CreateAccessKeyOutput{
					AccessKey: &iamtypes.AccessKey{
						AccessKeyId:     aws.String(ak),
						SecretAccessKey: aws.String(oldSecret),
					},
				}, nil)
				b.iamClient = mock

				err := b.createCredential(bgCTX, config.StorageView, cred.config, true)
				if err != nil {
					t.Fatalf("couldn't insert credential %d: %s", i, err)
				}

				item := &queue.Item{
					Key:      cred.config.Name,
					Value:    cred.config,
					Priority: time.Now().Add(-1 * cred.age).Add(cred.config.RotationPeriod).Unix(),
				}
				err = b.credRotationQueue.Push(item)
				if err != nil {
					t.Fatalf("couldn't push item onto queue: %s", err)
				}
			}

			// update aws responses, same argument for why it's okay every cred will be the same
			mock := mock_aws.NewMockIAMAPI(gomock.NewController(t))
			for _, cred := range c.creds {
				if cred.changed {
					mock.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&iam.GetUserOutput{
						User: &iamtypes.User{
							UserId:   aws.String("unique-id"),
							UserName: aws.String("jane-doe"),
						},
					}, nil)
					mock.EXPECT().ListAccessKeys(gomock.Any(), gomock.Any()).Return(&iam.ListAccessKeysOutput{
						AccessKeyMetadata: []iamtypes.AccessKeyMetadata{
							{AccessKeyId: aws.String(ak)},
						},
					}, nil)
					mock.EXPECT().CreateAccessKey(gomock.Any(), gomock.Any()).Return(&iam.CreateAccessKeyOutput{
						AccessKey: &iamtypes.AccessKey{
							AccessKeyId:     aws.String(ak),
							SecretAccessKey: aws.String(newSecret),
						},
					}, nil)
				}
			}

			b.iamClient = mock

			req := &logical.Request{
				Storage: config.StorageView,
			}
			err := b.rotateExpiredStaticCreds(bgCTX, req)
			if err != nil {
				t.Fatalf("got an error rotating credentials: %s", err)
			}

			// check our credentials
			for i, cred := range c.creds {
				entry, err := config.StorageView.Get(bgCTX, formatCredsStoragePath(cred.config.Name))
				if err != nil {
					t.Fatalf("got an error retrieving credentials %d", i)
				}
				var out awsCredentials
				err = entry.DecodeJSON(&out)
				if err != nil {
					t.Fatalf("could not unmarshal storage view entry for cred %d to an aws credential: %s", i, err)
				}

				if cred.changed && out.SecretAccessKey != newSecret {
					t.Fatalf("expected the key for cred %d to have changed, but it hasn't", i)
				} else if !cred.changed && out.SecretAccessKey != oldSecret {
					t.Fatalf("expected the key for cred %d to have stayed the same, but it changed", i)
				}
			}
		})
	}
}

// TestCreateCredential verifies that credential creation firstly only deletes credentials if it needs to (i.e., two
// or more credentials on IAM), and secondly correctly deletes the oldest one.
func TestCreateCredential(t *testing.T) {
	cases := []struct {
		name      string
		username  string
		id        string
		mockCalls func(expect *mock_aws.MockIAMAPIMockRecorder)
	}{
		{
			name:     "zero keys",
			username: "jane-doe",
			id:       "unique-id",
			mockCalls: func(expect *mock_aws.MockIAMAPIMockRecorder) {
				expect.ListAccessKeys(gomock.Any(), gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []iamtypes.AccessKeyMetadata{},
				}, nil)
				expect.CreateAccessKey(gomock.Any(), gomock.Any()).Return(&iam.CreateAccessKeyOutput{
					AccessKey: &iamtypes.AccessKey{
						AccessKeyId:     aws.String("key"),
						SecretAccessKey: aws.String("itsasecret"),
					},
				}, nil)
			},
		},
		{
			name:     "one key",
			username: "jane-doe",
			id:       "unique-id",
			mockCalls: func(expect *mock_aws.MockIAMAPIMockRecorder) {
				expect.ListAccessKeys(gomock.Any(), gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []iamtypes.AccessKeyMetadata{
						{AccessKeyId: aws.String("foo"), CreateDate: aws.Time(time.Now())},
					},
				}, nil)
				expect.CreateAccessKey(gomock.Any(), gomock.Any()).Return(&iam.CreateAccessKeyOutput{
					AccessKey: &iamtypes.AccessKey{
						AccessKeyId:     aws.String("key"),
						SecretAccessKey: aws.String("itsasecret"),
					},
				}, nil)
			},
		},
		{
			name:     "two keys",
			username: "jane-doe",
			id:       "unique-id",
			mockCalls: func(expect *mock_aws.MockIAMAPIMockRecorder) {
				expect.ListAccessKeys(gomock.Any(), gomock.Any()).Return(&iam.ListAccessKeysOutput{
					AccessKeyMetadata: []iamtypes.AccessKeyMetadata{
						{AccessKeyId: aws.String("foo"), CreateDate: aws.Time(time.Time{})},
						{AccessKeyId: aws.String("bar"), CreateDate: aws.Time(time.Now())},
					},
				}, nil)
				expect.DeleteAccessKey(gomock.Any(), mock_aws.Eq(&iam.DeleteAccessKeyInput{
					AccessKeyId: aws.String("foo"),
				}))
				expect.CreateAccessKey(gomock.Any(), gomock.Any()).Return(&iam.CreateAccessKeyOutput{
					AccessKey: &iamtypes.AccessKey{
						AccessKeyId:     aws.String("key"),
						SecretAccessKey: aws.String("itsasecret"),
					},
				}, nil)
			},
		},
	}

	config := logical.TestBackendConfig()
	config.StorageView = &logical.InmemStorage{}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := Backend(config)
			mock := mock_aws.NewMockIAMAPI(gomock.NewController(t))
			mock.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(&iam.GetUserOutput{
				User: &iamtypes.User{
					UserId:   aws.String(c.id),
					UserName: aws.String(c.username),
				},
			}, nil)

			c.mockCalls(mock.EXPECT())
			b.iamClient = mock

			err := b.createCredential(context.Background(), config.StorageView, staticRoleEntry{Username: c.username, ID: c.id}, true)
			if err != nil {
				t.Fatalf("got an error we didn't expect: %q", err)
			}
		})
	}
}
