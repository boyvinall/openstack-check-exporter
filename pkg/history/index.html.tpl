<html>
<head>
    <title>Openstack Check Exporter</title>
    <style>
        table {
            border-collapse: collapse;
        }
        table, th, td {
            border: 1px solid black;
            padding: 2px 10px;
        }
        th {
            background-color: #33e;
            color: white;
        }
        body {
            font-family: "Helvetica Neue", Helvetica, Arial, sans-serif;
            font-size: 14px;
            line-height: 20px;
            font-weight: 400;
            color: #3b3b3b;
        }
        .duration {
            display: inline-block;
            background-color: #bbf;
            max-width: 300px;
        }
    </style>
</head>
<body>
<h1>Openstack Check Exporter</h1>
<p>
<a href="/metrics">Metrics</a><br>
<a href="?">Show all checks</a>
</p>
<table>
<tr>
    <th>Completed</th>
    <th>Cloud</th>
    <th>Name</th>
    <th colspan="2">Duration</th>
    <th>Error</th>
    <th>Detail</th>
</tr>
{{range .}}
    <tr>
        <td>{{(.Start.Add .Duration).UTC.Format "2006-01-02T15:04:05Z07:00"}}</td>
        <td>{{.Cloud}}</td>
        <td><a href="?name={{.Name}}">{{.Name}}</a></td>
        <td>{{duration .Duration}}</td>
        <td><div class="duration" style="width:{{width .Duration}}px">&nbsp;</div></td>
        <td>{{.Error}}</td>
        <td><a href="/detail/{{.ID}}">{{.ID}}</a></td>
    </tr>
{{end}}
</table>
</body>
</html>