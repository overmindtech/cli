<p align="center">
  <img alt="Overmind" src="{{ .AssetPath }}/logo.png" width="124px" align="center">
    <h3 align="center">
      <a href="{{ .ChangeUrl }}">Open in Overmind ↗  </a>
   </h3>
</p>

---

{{ if .TagsLine -}}
 {{ .TagsLine }}
{{ end -}}

<h3>🔥 Risks</h3>

{{ if not .Risks }}

> [!NOTE] > **Overmind has not identified any risks associated with this change**
> This could be due to the change being low risk with no impact on other parts of the system, or involving resources that Overmind currently does not support.

{{ else -}}
{{ range .Risks }}
**{{ .Title }}** `{{.SeverityText }}`  [Open Risk ↗]({{ .RiskUrl }})
{{ .Description }}

{{ end }}
{{ end -}}

---

<h3>🟣 Expected Changes</h3>

{{ range .ExpectedChanges -}}

<details>
<summary> {{ .StatusSymbol }} {{ .Type }} › {{ .Title }}</summary>

{{ if .Diff -}}

```diff
{{ .Diff }}
```

{{ else -}}
_No changed attributes_
{{ end -}}

</details>
{{ else -}}
> [!NOTE]
> **No expected changes found.**
{{ end }}

---

{{ if .UnmappedChanges -}}

<h3>🟠 Unmapped Changes</h3>

{{ range .UnmappedChanges -}}

<details>
<summary > {{ .StatusSymbol }} {{ .Type }} › {{ .Title }}</summary>

{{ if .Diff -}}

```diff
{{ .Diff }}
```

{{ else -}}
_No changed attributes_
{{ end -}}

</details>
{{ end }}
{{ end }}

---

<h3>💥 Blast Radius</h3>

**Items** ` {{ .BlastItems }} `

**Edges** ` {{ .BlastEdges }} `
