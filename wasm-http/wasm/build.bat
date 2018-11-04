set GOOS=js
set GOARCH=wasm
go build -o test.wasm main_js_wasm.go
set GOOS=
set GOARCH=