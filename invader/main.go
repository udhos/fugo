// +build darwin linux windows

package main

import (
	"log"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
)

func main() {
	app.Main(func(a app.App) {
		for e := range a.Events() {
			log.Printf("Some event: %v", e)
			switch t := a.Filter(e).(type) {
			case lifecycle.Event:
				log.Printf("Lifecycle: %v", t)
			case paint.Event:
				log.Printf("Call OpenGL here: %v", t)
				a.Publish()
			}
		}
	})
}
