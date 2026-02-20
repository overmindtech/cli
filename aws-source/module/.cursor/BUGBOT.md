# Terraform Module Review Rules

## HCL: IAM policy must stay read-only

If any changed `.tf` file modifies an IAM policy statement's `Action` list:

- Verify every action uses only read-only prefixes: `Get*`, `Describe*`, `List*`, `GetBucket*`, `ListAllMyBuckets`, `ListTagsForResource`, `GetMetricData`.
- Add a blocking Bug titled "IAM policy contains write actions" if any action allows mutation (e.g., `Put*`, `Create*`, `Delete*`, `Update*`, `Attach*`, `Detach*`).
- Body: "The Overmind IAM role must be strictly read-only. Write actions violate customer trust policies and the principle of least privilege. Remove the offending actions."

## Provider Go: Use diag.Diagnostics for errors

If any changed `.go` file in `provider/` returns an error from a resource or data source CRUD function using bare `fmt.Errorf` or `errors.New`:

- Add a warning titled "Use diag.Diagnostics instead of bare errors"
- Body: "Terraform provider resource and data source functions should return errors via `diag.Diagnostics` (e.g., `diag.FromErr(err)`) so that Terraform can display structured error output to users. See the [Terraform Plugin Framework documentation](https://developer.hashicorp.com/terraform/plugin/framework/diagnostics) for guidance."
