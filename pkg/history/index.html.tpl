<html>
<head>
    <title>Test</title>
    <style>
        table {
            border-collapse: collapse;
        }
        table, th, td {
            border: 1px solid black;
            padding: 2px 10px;
        }
        th {
            background-color: #4CAF50;
            color: white;
        }
        body {
            font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
            font-size: 14px;
            line-height: 20px;
            font-weight: 400;
            color: #3b3b3b;
        }
    </style>
</head>
<body>
<table>
<tr>
    <th>Start</th>
    <th>Cloud</th>
    <th>Name</th>
    <th>Duration</th>
    <th>Error</th>
    <th>Detail</th>
</tr>
{{range .}}
    <tr>
        <td>{{.Start.UTC.Format "2006-01-02T15:04:05Z07:00"}}</td>
        <td>{{.Cloud}}</td>
        <td>{{.Name}}</td>
        <td>{{.Duration}}</td>
        <td>{{.Error}}</td>
        <td><a href="/detail/{{.ID}}">{{.ID}}</a></td>
    </tr>
{{end}}
</table>
</body>
</html>