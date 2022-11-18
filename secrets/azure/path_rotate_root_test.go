package azuresecrets

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/vault/sdk/logical"
)

func TestRotateRootSuccess(t *testing.T) {
	b, s := getTestBackend(t, true)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "rotate-root",
		Data:      map[string]interface{}{},
		Storage:   s,
	})

	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	config, err := b.getConfig(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if config.ClientSecret == "" {
		t.Fatal(fmt.Errorf("root password was empty after rotate root, it shouldn't be"))
	}

	if config.NewClientSecret == config.ClientSecret {
		t.Fatal("old and new password equal after rotate-root, it shouldn't be")
	}

	if config.NewClientSecret == "" {
		t.Fatal("new password is empty, it shouldn't be")
	}

	if config.NewClientSecretKeyID == "" {
		t.Fatal("new password key id is empty, it shouldn't be")
	}

	if !b.updatePassword {
		t.Fatal("update password is false, it shouldn't be")
	}

	config.NewClientSecretCreated = config.NewClientSecretCreated.Add(-(time.Minute * 1))
	err = b.saveConfig(context.Background(), config, s)
	if err != nil {
		t.Fatal(err)
	}

	err = b.periodicFunc(context.Background(), &logical.Request{
		Storage: s,
	})

	if err != nil {
		t.Fatal(err)
	}

	newConfig, err := b.getConfig(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}

	if newConfig.ClientSecret != config.NewClientSecret {
		t.Fatal(fmt.Errorf("old and new password aren't equal after periodic function, they should be"))
	}
}

func TestRotateRootPeroidicFunctionBeforeMinute(t *testing.T) {
	b, s := getTestBackend(t, true)

	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "rotate-root",
		Data:      map[string]interface{}{},
		Storage:   s,
	})

	if err != nil {
		t.Fatal(err)
	}

	if resp != nil && resp.IsError() {
		t.Fatal(resp.Error())
	}

	tests := []struct {
		Name    string
		Created time.Duration
	}{
		{
			Name:    "1 second test:",
			Created: time.Second * 1,
		},
		{
			Name:    "5 seconds test:",
			Created: time.Second * 5,
		},
		{
			Name:    "30 seconds test:",
			Created: time.Second * 30,
		},
		{
			Name:    "50 seconds test:",
			Created: time.Second * 50,
		},
	}

	for _, test := range tests {
		t.Log(test.Name)
		config, err := b.getConfig(context.Background(), s)
		if err != nil {
			t.Fatal(err)
		}

		config.NewClientSecretCreated = time.Now().Add(-(test.Created))
		err = b.saveConfig(context.Background(), config, s)
		if err != nil {
			t.Fatal(test.Name, err)
		}

		err = b.periodicFunc(context.Background(), &logical.Request{
			Storage: s,
		})

		if err != nil {
			t.Fatal(test.Name, err)
		}

		newConfig, err := b.getConfig(context.Background(), s)
		if err != nil {
			t.Fatal(test.Name, err)
		}

		if newConfig.ClientSecret == config.NewClientSecret {
			t.Fatal(test.Name, fmt.Errorf("old and new password are equal after periodic function, they shouldn't be"))
		}
	}
}

func TestIntersectStrings(t *testing.T) {
	type testCase struct {
		a      []string
		b      []string
		expect []string
	}

	tests := map[string]testCase{
		"nil slices": {
			a:      nil,
			b:      nil,
			expect: []string{},
		},
		"a is nil": {
			a:      nil,
			b:      []string{"foo"},
			expect: []string{},
		},
		"b is nil": {
			a:      []string{"foo"},
			b:      nil,
			expect: []string{},
		},
		"a is empty": {
			a:      []string{},
			b:      []string{"foo"},
			expect: []string{},
		},
		"b is empty": {
			a:      []string{"foo"},
			b:      []string{},
			expect: []string{},
		},
		"a equals b": {
			a:      []string{"foo"},
			b:      []string{"foo"},
			expect: []string{"foo"},
		},
		"a equals b (many)": {
			a:      []string{"foo", "bar", "baz", "qux", "quux", "quuz"},
			b:      []string{"foo", "bar", "baz", "qux", "quux", "quuz"},
			expect: []string{"foo", "bar", "baz", "qux", "quux", "quuz"},
		},
		"a equals b but out of order": {
			a:      []string{"foo", "bar", "baz", "qux", "quux", "quuz"},
			b:      []string{"quuz", "bar", "qux", "foo", "quux", "baz"},
			expect: []string{"quuz", "bar", "qux", "foo", "quux", "baz"},
		},
		"a is superset": {
			a:      []string{"foo", "bar", "baz"},
			b:      []string{"foo"},
			expect: []string{"foo"},
		},
		"a is superset out of order": {
			a:      []string{"bar", "foo", "baz"},
			b:      []string{"foo"},
			expect: []string{"foo"},
		},
		"b is superset": {
			a:      []string{"foo"},
			b:      []string{"foo", "bar", "baz"},
			expect: []string{"foo"},
		},
		"b is superset out of order": {
			a:      []string{"foo"},
			b:      []string{"bar", "foo", "baz"},
			expect: []string{"foo"},
		},
		"a not equal to b": {
			a:      []string{"foo", "bar", "baz"},
			b:      []string{"qux", "quux", "quuz"},
			expect: []string{},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := intersectStrings(test.a, test.b)
			if !reflect.DeepEqual(actual, test.expect) {
				t.Fatalf("Actual: %#v\nExpected: %#v\n", actual, test.expect)
			}
		})
	}
}

func assertNotNil(t *testing.T, val interface{}) {
	t.Helper()
	if val == nil {
		t.Fatalf("expected not nil, but was nil")
	}
}

func assertNotEmptyString(t *testing.T, str string) {
	t.Helper()
	if str == "" {
		t.Fatalf("string is empty")
	}
}

func assertStrSliceIsNotEmpty(t *testing.T, strs []string) {
	t.Helper()
	if strs == nil || len(strs) == 0 {
		t.Fatalf("string slice is empty")
	}
}

func assertStrSliceIsEmpty(t *testing.T, strs []string) {
	t.Helper()
	if strs != nil && len(strs) > 0 {
		t.Fatalf("string slice is not empty")
	}
}

func strPtr(str string) *string {
	return &str
}
