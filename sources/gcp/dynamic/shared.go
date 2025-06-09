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

func linkItem(ctx context.Context, projectID string, sdpItem *sdp.Item, sdpAssetType shared.ItemType, linker *gcpshared.Linker, resp any) {
	if value, ok := resp.(string); ok {
		linker.AutoLink(ctx, projectID, sdpItem, sdpAssetType, value)
		return
	}

	if listAny, ok := resp.([]any); ok {
		for _, v := range listAny {
			linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, v)
		}
		return
	}

	if mapAny, ok := resp.(map[string]any); ok {
		for _, item := range mapAny {
			linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, item)
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
	if name, ok := resp["name"].(string); ok {
		attrValues := gcpshared.ExtractPathParams(name, uniqueAttrKeys...)
		uniqueAttrValue := strings.Join(attrValues, shared.QuerySeparator)
		err = sdpItem.GetAttributes().Set("uniqueAttr", uniqueAttrValue)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unable to determine self link")
	}

	for _, v := range resp {
		linkItem(ctx, projectID, sdpItem, sdpAssetType, linker, v)
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

func externalCallMulti(ctx context.Context, httpCli *http.Client, httpHeader http.Header, url string) ([]map[string]interface{}, error) {
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
		return nil, fmt.Errorf("failed to make the GET call for the %s URL. HTTP Status: %s", url, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err = json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	items, ok := result["items"].([]any)
	if !ok {
		log.WithContext(ctx).WithFields(log.Fields{
			"url": url,
		}).Warnf("failed to cast resp as a list of items: %v", result)

		return nil, nil
	}

	var ii []map[string]interface{}
	for _, item := range items {
		if itemMap, ok := item.(map[string]interface{}); ok {
			ii = append(ii, itemMap)
		}
	}

	return ii, nil
}
