package plugin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Using the same time parsing logic from https://github.com/coreos/go-oidc
// This code is licensed under the Apache 2.0 license
type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	var unix int64

	if t, err := n.Int64(); err == nil {
		unix = t
	} else {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		unix = int64(f)
	}
	*j = jsonTime(time.Unix(unix, 0))
	return nil
}

// ResourceID represents a parsed long-form Azure Resource Manager ID
// with the Subscription ID, Resource Group and the Provider as top-
// level fields, and other key-value pairs available via a map in the
// Path field.
type ResourceID struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	Path           map[string]string
}

// parseAzureResourceID converts a long-form Azure Resource Manager ID
// into a ResourceID.
func parseAzureResourceID(id string) (*ResourceID, error) {
	path := strings.TrimSpace(id)
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	components := strings.Split(path, "/")
	// We should have an even number of key-value pairs.
	if len(components)%2 != 0 {
		return nil, fmt.Errorf("The number of path segments is not divisible by 2 in %q", path)
	}

	var subscriptionID string

	// Put the constituent key-value pairs into a map
	componentMap := make(map[string]string, len(components)/2)
	for current := 0; current < len(components); current += 2 {
		key := components[current]
		value := components[current+1]

		// Check key/value for empty strings.
		if key == "" || value == "" {
			return nil, fmt.Errorf("Key/Value cannot be empty strings. Key: '%s', Value: '%s'", key, value)
		}

		// Catch the subscriptionID before it can be overwritten by another "subscriptions"
		// value in the ID which is the case for the Service Bus subscription resource
		if key == "subscriptions" && subscriptionID == "" {
			subscriptionID = value
		} else {
			componentMap[key] = value
		}
	}

	// Build up a ResourceID from the map
	resourceID := &ResourceID{
		SubscriptionID: subscriptionID,
	}

	if subscriptionID != "" {
		resourceID.SubscriptionID = subscriptionID
	} else {
		return nil, fmt.Errorf("No subscription ID found in: %q", path)
	}

	if resourceGroup, ok := componentMap["resourceGroups"]; ok {
		resourceID.ResourceGroup = resourceGroup
		delete(componentMap, "resourceGroups")
	} else {
		// Some Azure APIs are weird and provide things in lower case...
		// However it's not clear whether the casing of other elements in the URI
		// matter, so we explicitly look for that case here.
		if resourceGroup, ok := componentMap["resourcegroups"]; ok {
			resourceID.ResourceGroup = resourceGroup
			delete(componentMap, "resourcegroups")
		} else {
			return nil, fmt.Errorf("No resource group name found in: %q", path)
		}
	}

	// It is OK not to have a provider in the case of a resource group
	if provider, ok := componentMap["providers"]; ok {
		resourceID.Provider = provider
		delete(componentMap, "providers")
	}

	return resourceID, nil
}
