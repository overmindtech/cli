package dynamic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	getDescription = func(sdpAssetType shared.ItemType, uniqueAttributeKeys []string) string {
		selector := "\"name\""
		if len(uniqueAttributeKeys) > 1 {
			// i.e.: "datasets|tables" for bigquery tables
			selector = "\"" + strings.Join(uniqueAttributeKeys, shared.QuerySeparator) + "\""
		}

		return fmt.Sprintf("Get a %s by its %s", sdpAssetType, selector)
	}

	listDescription = func(sdpAssetType shared.ItemType) string {
		return fmt.Sprintf("List all %s", sdpAssetType)
	}

	searchDescription = func(sdpAssetType shared.ItemType, uniqueAttributeKeys []string, customSearchMethodDesc string) string {
		if customSearchMethodDesc != "" {
			return customSearchMethodDesc
		}

		if len(uniqueAttributeKeys) < 2 {
			panic("searchDescription requires at least two unique attribute keys")
		}
		// For service directory endpoint adapter, the uniqueAttributeKeys is: []string{"locations", "namespaces", "services", "endpoints"}
		// We want to create a selector like:
		// locations|namespaces|services
		// We remove the last key, because it defines the actual item selector
		selector := "\"" + strings.Join(uniqueAttributeKeys[:len(uniqueAttributeKeys)-1], shared.QuerySeparator) + "\""

		return fmt.Sprintf("Search for %s by its %s", sdpAssetType, selector)
	}
)

func linkItem(ctx context.Context, projectID string, sdpItem *sdp.Item, sdpAssetType shared.ItemType, linker *gcpshared.Linker, resp any, keys []string) {
	if value, ok := resp.(string); ok {
		linker.AutoLink(ctx, projectID, sdpItem, sdpAssetType, value, keys)
		return
	}

	if listAny, ok := resp.([]any); ok {
		for _, v := range listAny {
			linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, v, keys)
		}
		return
	}

	if mapAny, ok := resp.(map[string]any); ok {
		for k, item := range mapAny {
			linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, item, append(keys, k))
		}
		return
	}
}

func externalToSDP(ctx context.Context, projectID string, scope string, uniqueAttrKeys []string, resp map[string]interface{}, sdpAssetType shared.ItemType, linker *gcpshared.Linker) (*sdp.Item, error) {
	attributes, err := shared.ToAttributesWithExclude(resp, "labels")
	if err != nil {
		return nil, err
	}

	labels := make(map[string]string)
	if lls, ok := resp["labels"]; ok {
		if labelsAny, ok := lls.(map[string]any); ok {
			for lk, lv := range labelsAny {
				// Convert the label value to string
				labels[lk] = fmt.Sprintf("%v", lv)
			}
		}

	}

	sdpItem := &sdp.Item{
		Type:            sdpAssetType.String(),
		UniqueAttribute: "uniqueAttr",
		Attributes:      attributes,
		Scope:           scope,
		Tags:            labels,
	}

	// We need to keep an eye on this.
	// Name might not exist in the response for all APIs.
	if name, ok := resp["name"].(string); ok {
		attrValues := gcpshared.ExtractPathParams(name, uniqueAttrKeys...)
		uniqueAttrValue := strings.Join(attrValues, shared.QuerySeparator)
		err = sdpItem.GetAttributes().Set("uniqueAttr", uniqueAttrValue)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unable to determine the name")
	}

	for k, v := range resp {
		keys := []string{k}
		linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, v, keys)
	}

	return sdpItem, nil
}

func externalCallSingle(ctx context.Context, httpCli *http.Client, url string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpCli.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			if resp.StatusCode == http.StatusForbidden {
				return nil, &PermissionError{URL: url}
			}

			return nil, fmt.Errorf(
				"failed to make a GET call: %s, HTTP Status: %s, HTTP Body: %s",
				url,
				resp.Status,
				string(body),
			)
		}

		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":                 "gcp",
			"ovm.source.http.url":             url,
			"ovm.source.http.response-status": resp.Status,
		}).Warnf("failed to read the response body: %v", err)
		return nil, fmt.Errorf("failed to make call: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// externalCallMulti makes a paginated HTTP GET request to the specified URL and sends the results to the provided output channel.
func externalCallMulti(ctx context.Context, itemsSelector string, httpCli *http.Client, urlForList string, out chan<- map[string]any) error {
	if out == nil {
		return fmt.Errorf("no output channel provided")
	}

	currentURL := urlForList
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
		if err != nil {
			return err
		}

		resp, err := httpCli.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			// Read the body to provide more context in the error message
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close() // Close the response body
			if err == nil {
				return fmt.Errorf(
					"failed to make the GET call. HTTP Status: %s, HTTP Body: %s",
					resp.Status,
					string(body),
				)
			}

			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.source.type":                 "gcp",
				"ovm.source.http.url-for-list":    currentURL,
				"ovm.source.http.response-status": resp.Status,
			}).Warnf("failed to read the response body: %v", err)
			return fmt.Errorf("failed to make the GET call. HTTP Status: %s", resp.Status)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		var result map[string]interface{}
		if err = json.Unmarshal(data, &result); err != nil {
			return err
		}

		// Extract items from the current page
		itemsAny, ok := result[itemsSelector]
		if !ok {
			itemsSelector = "items" // Fallback to a generic "items" key
			itemsAny, ok = result[itemsSelector]
			if !ok {
				log.WithContext(ctx).WithFields(log.Fields{
					"ovm.source.type":                "gcp",
					"ovm.source.http.url-for-list":   currentURL,
					"ovm.source.http.items-selector": itemsSelector,
				}).Debugf("not found any items for %s: within %v", itemsSelector, result)
				break
			}
		}

		items, ok := itemsAny.([]any)
		if !ok {
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.source.http.url-for-list":   currentURL,
				"ovm.source.http.items-selector": itemsSelector,
			}).Warnf("failed to cast resp as a list of %s: within %v", itemsSelector, result)
			break
		}

		// Add items from this page to our collection
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				// If out channel is provided, send the item to it
				select {
				case out <- itemMap:
				case <-ctx.Done():
					log.WithContext(ctx).Warn("context cancelled while sending items")
					return ctx.Err()
				}
			}
		}

		// Check if there's a next page
		nextPageToken, ok := result["nextPageToken"].(string)
		if !ok || nextPageToken == "" {
			break // No more pages to process
		}

		// Properly construct the next page URL with the pageToken
		parsedURL, err := url.Parse(urlForList)
		if err != nil {
			return fmt.Errorf("failed to parse URL %s: %w", urlForList, err)
		}

		// Get existing query parameters or create new ones
		query := parsedURL.Query()
		query.Set("pageToken", nextPageToken)
		parsedURL.RawQuery = query.Encode()

		// Use the properly constructed URL for the next request
		currentURL = parsedURL.String()
	}

	return nil
}

func potentialLinksFromBlasts(itemType shared.ItemType, blasts map[shared.ItemType]map[string]*gcpshared.Impact) []string {
	var potentialLinks []string
	var potentialLinksMap = make(map[string]bool)
	for _, impact := range blasts[itemType] {
		potentialLinksMap[impact.ToSDPItemType.String()] = true
	}

	for it := range potentialLinksMap {
		potentialLinks = append(potentialLinks, it)
	}

	return potentialLinks
}

// aggregateSDPItems retrieves items from an external API and converts them to SDP items.
func aggregateSDPItems(ctx context.Context, a Adapter, url string) ([]*sdp.Item, error) {
	var items []*sdp.Item
	itemsSelector := a.uniqueAttributeKeys[len(a.uniqueAttributeKeys)-1] // Use the last key as the item selector

	out := make(chan map[string]interface{})
	p := pool.New().WithErrors().WithContext(ctx)
	p.Go(func(ctx context.Context) error {
		defer close(out)
		err := externalCallMulti(ctx, itemsSelector, a.httpCli, url, out)
		if err != nil {
			return fmt.Errorf("failed to retrieve items for %s: %w", url, err)
		}
		return nil
	},
	)

	for resp := range out {
		item, err := externalToSDP(ctx, a.projectID, a.scope, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker)
		if err != nil {
			log.WithError(err).Warn("failed to extract item from response")
		}

		items = append(items, item)
	}

	err := p.Wait()
	if err != nil {
		return nil, err
	}

	return items, nil
}

// streamSDPItems retrieves items from an external API and streams them as SDP items.
func streamSDPItems(ctx context.Context, a Adapter, url string, stream discovery.QueryResultStream, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	itemsSelector := a.uniqueAttributeKeys[len(a.uniqueAttributeKeys)-1] // Use the last key as the item selector

	out := make(chan map[string]interface{})
	p := pool.New().WithErrors().WithContext(ctx)
	p.Go(func(ctx context.Context) error {
		defer close(out)
		err := externalCallMulti(ctx, itemsSelector, a.httpCli, url, out)
		if err != nil {
			return fmt.Errorf("failed to retrieve items for %s: %w", url, err)
		}
		return nil
	})

	for resp := range out {
		item, err := externalToSDP(ctx, a.projectID, a.scope, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker)
		if err != nil {
			log.WithError(err).Warn("failed to extract item from response")
			continue
		}

		cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)

		stream.SendItem(item)
	}

	err := p.Wait()
	if err != nil {
		stream.SendError(err)
	}
}

func terraformMappingViaSearch(ctx context.Context, a Adapter, query string, cache *sdpcache.Cache, cacheKey sdpcache.CacheKey) ([]*sdp.Item, error) {
	// query is in the format of:
	// projects/{{project}}/datasets/{{dataset}}/tables/{{name}}
	// projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
	//
	// Extract the relevant parts from the query
	// We need to extract the path parameters based on the number of unique attribute keys
	// From projects/{{project}}/serviceAccounts/{{account}}/keys/{{key}}
	// we get: ["account", "key"]
	// if the unique attribute keys are ["serviceAccounts", "keys"]
	queryParts := gcpshared.ExtractPathParamsWithCount(query, len(a.uniqueAttributeKeys))
	if len(queryParts) != len(a.uniqueAttributeKeys) {
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to handle terraform mapping from query %s for %s",
				query,
				a.sdpAssetType,
			),
		}
	}

	// Reconstruct the query from the parts with default separator
	// For example, if the unique attribute keys are ["serviceAccounts", "keys"]
	// and the query parts are ["account", "key"], we get "account|key"
	query = strings.Join(queryParts, shared.QuerySeparator)

	// We use the GET endpoint for this query. Because the terraform mappings are for single items,
	getURL := a.getURLFunc(query)
	if getURL == "" {
		return nil, &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to construct the URL for the query \"%s\". SEARCH method description: %s",
				query,
				a.Metadata().GetSupportedQueryMethods().GetSearchDescription(),
			),
		}
	}

	resp, err := externalCallSingle(ctx, a.httpCli, getURL)
	if err != nil {
		return nil, err
	}

	item, err := externalToSDP(ctx, a.projectID, a.scope, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response to SDP: %w", err)
	}

	cache.StoreItem(item, shared.DefaultCacheDuration, cacheKey)

	return []*sdp.Item{item}, nil
}
