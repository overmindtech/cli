{{ $top := . -}}
# <img alt="banner" src="{{ .AssetPath }}/banner_dark.svg#gh-dark-mode-only"><img alt="banner" src="{{ .AssetPath }}/banner_light.svg#gh-light-mode-only">
{{ if .TagsLine -}}
{{ .TagsLine }}
{{ end -}}

# <img width="24" alt="warning" src="{{ .AssetPath }}/risks.svg"> Risks
{{ if not .Risks }}
Overmind has not identified any risks associated with this change.

This could be due to the change being low risk with no impact on other parts of the system, or involving resources that Overmind currently does not support.
{{ else -}}
{{ range .Risks }}
## <img width="16" alt="{{ .SeverityAlt }}" src="{{ $top.AssetPath }}/{{ .SeverityIcon }}"> {{ .Title }} [{{ .SeverityText }}]

{{ .Description }} [Open Risk]({{ .RiskUrl }})
{{ end }}
{{ end -}}

# <img width="24" alt="mapped" src="{{ .AssetPath }}/item.svg"> Expected Changes

{{ range .ExpectedChanges -}}
<details>
<summary><img width="16" alt="{{ .StatusAlt }}" src="{{ $top.AssetPath }}/{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

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
# <img width="24" alt="unmapped" src="{{ .AssetPath }}/unmapped.svg"> Unmapped Changes

> [!NOTE]
> These changes couldn't be mapped to a discoverable cloud resource and therefore won't be included in the blast radius calculation.

{{ range .UnmappedChanges -}}

<details>
<summary><img width="16" alt="{{ .StatusAlt }}" src="{{ $top.AssetPath }}/{{ .StatusIcon }}"> {{ .Type }} › {{ .Title }}</summary>

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

#  <img width="24" alt="edges" src="{{ .AssetPath }}/edge.svg"> Blast Radius

| <img width="16" alt="items" src="{{ .AssetPath }}/item.svg"> Items | <img width="14" alt="edges" src="{{ .AssetPath }}/edge.svg"> Edges |
| ------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| {{ .BlastItems }}                                                                                                                                       | {{ .BlastEdges }}                                                                                                                                       |

[Open Blast Radius]({{ .BlastRadiusUrl }})

[<img alt="Open in Overmind" src="{{ .AssetPath }}/open_in_overmind.svg">]({{ .ChangeUrl }})
