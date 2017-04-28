// +build darwin linux windows

package main

import (
	"log"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
)

func main() {
	log.Print("main begin")

	sec := time.Now()

	app.Main(func(a app.App) {
		log.Print("app.Main begin")

		sec = tick(sec, "app.Main")
		for e := range a.Events() {
			log.Printf("Some event: %v", e)
			switch t := a.Filter(e).(type) {
			case lifecycle.Event:
				log.Printf("Lifecycle: %v", t)
			case paint.Event:
				if t.External {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				log.Printf("Call OpenGL here: %v", t)
				a.Publish()

				// we request next paint event
				// in order to draw as fast as possible
				a.Send(paint.Event{})
			}
			sec = tick(sec, "range loop")
		}

		log.Print("app.Main end")
	})

	log.Print("main end")
}

func tick(t time.Time, s string) time.Time {
	n := time.Now()
	if n.Second() == t.Second() {
		return t
	}
	log.Printf("tick: %s", s)
	return n
}
