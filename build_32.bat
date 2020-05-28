set GOARCH=386
set GOOS=windows
go build -x -o gosvrk32.exe gosvrk.go  svlua.go sv3w_structs.go stat_structs.go
