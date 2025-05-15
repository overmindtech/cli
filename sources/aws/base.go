package aws

import (
	"fmt"

	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/shared"
)

type Base struct {
	accountID string
	region    string

	*shared.Base
}

func NewBase(
	accountID string,
	region string,
	category sdp.AdapterCategory,
	item shared.ItemType,
) *Base {
	return &Base{
		accountID: accountID,
		region:    region,
		Base: shared.NewBase(
			category,
			item,
			[]string{fmt.Sprintf("%s.%s", accountID, region)},
		),
	}
}

func (m *Base) AccountID() string {
	return m.accountID
}

func (m *Base) Region() string {
	return m.region
}
