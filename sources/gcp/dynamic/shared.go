package dynamic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"

	"github.com/overmindtech/workspace/discovery"
	"github.com/overmindtech/workspace/sdp-go"
	"github.com/overmindtech/workspace/sdpcache"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources"
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
			panic("searchDescription requires at least two unique attribute keys: " + sdpAssetType.String())
		}
		// For service directory endpoint adapter, the uniqueAttributeKeys is: []string{"locations", "namespaces", "services", "endpoints"}
		// We want to create a selector like:
		// locations|namespaces|services
		// We remove the last key, because it defines the actual item selector
		selector := "\"" + strings.Join(uniqueAttributeKeys[:len(uniqueAttributeKeys)-1], shared.QuerySeparator) + "\""

		return fmt.Sprintf("Search for %s by its %s", sdpAssetType, selector)
	}
)

// enrichNOTFOUNDQueryError sets Scope, SourceName, ItemType, ResponderName on a NOTFOUND QueryError when they are empty,
// so cached/returned errors have consistent metadata for debugging and cache inspection.
func enrichNOTFOUNDQueryError(err error, scope, sourceName, itemType string) {
	var qe *sdp.QueryError
	if err == nil || !errors.As(err, &qe) || qe.GetErrorType() != sdp.QueryError_NOTFOUND {
		return
	}
	if qe.GetScope() != "" {
		return
	}
	qe.Scope = scope
	qe.SourceName = sourceName
	qe.ItemType = itemType
	qe.ResponderName = sourceName
}

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

func externalToSDP(
	ctx context.Context,
	location gcpshared.LocationInfo,
	uniqueAttrKeys []string,
	resp map[string]interface{},
	sdpAssetType shared.ItemType,
	linker *gcpshared.Linker,
	nameSelector string,
) (*sdp.Item, error) {
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
		Scope:           location.ToScope(),
		Tags:            labels,
	}

	nameSel := nameSelector
	if nameSel == "" {
		nameSel = "name"
	}

	if name, ok := resp[nameSel].(string); ok {
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
		linkItem(ctx, location.ProjectID, sdpItem, sdpAssetType, linker, v, keys)
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
		body, readErr := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			// Return NOTFOUND regardless of body read so callers can cache via IsNotFound(err)
			return nil, &sdp.QueryError{
				ErrorType:   sdp.QueryError_NOTFOUND,
				ErrorString: fmt.Sprintf("resource not found: %s", url),
			}
		}
		if readErr == nil {
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
		}).Warnf("failed to read the response body: %v", readErr)
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
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				// Return QueryError NOTFOUND so callers (streamSDPItems, aggregateSDPItems) can cache via IsNotFound(err)
				return &sdp.QueryError{
					ErrorType:   sdp.QueryError_NOTFOUND,
					ErrorString: fmt.Sprintf("resource not found: %s", currentURL),
				}
			}
			if readErr == nil {
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
			}).Warnf("failed to read the response body: %v", readErr)
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
					"ovm.source.http.result":         result,
				}).Debug("not found any items in the result")
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
	potentialLinksMap := make(map[string]bool)
	for _, impact := range blasts[itemType] {
		potentialLinksMap[impact.ToSDPItemType.String()] = true
		// Special case: stdlib.NetworkIP and stdlib.NetworkDNS are interchangeable
		// because the linker automatically detects whether a value is an IP address or DNS name
		// If you specify either one, both are included in potential links
		if impact.ToSDPItemType.String() == "ip" || impact.ToSDPItemType.String() == "dns" {
			potentialLinksMap["ip"] = true
			potentialLinksMap["dns"] = true
		}
	}

	potentialLinks := make([]string, 0, len(potentialLinksMap))
	for it := range potentialLinksMap {
		potentialLinks = append(potentialLinks, it)
	}

	// Sort to ensure deterministic ordering
	slices.Sort(potentialLinks)

	return potentialLinks
}

// aggregateSDPItems retrieves items from an external API and converts them to SDP items.
func aggregateSDPItems(ctx context.Context, a Adapter, url string, location gcpshared.LocationInfo) ([]*sdp.Item, error) {
	var items []*sdp.Item
	itemsSelector := a.uniqueAttributeKeys[len(a.uniqueAttributeKeys)-1] // Use the last key as the item selector

	if a.listResponseSelector != "" {
		itemsSelector = a.listResponseSelector
	}

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

	hadExtractError := false
	var lastExtractErr error
	for resp := range out {
		item, err := externalToSDP(ctx, location, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker, a.nameSelector)
		if err != nil {
			log.WithError(err).Warn("failed to extract item from response")
			hadExtractError = true
			lastExtractErr = err
			continue
		}

		items = append(items, item)
	}

	err := p.Wait()
	if err != nil {
		// If we have items but the pool failed with NOTFOUND (e.g. 404 on a later pagination page),
		// return the items we collected so the caller does not cache NOTFOUND for a non-empty result.
		if sources.IsNotFound(err) && len(items) > 0 {
			return items, nil
		}
		return nil, err
	}

	// If all items failed extraction, return error so caller does not cache NOTFOUND (matches streamSDPItems)
	if len(items) == 0 && hadExtractError && lastExtractErr != nil {
		return nil, lastExtractErr
	}

	return items, nil
}

// streamSDPItems retrieves items from an external API and streams them as SDP items.
func streamSDPItems(ctx context.Context, a Adapter, url string, location gcpshared.LocationInfo, stream discovery.QueryResultStream, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) {
	itemsSelector := a.uniqueAttributeKeys[len(a.uniqueAttributeKeys)-1] // Use the last key as the item selector
	if a.listResponseSelector != "" {
		itemsSelector = a.listResponseSelector
	}

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

	itemsSent := 0
	hadExtractError := false
	for resp := range out {
		item, err := externalToSDP(ctx, location, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker, a.nameSelector)
		if err != nil {
			log.WithError(err).Warn("failed to extract item from response")
			hadExtractError = true
			continue
		}

		cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)
		itemsSent++

		stream.SendItem(item)
	}

	err := p.Wait()
	if err != nil {
		// Only cache NOTFOUND when no items were sent. For NOTFOUND, don't send error on stream
		// so behaviour matches cached path (0 items, no error). When items were already sent,
		// also don't send NOTFOUND (consistent with aggregateSDPItems returning items, nil).
		if sources.IsNotFound(err) && itemsSent == 0 {
			cache.StoreError(ctx, err, shared.DefaultCacheDuration, cacheKey)
		}
		if !sources.IsNotFound(err) {
			stream.SendError(err)
		}
	} else if itemsSent == 0 && !hadExtractError {
		// Cache not-found when no items were sent AND no extraction errors occurred
		// If we had extraction errors, items may exist but couldn't be processed
		notFoundErr := &sdp.QueryError{
			ErrorType:     sdp.QueryError_NOTFOUND,
			ErrorString:   fmt.Sprintf("no %s found in scope %s", a.sdpAssetType.String(), location.ToScope()),
			Scope:         location.ToScope(),
			SourceName:    a.Name(),
			ItemType:      a.sdpAssetType.String(),
			ResponderName: a.Name(),
		}
		cache.StoreError(ctx, notFoundErr, shared.DefaultCacheDuration, cacheKey)
	}
	// Note: No items found is valid. The caller's defer done() will release pending work.
}

func terraformMappingViaSearch(ctx context.Context, a Adapter, query string, location gcpshared.LocationInfo, cache sdpcache.Cache, cacheKey sdpcache.CacheKey) ([]*sdp.Item, error) {
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
		err := &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to handle terraform mapping from query %s for %s",
				query,
				a.sdpAssetType,
			),
		}
		cache.StoreError(ctx, err, shared.DefaultCacheDuration, cacheKey)
		return nil, err
	}

	// Reconstruct the query from the parts with default separator
	// For example, if the unique attribute keys are ["serviceAccounts", "keys"]
	// and the query parts are ["account", "key"], we get "account|key"
	query = strings.Join(queryParts, shared.QuerySeparator)

	// We use the GET endpoint for this query. Because the terraform mappings are for single items,
	getURL := a.getURLFunc(query, location)
	if getURL == "" {
		err := &sdp.QueryError{
			ErrorType: sdp.QueryError_OTHER,
			ErrorString: fmt.Sprintf(
				"failed to construct the URL for the query \"%s\". SEARCH method description: %s",
				query,
				a.Metadata().GetSupportedQueryMethods().GetSearchDescription(),
			),
		}
		cache.StoreError(ctx, err, shared.DefaultCacheDuration, cacheKey)
		return nil, err
	}

	resp, err := externalCallSingle(ctx, a.httpCli, getURL)
	if err != nil {
		enrichNOTFOUNDQueryError(err, location.ToScope(), a.Name(), a.Type())
		if sources.IsNotFound(err) {
			cache.StoreError(ctx, err, shared.DefaultCacheDuration, cacheKey)
			// Return empty result, nil error so behaviour matches cached NOTFOUND (caller converts to [], nil)
			return []*sdp.Item{}, nil
		}
		return nil, err
	}

	item, err := externalToSDP(ctx, location, a.uniqueAttributeKeys, resp, a.sdpAssetType, a.linker, a.nameSelector)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to convert response to SDP: %w", err)
		return nil, wrappedErr
	}

	cache.StoreItem(ctx, item, shared.DefaultCacheDuration, cacheKey)

	return []*sdp.Item{item}, nil
}
