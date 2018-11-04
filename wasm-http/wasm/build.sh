#!/bin/bash

build () {
	local here=$(cd $(dirname $BASH_SOURCE) ; pwd)
	cd $here

	GOOS=js GOARCH=wasm go build -o test.wasm main_js_wasm.go
}

build
