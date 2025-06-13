package dynamic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	searchDescription = func(sdpAssetType shared.ItemType, scope string, uniqueAttributeKeys []string) string {
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

func externalCallSingle(ctx context.Context, httpCli *http.Client, httpHeaders http.Header, url string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = httpHeaders
	resp, err := httpCli.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return nil, fmt.Errorf("failed to make a GET call: %s, HTTP Status: %s, HTTP Body: %s", url, resp.Status, string(body))
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

func externalCallMulti(ctx context.Context, itemsSelector string, httpCli *http.Client, httpHeader http.Header, url string) ([]map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header = httpHeader
	resp, err := httpCli.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the body to provide more context in the error message
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			return nil, fmt.Errorf("failed to make the GET call. HTTP Status: %s, HTTP Body: %s", resp.Status, string(body))
		}

		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.gcp.dynamic.http.get.url":            url,
			"ovm.gcp.dynamic.http.get.responseStatus": resp.Status,
		}).Warnf("failed to read the response body: %v", err)
		return nil, fmt.Errorf("failed to make the GET callL. HTTP Status: %s", resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	items, ok := result[itemsSelector].([]any)
	if !ok {
		itemsSelector = "items" // Fallback to a generic "items" key
		items, ok = result[itemsSelector].([]any)
		if !ok {
			log.WithContext(ctx).WithFields(log.Fields{
				"ovm.gcp.dynamic.http.get.url":           url,
				"ovm.gcp.dynamic.http.get.itemsSelector": itemsSelector,
			}).Warnf("failed to cast resp as a list of %s: %v", itemsSelector, result)
			return nil, nil
		}
	}

	var ii []map[string]interface{}
	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			ii = append(ii, itemMap)
		}
	}

	return ii, nil
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
