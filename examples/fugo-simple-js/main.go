package main

import (
	"fmt"
	"time"
	//"github.com/gopherjs/gopherjs/js"
	//"github.com/gopherjs/webgl"
	//"github.com/udhos/goglmath"
	//"github.com/udhos/gwob"
)

func main() {
	log("main: hi")
}

func log(msg string) {
	m := fmt.Sprintf("log: %s", msg)
	println(time.Now().String() + " " + m)
}
