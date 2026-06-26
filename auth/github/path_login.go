package github

import (
	"context"
	"fmt"
	"net/url"

	"github.com/google/go-github/github"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/helper/cidrutil"
	"github.com/openbao/openbao/sdk/v2/helper/policyutil"
	"github.com/openbao/openbao/sdk/v2/logical"
)

const (
	// GitHub API pagination constants
	defaultPerPage = 100
)

// AuthenticationError represents errors during GitHub authentication
type AuthenticationError struct {
	Reason  string
	Details string
}

func (e *AuthenticationError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s", e.Reason, e.Details)
	}
	return e.Reason
}

// newAuthError creates a new authentication error
func newAuthError(reason, details string) *AuthenticationError {
	return &AuthenticationError{
		Reason:  reason,
		Details: details,
	}
}

func pathLogin(b *backend) *framework.Path {
	return &framework.Path{
		Pattern: "login",

		DisplayAttrs: &framework.DisplayAttributes{
			OperationPrefix: operationPrefixGithub,
			OperationVerb:   "login",
		},

		Fields: map[string]*framework.FieldSchema{
			"token": {
				Type:        framework.TypeString,
				Description: "GitHub personal API token",
			},
		},

		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation:         b.pathLogin,
			logical.AliasLookaheadOperation: b.pathLoginAliasLookahead,
		},
	}
}

func (b *backend) pathLoginAliasLookahead(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	token := data.Get("token").(string)

	verifyResp, err := b.verifyCredentials(ctx, req, token)
	if err != nil {
		return nil, err
	}

	return &logical.Response{
		Warnings: verifyResp.Warnings,
		Auth: &logical.Auth{
			Alias: &logical.Alias{
				Name: *verifyResp.User.Login,
			},
		},
	}, nil
}

func (b *backend) pathLogin(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	token := data.Get("token").(string)

	verifyResp, err := b.verifyCredentials(ctx, req, token)
	if err != nil {
		return nil, err
	}

	auth := &logical.Auth{
		InternalData: map[string]interface{}{
			"token": token,
		},
		Metadata: map[string]string{
			"username": *verifyResp.User.Login,
			"org":      *verifyResp.Org.Login,
		},
		DisplayName: *verifyResp.User.Login,
		Alias: &logical.Alias{
			Name: *verifyResp.User.Login,
		},
	}
	if err := verifyResp.Config.PopulateTokenAuth(auth, req); err != nil {
		return nil, fmt.Errorf("failed to populate token auth: %w", err)
	}

	// Add in configured policies from user/group mapping
	if len(verifyResp.Policies) > 0 {
		auth.Policies = append(auth.Policies, verifyResp.Policies...)
	}

	resp := &logical.Response{
		Warnings: verifyResp.Warnings,
		Auth:     auth,
	}

	for _, teamName := range verifyResp.TeamNames {
		if teamName == "" {
			continue
		}
		resp.Auth.GroupAliases = append(resp.Auth.GroupAliases, &logical.Alias{
			Name: teamName,
		})
	}

	return resp, nil
}

func (b *backend) pathLoginRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	if req.Auth == nil {
		return nil, fmt.Errorf("request auth was nil")
	}

	tokenRaw, ok := req.Auth.InternalData["token"]
	if !ok {
		return nil, fmt.Errorf("token created in previous version of Vault cannot be validated properly at renewal time")
	}
	token := tokenRaw.(string)

	verifyResp, err := b.verifyCredentials(ctx, req, token)
	if err != nil {
		return nil, err
	}

	if !policyutil.EquivalentPolicies(verifyResp.Policies, req.Auth.TokenPolicies) {
		return nil, fmt.Errorf("policies do not match")
	}

	resp := &logical.Response{Auth: req.Auth}
	resp.Auth.Period = verifyResp.Config.TokenPeriod
	resp.Auth.TTL = verifyResp.Config.TokenTTL
	resp.Auth.MaxTTL = verifyResp.Config.TokenMaxTTL
	resp.Warnings = verifyResp.Warnings

	// Remove old aliases
	resp.Auth.GroupAliases = nil

	for _, teamName := range verifyResp.TeamNames {
		resp.Auth.GroupAliases = append(resp.Auth.GroupAliases, &logical.Alias{
			Name: teamName,
		})
	}

	return resp, nil
}

// verifyCredentials authenticates and authorizes a GitHub user token.
// It performs the complete authentication flow:
// 1. Loads and validates configuration
// 2. Validates request source (CIDR check)
// 3. Authenticates with GitHub
// 4. Verifies organization membership
// 5. Resolves team memberships and policies
func (b *backend) verifyCredentials(ctx context.Context, req *logical.Request, token string) (*verifyCredentialsResp, error) {
	// Load and validate configuration
	config, err := b.loadAndValidateConfig(ctx, req)
	if err != nil {
		return nil, err
	}

	// Create authenticated GitHub client
	client, err := b.createConfiguredClient(ctx, req.Storage, token, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Authenticate and authorize the user
	user, org, warnings, err := b.authenticateAndAuthorizeUser(ctx, req, client, config)
	if err != nil {
		return nil, err
	}

	// Resolve user's team memberships and policies
	teamNames, policies, err := b.resolveUserPolicies(ctx, req.Storage, client, org, user)
	if err != nil {
		return nil, err
	}

	return &verifyCredentialsResp{
		User:      user,
		Org:       org,
		Policies:  policies,
		TeamNames: teamNames,
		Config:    config,
		Warnings:  warnings,
	}, nil
}

// loadAndValidateConfig loads the backend configuration and validates the request source
func (b *backend) loadAndValidateConfig(ctx context.Context, req *logical.Request) (*config, error) {
	config, err := b.Config(ctx, req.Storage)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if config == nil {
		return nil, newAuthError("configuration not set", "GitHub auth backend has not been configured")
	}

	// Check for CIDR restrictions
	if err := b.checkCIDRMatch(req, config); err != nil {
		return nil, err
	}

	return config, nil
}

// authenticateAndAuthorizeUser performs GitHub user authentication and organization authorization
func (b *backend) authenticateAndAuthorizeUser(ctx context.Context, req *logical.Request, client *github.Client, config *config) (*github.User, *github.Organization, []string, error) {
	// Get the authenticated user from GitHub
	user, err := b.getGitHubUser(ctx, client)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}

	// Verify the user is a member of the required organization
	org, warnings, err := b.checkOrganizationMembership(ctx, client, user, config)
	if err != nil {
		return nil, nil, nil, err
	}

	return user, org, warnings, nil
}

// resolveUserPolicies resolves the user's team memberships and associated policies
func (b *backend) resolveUserPolicies(ctx context.Context, storage logical.Storage, client *github.Client, org *github.Organization, user *github.User) ([]string, []string, error) {
	// Get all teams the user belongs to in the organization
	teamNames, err := b.getUserTeams(ctx, client, org, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user teams: %w", err)
	}

	// Get policies mapped to the user's teams and username
	policies, err := b.getPoliciesForUser(ctx, storage, teamNames, user.GetLogin())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get policies: %w", err)
	}

	return teamNames, policies, nil
}

// checkCIDRMatch verifies the request comes from an allowed CIDR
func (b *backend) checkCIDRMatch(req *logical.Request, config *config) error {
	if len(config.TokenBoundCIDRs) > 0 {
		if req.Connection == nil {
			return logical.ErrPermissionDenied
		}
		if !cidrutil.RemoteAddrIsOk(req.Connection.RemoteAddr, config.TokenBoundCIDRs) {
			return logical.ErrPermissionDenied
		}
	}
	return nil
}

// createConfiguredClient creates a GitHub client with proper configuration
func (b *backend) createConfiguredClient(ctx context.Context, storage logical.Storage, token string, config *config) (*github.Client, error) {
	client, err := b.Client(token)
	if err != nil {
		return nil, err
	}

	if config.BaseURL != "" {
		parsedURL, err := url.Parse(config.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse configured base_url: %w", err)
		}
		client.BaseURL = parsedURL
	}

	// Handle organization ID auto-setup if needed
	if config.OrganizationID == 0 {
		if err := b.setupOrganizationID(ctx, storage, client, config); err != nil {
			return nil, err
		}
	}

	return client, nil
}

// setupOrganizationID sets up the organization ID if it's missing from config
func (b *backend) setupOrganizationID(ctx context.Context, storage logical.Storage, client *github.Client, config *config) error {
	err := config.setOrganizationID(ctx, client)
	if err != nil {
		return fmt.Errorf("failed to set the organization_id on login for organization '%s': %w", config.Organization, err)
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return fmt.Errorf("failed to create storage entry: %w", err)
	}

	if err := storage.Put(ctx, entry); err != nil {
		return fmt.Errorf("failed to store updated config: %w", err)
	}

	return nil
}

// getGitHubUser retrieves the current user from GitHub API
func (b *backend) getGitHubUser(ctx context.Context, client *github.Client) (*github.User, error) {
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		return nil, newAuthError("failed to get user from GitHub", err.Error())
	}
	if user.Login == nil {
		return nil, newAuthError("invalid user response", "user login is nil")
	}
	return user, nil
}

// checkOrganizationMembership verifies the user is a member of the required organization
func (b *backend) checkOrganizationMembership(ctx context.Context, client *github.Client, user *github.User, config *config) (*github.Organization, []string, error) {
	var warnings []string

	// First, get the organization details
	org, _, err := client.Organizations.Get(ctx, config.Organization)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get organization %q: %w", config.Organization, err)
	}

	// Verify the organization ID matches our config
	if org.GetID() != config.OrganizationID {
		return nil, nil, newAuthError("organization ID mismatch",
			fmt.Sprintf("organization '%s' has ID %d, but config expects ID %d",
				config.Organization, org.GetID(), config.OrganizationID))
	}

	// Check membership using the more efficient GetOrgMembership API
	membership, _, err := client.Organizations.GetOrgMembership(ctx, user.GetLogin(), config.Organization)
	if err != nil {
		// Handle different error cases
		if githubErr, ok := err.(*github.ErrorResponse); ok {
			switch githubErr.Response.StatusCode {
			case 404:
				// User is not a member or membership is private
				return nil, nil, newAuthError("user is not part of required org",
					fmt.Sprintf("user '%s' is not a member of organization '%s' or membership is private",
						user.GetLogin(), config.Organization))
			case 403:
				// Requester lacks permission to view membership
				return nil, nil, newAuthError("insufficient permissions",
					fmt.Sprintf("insufficient permissions to check membership for user '%s' in organization '%s'",
						user.GetLogin(), config.Organization))
			default:
				return nil, nil, fmt.Errorf("failed to check organization membership: %w", err)
			}
		}
		return nil, nil, fmt.Errorf("failed to check organization membership: %w", err)
	}

	// Verify the membership is active
	membershipState := membership.GetState()
	if membershipState != "active" {
		return nil, nil, newAuthError("user membership not active",
			fmt.Sprintf("user '%s' membership in organization '%s' is not active (state: %s)",
				user.GetLogin(), config.Organization, membershipState))
	}

	return org, warnings, nil
}

// getUserTeams gets all teams for the user in the specified organization
func (b *backend) getUserTeams(ctx context.Context, client *github.Client, org *github.Organization, user *github.User) ([]string, error) {
	teams, err := b.fetchUserTeamsForOrg(ctx, client, org)
	if err != nil {
		return nil, err
	}
	return b.extractTeamNames(teams), nil
}

// fetchUserTeamsForOrg retrieves all teams for a user in a specific organization
// using pagination to handle large team lists efficiently
func (b *backend) fetchUserTeamsForOrg(ctx context.Context, client *github.Client, org *github.Organization) ([]*github.Team, error) {
	var allTeams []*github.Team

	teamOpt := &github.ListOptions{
		PerPage: defaultPerPage,
	}

	// More efficient approach: Get user's teams directly for the specific organization
	// This avoids listing ALL user teams across ALL organizations and then filtering
	for {
		teams, resp, err := client.Teams.ListUserTeams(ctx, teamOpt)
		if err != nil {
			return nil, fmt.Errorf("failed to list user teams: %w", err)
		}

		// Only include teams from the specified organization
		allTeams = append(allTeams, b.filterTeamsByOrg(teams, org)...)

		if resp.NextPage == 0 {
			break
		}
		teamOpt.Page = resp.NextPage
	}

	return allTeams, nil
}

// filterTeamsByOrg filters teams to only include those from the specified organization
func (b *backend) filterTeamsByOrg(teams []*github.Team, org *github.Organization) []*github.Team {
	var filtered []*github.Team
	for _, t := range teams {
		if t.Organization != nil && t.Organization.ID != nil && *t.Organization.ID == *org.ID {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// extractTeamNames extracts both name and slug from teams for policy mapping.
// GitHub teams can have different names and slugs (URL-friendly names).
// We include both to support policy mappings using either identifier.
func (b *backend) extractTeamNames(teams []*github.Team) []string {
	var teamNames []string

	for _, t := range teams {
		// Always include the team name if available
		if t.Name != nil {
			teamNames = append(teamNames, *t.Name)
		}

		// Include the slug only if it differs from the name
		// This allows policies to be mapped to either the display name or URL slug
		if t.Slug != nil && t.Name != nil && *t.Name != *t.Slug {
			teamNames = append(teamNames, *t.Slug)
		}
	}

	return teamNames
}

// getPoliciesForUser retrieves policies for teams and user
func (b *backend) getPoliciesForUser(ctx context.Context, storage logical.Storage, teamNames []string, username string) ([]string, error) {
	groupPoliciesList, err := b.TeamMap.Policies(ctx, storage, teamNames...)
	if err != nil {
		return nil, fmt.Errorf("failed to get team policies: %w", err)
	}

	userPoliciesList, err := b.UserMap.Policies(ctx, storage, []string{username}...)
	if err != nil {
		return nil, fmt.Errorf("failed to get user policies: %w", err)
	}

	return append(groupPoliciesList, userPoliciesList...), nil
}

type verifyCredentialsResp struct {
	User      *github.User
	Org       *github.Organization
	Policies  []string
	TeamNames []string

	// Warnings to send back to the caller
	Warnings []string

	// This is just a cache to send back to the caller
	Config *config
}
