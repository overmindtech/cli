# <img width="24" alt="mapped" src="https://github.com/overmindtech/ovm-cli/assets/8799341/311e1bb0-e3da-4499-8db3-b97ec7674484"> Expected Changes

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


## <img width="20" alt="unmapped" src="https://github.com/overmindtech/ovm-cli/assets/8799341/46002c31-0c19-45c5-92ac-2a339e25b00e"> Unmapped Changes

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

| <img width="16" alt="mapped" src="https://github.com/overmindtech/ovm-cli/assets/8799341/311e1bb0-e3da-4499-8db3-b97ec7674484"> Items | <img width="16" alt="edge" src="https://github.com/overmindtech/ovm-cli/assets/8799341/437dcecd-117d-474d-a6fd-1aa241fb0fd0"> Edges |
|---|---|
| {{ .BlastItems }} | {{ .BlastEdges }}

[Open in Overmind]({{ .ChangeUrl }})



{{if .Risks }}
# <img width="24" alt="warning" src="https://github.com/overmindtech/ovm-cli/assets/8799341/fd3b183f-92b3-4aab-987d-40452f92bdbb"> Risks

{{range .Risks }}
## <img width="18" alt="{{ .SeverityAlt }}" src="{{ .SeverityIcon }}"> Impact on Target Groups [High]

The various target groups including \"944651592624.eu-west-2.elbv2-target-group.k8s-default-nats-4650f3a363\", \"944651592624.eu-west-2.elbv2-target-group.k8s-default-smartloo-fd2416d9f8\", etc., that work alongside the load balancer for traffic routing may be indirectly affected if the security group change causes networking issues. This is especially important if these target groups rely on different ports other than 8080 for health checks or for directing incoming requests to backend services.
{{end}}
{{end}}
