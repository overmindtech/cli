package proc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	resourcemanagerpb "cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/overmindtech/cli/discovery"
	"github.com/overmindtech/cli/sdp-go"
	"github.com/overmindtech/cli/sources/gcp/dynamic"
	_ "github.com/overmindtech/cli/sources/gcp/dynamic/adapters" // Import all adapters to register them
	"github.com/overmindtech/cli/sources/gcp/manual"
	gcpshared "github.com/overmindtech/cli/sources/gcp/shared"
)

// Metadata contains the metadata for the GCP source
var Metadata = sdp.AdapterMetadataList{}

// GCPConfig holds configuration for GCP source
type GCPConfig struct {
	Parent    string // Optional: Can be organization, folder, or project. If empty, will discover all accessible projects
	ProjectID string // Deprecated: Use Parent instead. Optional: If empty, will discover all accessible projects
	Regions   []string
	Zones     []string

	ImpersonationServiceAccountEmail string // leave empty for direct access using Application Default Credentials
}

// ParentType represents the type of GCP parent resource
type ParentType int

const (
	ParentTypeUnknown ParentType = iota
	ParentTypeOrganization
	ParentTypeFolder
	ParentTypeProject
)

// detectParentType determines the type of parent resource based on its format
func detectParentType(parent string) (ParentType, error) {
	if parent == "" {
		return ParentTypeUnknown, fmt.Errorf("parent is empty")
	}

	// Check for organization format
	if len(parent) >= len("organizations/") && parent[:len("organizations/")] == "organizations/" {
		return ParentTypeOrganization, nil
	}

	// Check for folder format
	if len(parent) >= len("folders/") && parent[:len("folders/")] == "folders/" {
		return ParentTypeFolder, nil
	}

	// Check for explicit project format
	if len(parent) >= len("projects/") && parent[:len("projects/")] == "projects/" {
		return ParentTypeProject, nil
	}

	// If none of the above, assume it's a project ID
	// GCP project IDs must:
	// - Start with a lowercase letter
	// - Contain only lowercase letters, digits, and hyphens
	// - Be between 6 and 30 characters
	// This is a simplified check - we'll let the API validate the actual format
	if len(parent) >= 6 && len(parent) <= 30 {
		return ParentTypeProject, nil
	}

	return ParentTypeUnknown, fmt.Errorf("unable to determine parent type from: %s. Expected formats: 'organizations/{org_id}', 'folders/{folder_id}', or project ID", parent)
}

// normalizeParent converts a parent string to its canonical format
// For projects, it converts "projects/{project_id}" to just the project ID
// For organizations and folders, it ensures the format is correct
func normalizeParent(parent string, parentType ParentType) (string, error) {
	switch parentType {
	case ParentTypeOrganization:
		// Organizations should be in format "organizations/{org_id}"
		// Validate that there's an ID after the prefix
		prefix := "organizations/"
		if !strings.HasPrefix(parent, prefix) || len(parent) <= len(prefix) {
			return "", fmt.Errorf("invalid organization format: %s. Expected 'organizations/{org_id}'", parent)
		}
		return parent, nil
	case ParentTypeFolder:
		// Folders should be in format "folders/{folder_id}"
		// Validate that there's an ID after the prefix
		prefix := "folders/"
		if !strings.HasPrefix(parent, prefix) || len(parent) <= len(prefix) {
			return "", fmt.Errorf("invalid folder format: %s. Expected 'folders/{folder_id}'", parent)
		}
		return parent, nil
	case ParentTypeProject:
		// Extract project ID from "projects/{project_id}" format if present
		var projectID string
		if strings.HasPrefix(parent, "projects/") {
			projectID = parent[len("projects/"):]
		} else {
			projectID = parent
		}
		// Validate that the project ID is not empty
		if projectID == "" {
			return "", fmt.Errorf("invalid project format: %s. Expected 'projects/{project_id}' or a valid project ID", parent)
		}
		return projectID, nil
	case ParentTypeUnknown:
		return "", fmt.Errorf("unknown parent type")
	default:
		return "", fmt.Errorf("unknown parent type")
	}
}

func init() {
	// Register the GCP source metadata for documentation purposes
	ctx := context.Background()

	// project, regions, and zones are just placeholders here
	// They are not used in the metadata content
	discoveryAdapters, err := adapters(
		ctx,
		"project",
		[]string{"region"},
		[]string{"zone"},
		"",
		nil,
		false,
	)
	if err != nil {
		panic(fmt.Errorf("error creating adapters: %w", err))
	}

	for _, adapter := range discoveryAdapters {
		Metadata.Register(adapter.Metadata())
	}

	log.Debug("Registered GCP source metadata", " with ", len(Metadata.AllAdapterMetadata()), " adapters")
}

func Initialize(ctx context.Context, ec *discovery.EngineConfig, cfg *GCPConfig) (*discovery.Engine, error) {
	engine, err := discovery.NewEngine(ec)
	if err != nil {
		return nil, fmt.Errorf("error initializing Engine: %w", err)
	}

	var permissionCheck func() error

	var startupErrorMutex sync.Mutex
	startupError := errors.New("source is starting")
	if ec.HeartbeatOptions == nil {
		ec.HeartbeatOptions = &discovery.HeartbeatOptions{}
	}
	ec.HeartbeatOptions.HealthCheck = func(_ context.Context) error {
		startupErrorMutex.Lock()
		defer startupErrorMutex.Unlock()
		if startupError != nil {
			// If there is a startup error, return it
			return startupError
		}

		if permissionCheck != nil {
			// If the permission check is set, run it
			return permissionCheck()
		}
		return nil
	}

	engine.StartSendingHeartbeats(ctx)

	err = func() error {
		var logmsg string
		// Use provided config, otherwise fall back to viper
		if cfg != nil {
			logmsg = "Using directly provided config"
		} else {
			var err error
			cfg, err = readConfig()
			if err != nil {
				return fmt.Errorf("error creating config from command line: %w", err)
			}
			logmsg = "Using config from viper"

		}

		// Determine which projects to use based on the parent configuration
		var projectIDs []string
		if cfg.Parent == "" {
			// No parent specified - discover all accessible projects
			log.WithFields(log.Fields{
				"ovm.source.type": "gcp",
			}).Info("No parent specified, discovering all accessible projects")

			discoveredProjects, err := discoverProjects(ctx, cfg.ImpersonationServiceAccountEmail)
			if err != nil {
				return fmt.Errorf("error discovering projects: %w", err)
			}

			projectIDs = discoveredProjects
		} else {
			// Parent is specified - determine its type and discover accordingly
			parentType, err := detectParentType(cfg.Parent)
			if err != nil {
				return fmt.Errorf("error detecting parent type: %w", err)
			}

			normalizedParent, err := normalizeParent(cfg.Parent, parentType)
			if err != nil {
				return fmt.Errorf("error normalizing parent: %w", err)
			}

			switch parentType {
			case ParentTypeProject:
				// Single project - no discovery needed
				log.WithFields(log.Fields{
					"ovm.source.type":       "gcp",
					"ovm.source.parent":     cfg.Parent,
					"ovm.source.project_id": normalizedParent,
				}).Info("Using specified project")
				projectIDs = []string{normalizedParent}

			case ParentTypeOrganization, ParentTypeFolder:
				// Organization or folder - discover all projects within it
				log.WithFields(log.Fields{
					"ovm.source.type":   "gcp",
					"ovm.source.parent": cfg.Parent,
					"parent_type":       parentType,
				}).Info("Discovering projects under parent")

				discoveredProjects, err := discoverProjectsUnderSpecificParent(ctx, cfg.Parent, cfg.ImpersonationServiceAccountEmail)
				if err != nil {
					return fmt.Errorf("error discovering projects under parent %s: %w", cfg.Parent, err)
				}

				if len(discoveredProjects) == 0 {
					return fmt.Errorf("no accessible projects found under parent %s. Please ensure the service account has the 'resourcemanager.projects.list' permission via the 'roles/browser' predefined GCP role", cfg.Parent)
				}

				projectIDs = discoveredProjects

			case ParentTypeUnknown:
				return fmt.Errorf("unknown parent type for parent: %s", cfg.Parent)

			default:
				return fmt.Errorf("unknown parent type for parent: %s", cfg.Parent)
			}
		}

		logFields := log.Fields{
			"ovm.source.type":                                "gcp",
			"ovm.source.project_count":                       len(projectIDs),
			"ovm.source.regions":                             cfg.Regions,
			"ovm.source.zones":                               cfg.Zones,
			"ovm.source.impersonation-service-account-email": cfg.ImpersonationServiceAccountEmail,
		}
		if cfg.Parent != "" {
			logFields["ovm.source.parent"] = cfg.Parent
		}
		if cfg.ProjectID != "" {
			logFields["ovm.source.project_id"] = cfg.ProjectID
		}
		log.WithFields(logFields).Info(logmsg)

		// If still no regions/zones this is no valid config.
		if len(cfg.Regions) == 0 && len(cfg.Zones) == 0 {
			return fmt.Errorf("GCP source must specify at least one region or zone")
		}

		linker := gcpshared.NewLinker()

		// Create adapters for all projects
		var allAdapters []discovery.Adapter
		cloudResourceManagerProjectAdapters := make(map[string]discovery.Adapter)

		for _, projectID := range projectIDs {
			log.WithFields(log.Fields{
				"ovm.source.type":       "gcp",
				"ovm.source.project_id": projectID,
			}).Debug("Creating adapters for project")

			discoveryAdapters, err := adapters(ctx, projectID, cfg.Regions, cfg.Zones, cfg.ImpersonationServiceAccountEmail, linker, true)
			if err != nil {
				return fmt.Errorf("error creating discovery adapters for project %s: %w", projectID, err)
			}

			allAdapters = append(allAdapters, discoveryAdapters...)

			// Collect cloud resource manager project adapters for health checks
			for _, adapter := range discoveryAdapters {
				if adapter.Type() == gcpshared.CloudResourceManagerProject.String() {
					cloudResourceManagerProjectAdapters[projectID] = adapter
				}
			}
		}

		if len(cloudResourceManagerProjectAdapters) == 0 {
			return fmt.Errorf("cloud resource manager project adapter not found")
		}

		// Verify we have an adapter for each project
		for _, projectID := range projectIDs {
			if _, exists := cloudResourceManagerProjectAdapters[projectID]; !exists {
				return fmt.Errorf("cloud resource manager project adapter not found for project %s", projectID)
			}
		}

		permissionCheck = func() error {
			// Check permissions for all projects
			for _, projectID := range projectIDs {
				adapter := cloudResourceManagerProjectAdapters[projectID]
				// Get the project from the cloud resource manager
				// Giving this permission is mandatory for the GCP source health check
				prj, err := adapter.Get(ctx, projectID, projectID, false)
				if err != nil {
					// Check if this is a permission error and provide a simplified message
					var permissionError *dynamic.PermissionError
					if errors.As(err, &permissionError) {
						return fmt.Errorf("insufficient permissions to access GCP project '%s'. "+
							"Please ensure the service account has the 'resourcemanager.projects.get' permission via the 'roles/browser' predefined GCP role", projectID)
					}
					return fmt.Errorf("error accessing project %s: %w", projectID, err)
				}

				if prj == nil {
					return fmt.Errorf("project %s not found in cloud resource manager", projectID)
				}

				prjID, err := prj.GetAttributes().Get("projectId")
				if err != nil {
					return fmt.Errorf("error getting project ID from project %s: %w", projectID, err)
				}

				prjIDStr, ok := prjID.(string)
				if !ok {
					return fmt.Errorf("project ID is not a string for project %s: %v", projectID, prjID)
				}

				if prjIDStr != projectID {
					return fmt.Errorf("project ID mismatch for project %s: expected %s, got %s", projectID, projectID, prjIDStr)
				}
			}

			return nil
		}

		err = permissionCheck()
		if err != nil {
			return fmt.Errorf("error checking permissions: %w", err)
		}

		// Add the adapters to the engine
		err = engine.AddAdapters(allAdapters...)
		if err != nil {
			return fmt.Errorf("error adding adapters to engine: %w", err)
		}

		return nil
	}()

	startupErrorMutex.Lock()
	startupError = err
	startupErrorMutex.Unlock()
	brokenHeart := engine.SendHeartbeat(ctx, nil) // Send the error immediately through the custom health check func
	if brokenHeart != nil {
		log.WithError(brokenHeart).Error("Error sending heartbeat")
	}

	if err != nil {
		log.WithError(err).Debug("Error initializing GCP source")

		return nil, fmt.Errorf("error initializing GCP source: %w", err)
	}

	log.Debug("Sources initialized")
	// If there is no error then return the engine
	return engine, nil
}

func readConfig() (*GCPConfig, error) {
	parent := viper.GetString("gcp-parent")
	projectID := viper.GetString("gcp-project-id")

	// Handle backwards compatibility
	// If both are specified, parent takes precedence (with a warning)
	// If only project-id is specified, convert it to parent format for internal use
	if parent != "" && projectID != "" {
		log.WithFields(log.Fields{
			"ovm.source.type": "gcp",
		}).Warn("Both --gcp-parent and --gcp-project-id are specified. Using --gcp-parent. Note: --gcp-project-id is deprecated, please use --gcp-parent instead.")
	} else if projectID != "" {
		log.WithFields(log.Fields{
			"ovm.source.type": "gcp",
		}).Warn("Using deprecated --gcp-project-id flag. Please use --gcp-parent instead for future compatibility.")
		// Convert project ID to parent format for internal consistency
		parent = projectID
	}

	l := &GCPConfig{
		Parent:                           parent,
		ProjectID:                        projectID, // Keep for backwards compatibility in logging/debugging
		ImpersonationServiceAccountEmail: viper.GetString("gcp-impersonation-service-account-email"),
	}

	// TODO: In the future, we will try to get the zones via Search API
	// https://github.com/overmindtech/workspace/issues/1340

	zones := viper.GetStringSlice("gcp-zones")
	regions := viper.GetStringSlice("gcp-regions")
	if len(zones) == 0 && len(regions) == 0 {
		return nil, fmt.Errorf("need at least one gcp-zones or gcp-regions value")
	}

	uniqueRegions := make(map[string]bool)
	for _, region := range regions {
		uniqueRegions[region] = true
	}

	for _, zone := range zones {
		if zone == "" {
			return nil, fmt.Errorf("zone name is empty")
		}

		l.Zones = append(l.Zones, zone)

		region := gcpshared.ZoneToRegion(zone)
		if region == "" {
			return nil, fmt.Errorf("zone %s is not valid", zone)
		}

		uniqueRegions[region] = true
	}

	for region := range uniqueRegions {
		l.Regions = append(l.Regions, region)
	}

	return l, nil
}

// discoverProjects uses the Cloud Resource Manager API to discover all projects accessible to the service account
// Requires the resourcemanager.projects.list permission (included in roles/browser)
// It recursively traverses the organization/folder hierarchy since the API only returns direct children
func discoverProjects(ctx context.Context, impersonationServiceAccountEmail string) ([]string, error) {
	// Create client options
	var clientOpts []option.ClientOption
	if impersonationServiceAccountEmail != "" {
		// Use impersonation credentials
		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: impersonationServiceAccountEmail,
			Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create impersonated token source: %w", err)
		}
		clientOpts = append(clientOpts, option.WithTokenSource(ts))
	}

	// Create clients for organizations, folders, and projects
	orgsClient, err := resourcemanager.NewOrganizationsClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create organizations client: %w", err)
	}
	defer orgsClient.Close()

	foldersClient, err := resourcemanager.NewFoldersClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create folders client: %w", err)
	}
	defer foldersClient.Close()

	projectsClient, err := resourcemanager.NewProjectsClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create projects client: %w", err)
	}
	defer projectsClient.Close()

	// Use a map to track discovered projects and avoid duplicates
	projectSet := make(map[string]bool)

	// Search for organizations (no parent needed)
	var organizationParents []string
	orgIt := orgsClient.SearchOrganizations(ctx, &resourcemanagerpb.SearchOrganizationsRequest{})
	for {
		org, err := orgIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			// Not all accounts have organizations (e.g., personal accounts), so this is not fatal
			log.WithError(err).Debug("Error searching organizations, continuing without org-based discovery")
			break
		}
		organizationParents = append(organizationParents, org.GetName())
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type": "gcp",
			"organization":    org.GetName(),
		}).Debug("Discovered organization")
	}

	// Recursively discover projects under each organization
	for _, orgParent := range organizationParents {
		if err := discoverProjectsUnderParent(ctx, orgParent, projectsClient, foldersClient, projectSet); err != nil {
			log.WithContext(ctx).WithError(err).WithField("parent", orgParent).Debug("Error discovering projects under organization, continuing")
		}
	}

	// Convert map to slice
	var projects []string
	for projectID := range projectSet {
		projects = append(projects, projectID)
	}

	if len(projects) == 0 {
		if len(organizationParents) == 0 {
			return nil, fmt.Errorf("no accessible projects found. If you're using a personal account without an organization, please specify --gcp-project-id explicitly")
		}
		return nil, fmt.Errorf("no accessible projects found. Please ensure the service account has the 'resourcemanager.projects.list' permission via the 'roles/browser' predefined GCP role")
	}

	log.WithContext(ctx).WithFields(log.Fields{
		"ovm.source.type":          "gcp",
		"ovm.source.project_count": len(projects),
	}).Info("Successfully discovered projects")

	return projects, nil
}

// discoverProjectsUnderParent recursively discovers all projects under a given parent (organization or folder)
// It lists direct child projects and folders, then recursively processes each folder
func discoverProjectsUnderParent(
	ctx context.Context,
	parent string,
	projectsClient *resourcemanager.ProjectsClient,
	foldersClient *resourcemanager.FoldersClient,
	projectSet map[string]bool,
) error {
	// List direct projects under this parent
	projectIt := projectsClient.ListProjects(ctx, &resourcemanagerpb.ListProjectsRequest{
		Parent: parent,
	})
	for {
		project, err := projectIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			// Log but continue - permission errors on individual parents shouldn't stop discovery
			log.WithContext(ctx).WithError(err).WithField("parent", parent).Debug("Error listing projects under parent, continuing")
			break
		}

		// Only include active projects
		if project.GetState() == resourcemanagerpb.Project_ACTIVE && project.GetProjectId() != "" {
			projectID := project.GetProjectId()
			if !projectSet[projectID] {
				projectSet[projectID] = true
				log.WithContext(ctx).WithFields(log.Fields{
					"ovm.source.type":         "gcp",
					"ovm.source.project_id":   projectID,
					"ovm.source.display_name": project.GetDisplayName(),
					"parent":                  parent,
				}).Debug("Discovered project")
			}
		}
	}

	// List direct folders under this parent
	folderIt := foldersClient.ListFolders(ctx, &resourcemanagerpb.ListFoldersRequest{
		Parent: parent,
	})
	for {
		folder, err := folderIt.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			// Log but continue - permission errors on individual folders shouldn't stop discovery
			log.WithContext(ctx).WithError(err).WithField("parent", parent).Debug("Error listing folders under parent, continuing")
			break
		}

		folderName := folder.GetName()
		log.WithFields(log.Fields{
			"ovm.source.type": "gcp",
			"folder":          folderName,
			"parent":          parent,
		}).Debug("Discovered folder")

		// Recursively discover projects under this folder
		if err := discoverProjectsUnderParent(ctx, folderName, projectsClient, foldersClient, projectSet); err != nil {
			log.WithContext(ctx).WithError(err).WithField("parent", folderName).Debug("Error discovering projects under folder, continuing")
		}
	}

	return nil
}

// discoverProjectsUnderSpecificParent discovers all projects under a specific parent (organization or folder)
// This is similar to discoverProjects but starts from a specific parent instead of searching for all organizations
func discoverProjectsUnderSpecificParent(ctx context.Context, parent string, impersonationServiceAccountEmail string) ([]string, error) {
	// Create client options
	var clientOpts []option.ClientOption
	if impersonationServiceAccountEmail != "" {
		// Use impersonation credentials
		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: impersonationServiceAccountEmail,
			Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create impersonated token source: %w", err)
		}
		clientOpts = append(clientOpts, option.WithTokenSource(ts))
	}

	// Create clients for folders and projects
	foldersClient, err := resourcemanager.NewFoldersClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create folders client: %w", err)
	}
	defer foldersClient.Close()

	projectsClient, err := resourcemanager.NewProjectsClient(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create projects client: %w", err)
	}
	defer projectsClient.Close()

	// Use a map to track discovered projects and avoid duplicates
	projectSet := make(map[string]bool)

	// Recursively discover projects under the specified parent
	if err := discoverProjectsUnderParent(ctx, parent, projectsClient, foldersClient, projectSet); err != nil {
		return nil, fmt.Errorf("error discovering projects under parent %s: %w", parent, err)
	}

	// Convert map to slice
	var projects []string
	for projectID := range projectSet {
		projects = append(projects, projectID)
	}

	// Return the list even if empty - let the caller handle the empty case
	// with a more informative error message
	if len(projects) > 0 {
		log.WithContext(ctx).WithFields(log.Fields{
			"ovm.source.type":          "gcp",
			"ovm.source.parent":        parent,
			"ovm.source.project_count": len(projects),
		}).Info("Successfully discovered projects under parent")
	}

	return projects, nil
}

// adapters returns a list of discovery adapters for GCP
// It includes both manual adapters and dynamic adapters.
func adapters(
	ctx context.Context,
	projectID string,
	regions []string,
	zones []string,
	impersonationServiceAccountEmail string,
	linker *gcpshared.Linker,
	initGCPClients bool,
) ([]discovery.Adapter, error) {
	discoveryAdapters := make([]discovery.Adapter, 0)

	var tokenSource *oauth2.TokenSource
	if impersonationServiceAccountEmail != "" {
		// Base credentials sourced from ADC
		ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: impersonationServiceAccountEmail,
			// Broad access to all GCP resources
			// It is restricted by the IAM permissions of the service account
			Scopes: []string{"https://www.googleapis.com/auth/cloud-platform"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create token source: %w", err)
		}
		tokenSource = &ts
	}

	// Add manual adapters
	manualAdapters, err := manual.Adapters(
		ctx,
		projectID,
		regions,
		zones,
		tokenSource,
		initGCPClients,
	)
	if err != nil {
		return nil, err
	}

	initiatedManualAdapters := make(map[string]bool)
	for _, adapter := range manualAdapters {
		initiatedManualAdapters[adapter.Type()] = true
	}

	discoveryAdapters = append(discoveryAdapters, manualAdapters...)

	httpClient := http.DefaultClient
	if initGCPClients {
		var errCli error
		httpClient, errCli = gcpshared.GCPHTTPClientWithOtel(ctx, impersonationServiceAccountEmail)
		if errCli != nil {
			return nil, fmt.Errorf("error creating GCP HTTP client: %w", errCli)
		}
	}

	// Add dynamic adapters
	dynamicAdapters, err := dynamic.Adapters(
		projectID,
		regions,
		zones,
		linker,
		httpClient,
		initiatedManualAdapters,
	)
	if err != nil {
		return nil, err
	}

	discoveryAdapters = append(discoveryAdapters, dynamicAdapters...)

	return discoveryAdapters, nil
}
