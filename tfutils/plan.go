package tfutils

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// NOTE: These definitions are copied from the
// https://pkg.go.dev/github.com/hashicorp/terraform/internal/command/jsonplan
// package, which is internal so should be imported directly. Hence why we have
// copied them here

// Plan is the top-level representation of the json format of a plan. It includes
// the complete config and current state.
type Plan struct {
	FormatVersion    string      `json:"format_version,omitempty"`
	TerraformVersion string      `json:"terraform_version,omitempty"`
	Variables        Variables   `json:"variables,omitempty"`
	PlannedValues    StateValues `json:"planned_values,omitempty"`
	// ResourceDrift and ResourceChanges are sorted in a user-friendly order
	// that is undefined at this time, but consistent.
	ResourceDrift      []ResourceChange  `json:"resource_drift,omitempty"`
	ResourceChanges    []ResourceChange  `json:"resource_changes,omitempty"`
	OutputChanges      map[string]Change `json:"output_changes,omitempty"`
	PriorState         State             `json:"prior_state,omitempty"`
	Config             planConfig        `json:"configuration,omitempty"`
	RelevantAttributes []ResourceAttr    `json:"relevant_attributes,omitempty"`
	Checks             json.RawMessage   `json:"checks,omitempty"`
	Timestamp          string            `json:"timestamp,omitempty"`
	Errored            bool              `json:"errored"`
}

// Config represents the complete configuration source
type planConfig struct {
	ProviderConfigs map[string]ProviderConfig `json:"provider_config,omitempty"`
	RootModule      ConfigModule              `json:"root_module,omitempty"`
}

// ProviderConfig describes all of the provider configurations throughout the
// configuration tree, flattened into a single map for convenience since
// provider configurations are the one concept in Terraform that can span across
// module boundaries.
type ProviderConfig struct {
	Name              string                 `json:"name,omitempty"`
	FullName          string                 `json:"full_name,omitempty"`
	Alias             string                 `json:"alias,omitempty"`
	VersionConstraint string                 `json:"version_constraint,omitempty"`
	ModuleAddress     string                 `json:"module_address,omitempty"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
}

type ConfigModule struct {
	Outputs map[string]output `json:"outputs,omitempty"`
	// Resources are sorted in a user-friendly order that is undefined at this
	// time, but consistent.
	Resources   []ConfigResource      `json:"resources,omitempty"`
	ModuleCalls map[string]moduleCall `json:"module_calls,omitempty"`
	Variables   variables             `json:"variables,omitempty"`
}

var escapeRegex = regexp.MustCompile(`\${([\w\.\[\]]*)}`)

// Digs for a config resource in this module or its children
func (m ConfigModule) DigResource(address string) *ConfigResource {
	addressSections := strings.Split(address, ".")

	if len(addressSections) == 0 {
		return nil
	}

	if addressSections[0] == "module" {
		// If it's addressed to a module, then we need to dig into that module
		if len(addressSections) < 2 {
			return nil
		}

		moduleName := addressSections[1]

		if module, ok := m.ModuleCalls[moduleName]; ok {
			// Dig through the correct module
			return module.Module.DigResource(strings.Join(addressSections[2:], "."))
		}
	} else {
		// If the address has brackets, than we need to extract the index and
		// return the resource at that index
		indexMatches := indexBrackets.FindStringSubmatch(address)
		var desiredIndex int
		var err error

		if len(indexMatches) == 0 {
			// Return the first result
			desiredIndex = 0
		} else {
			desiredIndex, err = strconv.Atoi(indexMatches[1])

			if err != nil {
				return nil
			}
		}

		// Remove the [] from the address if it exists
		address = indexBrackets.ReplaceAllString(address, "")

		// Look through the current module
		currentIndex := 0
		for _, r := range m.Resources {
			if r.Address == address {
				if currentIndex == desiredIndex {
					return &r
				}

				currentIndex++
			}
		}
	}

	return nil
}

type moduleCall struct {
	Source            string                 `json:"source,omitempty"`
	Expressions       map[string]interface{} `json:"expressions,omitempty"`
	CountExpression   *expression            `json:"count_expression,omitempty"`
	ForEachExpression *expression            `json:"for_each_expression,omitempty"`
	Module            ConfigModule           `json:"module,omitempty"`
	VersionConstraint string                 `json:"version_constraint,omitempty"`
	DependsOn         []string               `json:"depends_on,omitempty"`
}

// variables is the JSON representation of the variables provided to the current
// plan.
type variables map[string]*variable

type variable struct {
	Default     json.RawMessage `json:"default,omitempty"`
	Description string          `json:"description,omitempty"`
	Sensitive   bool            `json:"sensitive,omitempty"`
}

// Resource is the representation of a resource in the config
type ConfigResource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// ProviderConfigKey is the key into "provider_configs" (shown above) for
	// the provider configuration that this resource is associated with.
	//
	// NOTE: If a given resource is in a ModuleCall, and the provider was
	// configured outside of the module (in a higher level configuration file),
	// the ProviderConfigKey will not match a key in the ProviderConfigs map.
	ProviderConfigKey string `json:"provider_config_key,omitempty"`

	// Provisioners is an optional field which describes any provisioners.
	// Connection info will not be included here.
	Provisioners []provisioner `json:"provisioners,omitempty"`

	// Expressions" describes the resource-type-specific  content of the
	// configuration block.
	Expressions map[string]interface{} `json:"expressions,omitempty"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// CountExpression and ForEachExpression describe the expressions given for
	// the corresponding meta-arguments in the resource configuration block.
	// These are omitted if the corresponding argument isn't set.
	CountExpression   *expression `json:"count_expression,omitempty"`
	ForEachExpression *expression `json:"for_each_expression,omitempty"`

	DependsOn []string `json:"depends_on,omitempty"`
}

type output struct {
	Sensitive   bool       `json:"sensitive,omitempty"`
	Expression  expression `json:"expression,omitempty"`
	DependsOn   []string   `json:"depends_on,omitempty"`
	Description string     `json:"description,omitempty"`
}

type provisioner struct {
	Type        string                 `json:"type,omitempty"`
	Expressions map[string]interface{} `json:"expressions,omitempty"`
}

// expression represents any unparsed expression
type expression struct {
	// "constant_value" is set only if the expression contains no references to
	// other objects, in which case it gives the resulting constant value. This
	// is mapped as for the individual values in the common value
	// representation.
	ConstantValue json.RawMessage `json:"constant_value,omitempty"`

	// Alternatively, "references" will be set to a list of references in the
	// expression. Multi-step references will be unwrapped and duplicated for
	// each significant traversal step, allowing callers to more easily
	// recognize the objects they care about without attempting to parse the
	// expressions. Callers should only use string equality checks here, since
	// the syntax may be extended in future releases.
	References []string `json:"references,omitempty"`
}

// Variables is the JSON representation of the variables provided to the current
// plan.
type Variables map[string]*Variable

type Variable struct {
	Value json.RawMessage `json:"value,omitempty"`
}

// StateValues is the common representation of resolved values for both the
// prior state (which is always complete) and the planned new state.
type StateValues struct {
	Outputs    map[string]Output `json:"outputs,omitempty"`
	RootModule Module            `json:"root_module,omitempty"`
}

// Get a specific resource from this module or its children
func (m Module) DigResource(address string) *Resource {
	// Look through the current module
	for _, r := range m.Resources {
		if r.Address == address {
			return &r
		}
	}

	// Look through children
	for _, child := range m.ChildModules {
		resource := child.DigResource(address)

		if resource != nil {
			return resource
		}
	}

	return nil
}

// Module is the representation of a module in state. This can be the root
// module or a child module.
type Module struct {
	// Resources are sorted in a user-friendly order that is undefined at this
	// time, but consistent.
	Resources []Resource `json:"resources,omitempty"`

	// Address is the absolute module address, omitted for the root module
	Address string `json:"address,omitempty"`

	// Each module object can optionally have its own nested "child_modules",
	// recursively describing the full module tree.
	ChildModules []Module `json:"child_modules,omitempty"`
}

// Resource is the representation of a resource in the json plan
type Resource struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// Mode can be "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type string `json:"type,omitempty"`
	Name string `json:"name,omitempty"`

	// ProviderName allows the property "type" to be interpreted unambiguously
	// in the unusual situation where a provider offers a resource type whose
	// name does not start with its own name, such as the "googlebeta" provider
	// offering "google_compute_instance".
	ProviderName string `json:"provider_name,omitempty"`

	// SchemaVersion indicates which version of the resource type schema the
	// "values" property conforms to.
	SchemaVersion uint64 `json:"schema_version"`

	// AttributeValues is the JSON representation of the attribute values of the
	// resource, whose structure depends on the resource type schema. Any
	// unknown values are omitted or set to null, making them indistinguishable
	// from absent values.
	AttributeValues AttributeValues `json:"values,omitempty"`

	// SensitiveValues is similar to AttributeValues, but with all sensitive
	// values replaced with true, and all non-sensitive leaf values omitted.
	SensitiveValues json.RawMessage `json:"sensitive_values,omitempty"`
}

// AttributeValues is the JSON representation of the attribute values of the
// resource, whose structure depends on the resource type schema.
type AttributeValues map[string]interface{}

var indexBrackets = regexp.MustCompile(`\[(\d+)\]`)

// Digs through the attribute values to find the value at the given key. This
// supports nested keys i.e. "foo.bar" and arrays i.e. "foo[0]"
func (av AttributeValues) Dig(key string) (interface{}, bool) {
	sections := strings.Split(key, ".")

	if len(sections) == 0 {
		return nil, false
	}

	// Get the first section
	section := sections[0]

	// Check for an index
	indexMatches := indexBrackets.FindStringSubmatch(section)

	var value interface{}
	var ok bool

	if len(indexMatches) == 0 {
		// No index, just get the value
		value, ok = av[section]

		if !ok {
			return nil, false
		}
	} else {
		// Get the index
		index, err := strconv.Atoi(indexMatches[1])

		if err != nil {
			return nil, false
		}

		// Get the value
		keyName := indexBrackets.ReplaceAllString(section, "")
		arr, ok := av[keyName]

		if !ok {
			return nil, false
		}

		// Check if the value is an array
		array, ok := arr.([]interface{})

		if !ok {
			return nil, false
		}

		// Check if the index is in range
		if index < 0 || index >= len(array) {
			return nil, false
		}

		value = array[index]
	}

	// If there are no more sections, then we're done
	if len(sections) == 1 {
		return value, true
	}

	// If there are more sections, then we need to dig deeper
	childMap, ok := value.(map[string]interface{})

	if !ok {
		return nil, false
	}

	childAttributeValues := AttributeValues(childMap)

	return childAttributeValues.Dig(strings.Join(sections[1:], "."))
}

type Output struct {
	Sensitive bool            `json:"sensitive"`
	Type      json.RawMessage `json:"type,omitempty"`
	Value     json.RawMessage `json:"value,omitempty"`
}

// ResourceChange is a description of an individual change action that Terraform
// plans to use to move from the prior state to a new state matching the
// configuration.
type ResourceChange struct {
	// Address is the absolute resource address
	Address string `json:"address,omitempty"`

	// PreviousAddress is the absolute address that this resource instance had
	// at the conclusion of a previous run.
	//
	// This will typically be omitted, but will be present if the previous
	// resource instance was subject to a "moved" block that we handled in the
	// process of creating this plan.
	//
	// Note that this behavior diverges from the internal plan data structure,
	// where the previous address is set equal to the current address in the
	// common case, rather than being omitted.
	PreviousAddress string `json:"previous_address,omitempty"`

	// ModuleAddress is the module portion of the above address. Omitted if the
	// instance is in the root module.
	ModuleAddress string `json:"module_address,omitempty"`

	// "managed" or "data"
	Mode string `json:"mode,omitempty"`

	Type         string          `json:"type,omitempty"`
	Name         string          `json:"name,omitempty"`
	Index        json.RawMessage `json:"index,omitempty"`
	ProviderName string          `json:"provider_name,omitempty"`

	// "deposed", if set, indicates that this action applies to a "deposed"
	// object of the given instance rather than to its "current" object. Omitted
	// for changes to the current object.
	Deposed string `json:"deposed,omitempty"`

	// Change describes the change that will be made to this object
	Change Change `json:"change,omitempty"`

	// ActionReason is a keyword representing some optional extra context
	// for why the actions in Change.Actions were chosen.
	//
	// This extra detail is only for display purposes, to help a UI layer
	// present some additional explanation to a human user. The possible
	// values here might grow and change over time, so any consumer of this
	// information should be resilient to encountering unrecognized values
	// and treat them as an unspecified reason.
	ActionReason string `json:"action_reason,omitempty"`
}

// Change is the representation of a proposed change for an object.
type Change struct {
	// Actions are the actions that will be taken on the object selected by the
	// properties below. Valid actions values are:
	//    ["no-op"]
	//    ["create"]
	//    ["read"]
	//    ["update"]
	//    ["delete", "create"]
	//    ["create", "delete"]
	//    ["delete"]
	// The two "replace" actions are represented in this way to allow callers to
	// e.g. just scan the list for "delete" to recognize all three situations
	// where the object will be deleted, allowing for any new deletion
	// combinations that might be added in future.
	Actions []string `json:"actions,omitempty"`

	// Before and After are representations of the object value both before and
	// after the action. For ["create"] and ["delete"] actions, either "before"
	// or "after" is unset (respectively). For ["no-op"], the before and after
	// values are identical. The "after" value will be incomplete if there are
	// values within it that won't be known until after apply.
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`

	// AfterUnknown is an object value with similar structure to After, but
	// with all unknown leaf values replaced with true, and all known leaf
	// values omitted.  This can be combined with After to reconstruct a full
	// value after the action, including values which will only be known after
	// apply.
	AfterUnknown json.RawMessage `json:"after_unknown,omitempty"`

	// BeforeSensitive and AfterSensitive are object values with similar
	// structure to Before and After, but with all sensitive leaf values
	// replaced with true, and all non-sensitive leaf values omitted. These
	// objects should be combined with Before and After to prevent accidental
	// display of sensitive values in user interfaces.
	BeforeSensitive json.RawMessage `json:"before_sensitive,omitempty"`
	AfterSensitive  json.RawMessage `json:"after_sensitive,omitempty"`

	// ReplacePaths is an array of arrays representing a set of paths into the
	// object value which resulted in the action being "replace". This will be
	// omitted if the action is not replace, or if no paths caused the
	// replacement (for example, if the resource was tainted). Each path
	// consists of one or more steps, each of which will be a number or a
	// string.
	ReplacePaths json.RawMessage `json:"replace_paths,omitempty"`

	// Importing contains the import metadata about this operation. If importing
	// is present (ie. not null) then the change is an import operation in
	// addition to anything mentioned in the actions field. The actual contents
	// of the Importing struct is subject to change, so downstream consumers
	// should treat any values in here as strictly optional.
	Importing *Importing `json:"importing,omitempty"`

	// GeneratedConfig contains any HCL config generated for this resource
	// during planning as a string.
	//
	// If this is populated, then Importing should also be populated but this
	// might change in the future. However, nNot all Importing changes will
	// contain generated config.
	GeneratedConfig string `json:"generated_config,omitempty"`
}

// Importing is a nested object for the resource import metadata.
type Importing struct {
	// The original ID of this resource used to target it as part of planned
	// import operation.
	ID string `json:"id,omitempty"`
}

// ResourceAttr contains the address and attribute of an external for the
// RelevantAttributes in the plan.
type ResourceAttr struct {
	Resource string          `json:"resource"`
	Attr     json.RawMessage `json:"attribute"`
}

// State is the top-level representation of the json format of a terraform
// state.
type State struct {
	FormatVersion    string          `json:"format_version,omitempty"`
	TerraformVersion string          `json:"terraform_version,omitempty"`
	Values           *StateValues    `json:"values,omitempty"`
	Checks           json.RawMessage `json:"checks,omitempty"`
}
