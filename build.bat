set GOROOT=D:\dev\sdk\go\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.4.windows-amd64
set GOPATH=D:\dev\dev\sdk\gopath
set CGO_ENABLED=0
set GOARCH=arm64
set GOOS=android
D:\dev\sdk\go\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.4.windows-amd64\bin\go.exe build -o D:\dev\project\chrelyonly-pgsty-minio\build\strawberry_minio.so D:\dev\project\chrelyonly-pgsty-minio\main.go
@REM set GOARCH=amd64
@REM set GOOS=linux
@REM D:\dev\sdk\go\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.4.windows-amd64\bin\go.exe build -o D:\dev\project\chrelyonly-pgsty-minio\build\frp-linux-amd64 D:\dev\project\chrelyonly-pgsty-minio\main.go
@REM set GOARCH=amd64
@REM set GOOS=darwin
@REM D:\dev\sdk\go\gopath\pkg\mod\golang.org\toolchain@v0.0.1-go1.26.4.windows-amd64\bin\go.exe build -o D:\dev\project\chrelyonly-pgsty-minio\build\frp-darwin-amd64 D:\dev\project\chrelyonly-pgsty-minio\main.go
@REM
@REM
@REM
@REM go build -trimpath  -ldflags="-s -w" -o build/strawberry_minio main.go