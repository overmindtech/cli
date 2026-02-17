package sdp

import (
	"errors"
	"fmt"
)

// Validate ensures that an Item contains all required fields:
//   - Type: must be non-empty
//   - UniqueAttribute: must be non-empty
//   - Attributes: must not be nil
//   - Scope: must be non-empty
//   - UniqueAttributeValue: must be non-empty (derived from Attributes)
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

// Validate ensures a Reference contains all required fields:
//   - Type: must be non-empty
//   - UniqueAttributeValue: must be non-empty
//   - Scope: must be non-empty
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

// Validate ensures an Edge is valid by validating both the From and To references.
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

// Validate ensures a Response contains all required fields:
//   - Responder: must be non-empty
//   - UUID: must be non-empty
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

// Validate ensures a QueryError contains all required fields:
//   - UUID: must be non-empty
//   - ErrorString: must be non-empty
//   - Scope: must be non-empty
//   - SourceName: must be non-empty
//   - ItemType: must be non-empty
//   - ResponderName: must be non-empty
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

// Validate ensures a Query contains all required fields:
//   - Type: must be non-empty
//   - Scope: must be non-empty
//   - UUID: must be exactly 16 bytes
//   - Query: must be non-empty when method is GET
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
