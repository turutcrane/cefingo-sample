// +build js,wasm

package main

import "syscall/js"

func main() {
	global := js.Global()
	my := global.Get("my")
	window := global.Get("window")
	window.Call("alert", "Hello, " + my.Get("you").String() + "!")
}
