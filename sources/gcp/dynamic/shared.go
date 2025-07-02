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

	"github.com/overmindtech/cli/sdp-go"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
	"github.com/overmindtech/cli/sources/shared"
)

var (
	getDescription = func(sdpAssetType shared.ItemType, scope string, uniqueAttributeKeys []string) string {
		selector := "{name}"
		if len(uniqueAttributeKeys) > 1 {
			// i.e.: {datasets|tables} for bigquery tables
			selector = "{" + strings.Join(uniqueAttributeKeys, shared.QuerySeparator) + "}"
		}

		return fmt.Sprintf("Get a %s by its %s within its scope: %s", sdpAssetType, selector, scope)
	}

	listDescription = func(sdpAssetType shared.ItemType, scope string) string {
		return fmt.Sprintf("List all %s within its scope: %s", sdpAssetType, scope)
	}

	searchDescription = func(sdpAssetType shared.ItemType, scope string, uniqueAttributeKeys []string, customSearchMethodDesc string) string {
		if customSearchMethodDesc != "" {
			return customSearchMethodDesc
		}

		if len(uniqueAttributeKeys) < 2 {
			panic("searchDescription requires at least two unique attribute keys")
		}
		// For service directory endpoint adapter, the uniqueAttributeKeys is: []string{"locations", "namespaces", "services", "endpoints"}
		// We want to create a selector like:
		// {locations|namespaces|services}
		// We remove the last key, because it defines the actual item selector
		selector := "{" + strings.Join(uniqueAttributeKeys[:len(uniqueAttributeKeys)-1], shared.QuerySeparator) + "}"

		return fmt.Sprintf("Search for %s by its %s within its scope: %s", sdpAssetType, selector, scope)
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
			return nil, fmt.Errorf(
				"failed to make a GET call: %s, HTTP Status: %s, HTTP Body: %s",
				url,
				resp.Status,
				string(body),
			)
		}

		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.gcp.dynamic.http.get.url":            url,
			"ovm.gcp.dynamic.http.get.responseStatus": resp.Status,
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

// externalCallMulti makes a paginated HTTP GET request to the specified URL and returns all items found in the response.
// It handles pagination by checking for a "nextPageToken" in the response and continues to fetch until no more pages are available.
// This function can return items along with an error if a consecutive HTTP request fails or if the response cannot be parsed correctly.
// Therefore, it is important to check both the returned items and the error.
func externalCallMulti(ctx context.Context, itemsSelector string, httpCli *http.Client, urlForList string) ([]map[string]interface{}, error) {
	var allItems []map[string]interface{}
	currentURL := urlForList

	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
		if err != nil {
			return nil, err
		}

		resp, err := httpCli.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			// Read the body to provide more context in the error message
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close() // Close the response body
			if err == nil {
				return allItems, fmt.Errorf(
					"failed to make the GET call. HTTP Status: %s, HTTP Body: %s",
					resp.Status,
					string(body),
				)
			}

			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.gcp.dynamic.http.get.urlForList":     currentURL,
				"ovm.gcp.dynamic.http.get.responseStatus": resp.Status,
			}).Warnf("failed to read the response body: %v", err)
			return allItems, fmt.Errorf("failed to make the GET call. HTTP Status: %s", resp.Status)
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return allItems, err
		}

		var result map[string]interface{}
		if err = json.Unmarshal(data, &result); err != nil {
			return allItems, err
		}

		// Extract items from the current page
		itemsAny, ok := result[itemsSelector]
		if !ok {
			itemsSelector = "items" // Fallback to a generic "items" key
			itemsAny, ok = result[itemsSelector]
			if !ok {
				log.WithContext(ctx).WithFields(log.Fields{
					"ovm.gcp.dynamic.http.get.urlForList":    currentURL,
					"ovm.gcp.dynamic.http.get.itemsSelector": itemsSelector,
				}).Debugf("not found any items for %s: within %v", itemsSelector, result)
				break
			}
		}

		items, ok := itemsAny.([]any)
		if !ok {
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.gcp.dynamic.http.get.urlForList":    currentURL,
				"ovm.gcp.dynamic.http.get.itemsSelector": itemsSelector,
			}).Warnf("failed to cast resp as a list of %s: within %v", itemsSelector, result)
			break
		}

		// Add items from this page to our collection
		for _, item := range items {
			if itemMap, ok := item.(map[string]interface{}); ok {
				allItems = append(allItems, itemMap)
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
			return allItems, fmt.Errorf("failed to parse URL %s: %w", urlForList, err)
		}

		// Get existing query parameters or create new ones
		query := parsedURL.Query()
		query.Set("pageToken", nextPageToken)
		parsedURL.RawQuery = query.Encode()

		// Use the properly constructed URL for the next request
		currentURL = parsedURL.String()
	}

	return allItems, nil
}

func potentialLinksFromBlasts(itemType shared.ItemType, blasts map[shared.ItemType]map[string]*gcpshared.Impact) []string {
	var potentialLinks []string
	var potentialLinksMap = make(map[string]bool)
	for _, impact := range blasts[itemType] {
		potentialLinksMap[impact.ToSDPITemType.String()] = true
	}

	for it := range potentialLinksMap {
		potentialLinks = append(potentialLinks, it)
	}

	return potentialLinks
}
