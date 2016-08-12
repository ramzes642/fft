
<h2>Загрузить файл</h2>
<form method="post" enctype="multipart/form-data">
	<input type="file" name="f">
	<button>Загрузить файл</button>
</form>

<h2>Список файлов</h2>
{{range .}}<a href="/upload/{{.Name}}">{{.Name}}</a> <a href="/spec?f=upload/{{.Name}}">spec</a><br>{{end}}

