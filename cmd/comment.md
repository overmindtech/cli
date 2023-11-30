# <img width="24" alt="mapped" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/31cf83925e9db51a1b9296389615dadd66cdb7ebassets/item.svg"> Expected Changes

{{range .ExpectedChanges }}

<details>
<summary><img width="14" alt="{{ .StatusAlt }}" src="{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

{{if .Diff }}

```diff
{{ .Diff }}
```

{{else}}
(no changed attributes)
{{end}}

</details>
{{else}}
No expected changes found.
{{end}}

## <img width="20" alt="unmapped" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/31cf83925e9db51a1b9296389615dadd66cdb7ebassets/unmapped.svg"> Unmapped Changes

> [!NOTE]
> These changes couldn't be mapped to a real cloud resource and therefore won't be included in the blast radius calculation.

{{range .UnmappedChanges }}

<details>
<summary><img width="14" alt="{{ .StatusAlt }}" src="{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

{{if .Diff }}

```diff
{{ .Diff }}
```

{{else}}
(no changed attributes)
{{end}}

</details>
{{else}}
No unmapped changes found.
{{end}}

# Blast Radius

| <img width="16" alt="items" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/31cf83925e9db51a1b9296389615dadd66cdb7ebassets/item.svg"> Items | <img width="16" alt="edges" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/31cf83925e9db51a1b9296389615dadd66cdb7ebassets/edge.svg"> Edges |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| {{ .BlastItems }}                                                                                                                                       | {{ .BlastEdges }}                                                                                                                                       |

[Open in Overmind]({{ .ChangeUrl }})

{{if .Risks }}

# <img width="24" alt="warning" src="https://raw.githubusercontent.com/overmindtech/ovm-cli/31cf83925e9db51a1b9296389615dadd66cdb7ebassets/risks.svg"> Risks

{{range .Risks }}

## <img width="18" alt="{{ .SeverityAlt }}" src="{{ .SeverityIcon }}"> Impact on Target Groups [High]

The various target groups including \"944651592624.eu-west-2.elbv2-target-group.k8s-default-nats-4650f3a363\", \"944651592624.eu-west-2.elbv2-target-group.k8s-default-smartloo-fd2416d9f8\", etc., that work alongside the load balancer for traffic routing may be indirectly affected if the security group change causes networking issues. This is especially important if these target groups rely on different ports other than 8080 for health checks or for directing incoming requests to backend services.
{{end}}
{{end}}
