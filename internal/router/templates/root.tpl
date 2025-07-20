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
            {{ $name }} ({{ $item.GetType }}): {{ $item.GetValue }}
        </li>
        {{- end }}
    </ul>
</body>
</html>
