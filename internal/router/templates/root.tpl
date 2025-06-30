<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Metrics</title>
</head>
<body>
    <h1>Metrics</h1>
    <ul>
        {{- range $name, $item := . }}
        <li>
            {{ $name }} ({{ $item.MType }}):
            {{- if $item.Value }}{{ $item.Value }}{{- end }}
            {{- if $item.Delta }}{{ $item.Delta }}{{- end }}
        </li>
        {{- end }}
    </ul>
</body>
</html>
