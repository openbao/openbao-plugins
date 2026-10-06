package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/openbao/openbao/api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startBao(t *testing.T) *api.Client {
	pluginDir, err := filepath.Abs("./../../../bin")
	require.NoError(t, err)

	baoCtx, stopBao := context.WithCancel(context.Background())

	// start a bao process
	host := "localhost:8300"
	rootToken := "root"
	cmd := exec.CommandContext(baoCtx, "bao", "server", "-dev", "-dev-listen-address", host, "-dev-root-token-id", rootToken, "-dev-plugin-dir", pluginDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Start())

	t.Cleanup(func() {
		stopBao()
		cmd.Wait()
	})

	t.Log(pluginDir)

	// create API client
	conf := api.NewConfig()
	conf.Address = fmt.Sprintf("http://%s", host)
	client, err := api.NewClient(conf)
	require.NoError(t, err)
	client.SetToken(rootToken)

	// wait for healthiness
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		health, err := client.Sys().Health()
		require.NoError(collect, err)
		require.True(collect, health.Initialized)
	}, 20*time.Second, 100*time.Millisecond, "API should become available")

	// on cleanup, seal bao so plugin processes are properly stopped (on windows there is no sigterm, this is the most portable way if could find)
	t.Cleanup(func() {
		require.NoError(t, client.Sys().Seal())

		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			health, err := client.Sys().Health()
			require.NoError(collect, err)
			require.True(collect, health.Sealed)
		}, 20*time.Second, 100*time.Millisecond, "API should become sealed")
	})

	// find our plugin name (it contains the platform and architecture)
	pluginList, err := client.Sys().ListPlugins(&api.ListPluginsInput{Type: api.PluginTypeCredential})
	require.NoError(t, err)

	pluginIndex := slices.IndexFunc(pluginList.Details, func(plugin api.PluginDetails) bool {
		return strings.HasPrefix(plugin.Name, "openbao-plugin-auth-fido2")
	})
	require.NotEqual(t, -1, pluginIndex)
	pluginName := pluginList.Details[pluginIndex].Name

	// mount plugin
	require.NoError(t, client.Sys().EnableAuthWithOptions("passkey", &api.MountInput{
		Type: pluginName,
	}))

	// configure plugin
	_, err = client.Logical().Write("auth/passkey/config", map[string]any{
		"relying_party_id":      "localhost",
		"relying_party_origins": conf.Address,
	})
	require.NoError(t, err)

	// add additional auth mount (to test self-enroll)
	require.NoError(t, client.Sys().EnableAuthWithOptions("userpass", &api.MountInput{
		Type: "userpass",
	}))

	// create a users (for self-enroll)
	_, err = client.Logical().Write("auth/userpass/users/alice", map[string]any{
		"password":       "bob4ever",
		"token_policies": "passkey-self-enroll",
	})
	require.NoError(t, err)

	// create self-enroll policy
	require.NoError(t, client.Sys().PutPolicy("passkey-self-enroll", `path "auth/passkey/self-service-enroll" {
		capabilities = ["update"]
	}`))

	return client
}

func TestFirefox(t *testing.T) {
	testBrowser(t, "firefox")
}

func TestChromium(t *testing.T) {
	testBrowser(t, "chromium")
}

func testBrowser(t *testing.T, browserType string) {
	client := startBao(t)
	page := startBrowser(t, browserType)
	assertions := playwright.NewPlaywrightAssertions()

	// login with userpass
	_, err := page.Goto(client.Address() + "/ui/vault/auth?with=userpass")
	require.NoError(t, err)
	step(t, page, page.Locator("#username").Fill("alice"))
	step(t, page, page.Locator("#password").Fill("bob4ever"))
	step(t, page, page.GetByText("Sign In", playwright.PageGetByTextOptions{Exact: new(true)}).Click())

	step(t, page, page.GetByText("Secrets Engines").First().WaitFor())

	// self-enroll
	_, err = page.Goto(client.Address() + "/v1/auth/passkey/internal/ui/register.html")
	require.NoError(t, err)

	step(t, page, page.Locator("div", playwright.PageLocatorOptions{Has: page.GetByText("Self-Enrollment")}).GetByText("Enroll Now").Click())

	step(t, page, assertions.Locator(page.GetByText("Success!")).ToBeVisible())

	token, err := page.Locator("#client-token").InputValue()
	step(t, page, err)

	t.Log("token", token)
}

func startBrowser(t *testing.T, browserType string) playwright.Page {
	require.NoError(t, playwright.Install(&playwright.RunOptions{
		Browsers: []string{browserType},
	}))

	pw, err := playwright.Run()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pw.Stop())
	})

	var browserConstructor playwright.BrowserType
	switch browserType {
	case "firefox":
		browserConstructor = pw.Firefox
	case "chromium":
		browserConstructor = pw.Chromium
	case "webkit":
		browserConstructor = pw.WebKit
	}

	browser, err := browserConstructor.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: new(false),
	})
	require.NoError(t, err)

	browserCtx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		IgnoreHttpsErrors: new(true),
	})
	require.NoError(t, err)
	browserCtx.SetDefaultTimeout(10_000)

	require.NoError(t, browserCtx.Credentials().Install())

	page, err := browserCtx.NewPage()
	require.NoError(t, err)

	return page
}

func step(t *testing.T, page playwright.Page, err error) {
	if err != nil {
		t.Logf("Got error: %v", err)
		require.NoError(t, page.Pause())
	}
	t.Helper()
	require.NoError(t, err)
}
