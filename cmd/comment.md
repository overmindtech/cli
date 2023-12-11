{{ $top := . -}}
# <img width="24" alt="mapped" src="{{ .AssetPath }}/item.svg"> Expected Changes

{{ range .ExpectedChanges -}}
<details>
<summary><img width="14" alt="{{ .StatusAlt }}" src="{{ $top.AssetPath }}/{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

{{ if .Diff -}}
```diff
{{ .Diff }}
```

{{ else -}}
(no changed attributes)
{{ end -}}

</details>
{{ else -}}
No expected changes found.
{{ end }}

{{ if .UnmappedChanges -}}
## <img width="20" alt="unmapped" src="{{ .AssetPath }}/unmapped.svg"> Unmapped Changes

> [!NOTE]
> These changes couldn't be mapped to a real cloud resource and therefore won't be included in the blast radius calculation.

{{ range .UnmappedChanges -}}

<details>
<summary><img width="14" alt="{{ .StatusAlt }}" src="{{ $top.AssetPath }}/{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

{{ if .Diff -}}

```diff
{{ .Diff }}
```

{{ else -}}
(no changed attributes)
{{ end -}}

</details>
{{ end }}
{{ end }}

# Blast Radius

| <img width="16" alt="items" src="{{ .AssetPath }}/item.svg"> Items | <img width="16" alt="edges" src="{{ .AssetPath }}/edge.svg"> Edges |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| {{ .BlastItems }}                                                                                                                                       | {{ .BlastEdges }}                                                                                                                                       |

[Open in Overmind]({{ .ChangeUrl }})

{{ if .Risks }}
# <img width="24" alt="warning" src="{{ .AssetPath }}/risks.svg"> Risks

{{ range .Risks }}
## <img width="18" alt="{{ .SeverityAlt }}" src="{{ $top.AssetPath }}/{{ .SeverityIcon }}"> {{ .Title }} [{{ .SeverityText }}]

{{ .Description }}
{{ end }}
{{ end }}
