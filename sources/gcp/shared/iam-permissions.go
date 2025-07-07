package shared

// IAMPermissions is a map of IAM permissions in GCP that are required for the Overmind GCP Source to function properly.
// This map is populated during GCP Source initialization.
// It will be used for creating a custom role or defining the predefined roles in Google Cloud Platform.
// https://linear.app/overmind/issue/ENG-611/create-a-specific-permissions-list-for-gcp-source#comment-2217b610
var IAMPermissions = map[string]bool{}
