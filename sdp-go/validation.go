package sdp

import (
	"errors"
	"fmt"
)

// Validate Ensures that en item is valid (e.g. contains the required fields)
func (i *Item) Validate() error {
	if i == nil {
		return errors.New("Item is nil")
	}

	if i.GetType() == "" {
		return fmt.Errorf("item has empty Type: %v", i.GloballyUniqueName())
	}

	if i.GetUniqueAttribute() == "" {
		return fmt.Errorf("item has empty UniqueAttribute: %v", i.GloballyUniqueName())
	}

	if i.GetAttributes() == nil {
		return fmt.Errorf("item has nil Attributes: %v", i.GloballyUniqueName())
	}

	if i.GetScope() == "" {
		return fmt.Errorf("item has empty Scope: %v", i.GloballyUniqueName())
	}

	if i.UniqueAttributeValue() == "" {
		return fmt.Errorf("item has empty UniqueAttributeValue: %v", i.GloballyUniqueName())
	}

	return nil
}

// Validate Ensures a reference is valid
func (r *Reference) Validate() error {
	if r == nil {
		return errors.New("reference is nil")
	}
	if r.GetType() == "" {
		return fmt.Errorf("reference has empty Type: %v", r)
	}
	if r.GetUniqueAttributeValue() == "" {
		return fmt.Errorf("reference has empty UniqueAttributeValue: %v", r)
	}
	if r.GetScope() == "" {
		return fmt.Errorf("reference has empty Scope: %v", r)
	}

	return nil
}

// Validate Ensures an edge is valid
func (e *Edge) Validate() error {
	if e == nil {
		return errors.New("edge is nil")
	}

	var err error

	err = e.GetFrom().Validate()
	if err != nil {
		return err
	}

	err = e.GetTo().Validate()

	return err
}

// Validate Ensures a Response is valid
func (r *Response) Validate() error {
	if r == nil {
		return errors.New("response is nil")
	}

	if r.GetResponder() == "" {
		return fmt.Errorf("response has empty Responder: %v", r)
	}

	if len(r.GetUUID()) == 0 {
		return fmt.Errorf("response has empty UUID: %v", r)
	}

	return nil
}

// Validate Ensures a QueryError is valid
func (e *QueryError) Validate() error {
	if e == nil {
		return errors.New("queryError is nil")
	}

	if len(e.GetUUID()) == 0 {
		return fmt.Errorf("queryError has empty UUID: %w", e)
	}

	if e.GetErrorString() == "" {
		return fmt.Errorf("queryError has empty ErrorString: %w", e)
	}

	if e.GetScope() == "" {
		return fmt.Errorf("queryError has empty Scope: %w", e)
	}

	if e.GetSourceName() == "" {
		return fmt.Errorf("queryError has empty SourceName: %w", e)
	}

	if e.GetItemType() == "" {
		return fmt.Errorf("queryError has empty ItemType: %w", e)
	}

	if e.GetResponderName() == "" {
		return fmt.Errorf("queryError has empty ResponderName: %w", e)
	}

	return nil
}

// Validate Ensures a Query is valid
func (q *Query) Validate() error {
	if q == nil {
		return errors.New("query is nil")
	}

	if q.GetType() == "" {
		return fmt.Errorf("query has empty Type: %v", q)
	}

	if q.GetScope() == "" {
		return fmt.Errorf("query has empty Scope: %v", q)
	}

	if len(q.GetUUID()) != 16 {
		return fmt.Errorf("query has invalid UUID: %v", q)
	}

	if q.GetMethod() == QueryMethod_GET {
		if q.GetQuery() == "" {
			return fmt.Errorf("query cannot have empty Query when method is Get: %v", q)
		}
	}

	return nil
}
