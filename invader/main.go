// +build darwin linux windows

package main

import (
	"log"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"
)

type gameState struct {
	width  int
	height int
	gl     gl.Context
}

func main() {
	log.Print("main begin")

	game := &gameState{}
	var frames int
	var paints int
	sec := time.Now().Second()
	slowPaint := true

	app.Main(func(a app.App) {
		log.Print("app.Main begin")

	LOOP:
		for e := range a.Events() {
			switch t := a.Filter(e).(type) {
			case lifecycle.Event:
				log.Printf("Lifecycle: %v", t)

				if t.From > t.To && t.To == lifecycle.StageDead {
					log.Printf("lifecycle down to dead")
					break LOOP
				}

				if t.Crosses(lifecycle.StageAlive) == lifecycle.CrossOff {
					log.Printf("lifecycle cross down alive")
					break LOOP
				}

				switch t.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					glc, isGL := t.DrawContext.(gl.Context)
					if !isGL {
						log.Printf("Lifecycle: visible: bad GL context")
						continue LOOP
					}
					game.start(glc)
					a.Send(paint.Event{}) // start drawing
				case lifecycle.CrossOff:
					game.stop()
				}

			case paint.Event:
				if t.External || game.gl == nil {
					// As we are actively painting as fast as
					// we can (usually 60 FPS), skip any paint
					// events sent by the system.
					continue
				}

				paints++ // events

				if now := time.Now().Second(); now != sec {
					log.Printf("fps: %d, paints: %d", frames, paints)
					frames = 0
					paints = 0
					sec = now
				}

				if !slowPaint || frames == 0 {
					frames++ // draws
					game.paint()
					a.Publish()
				}

				if slowPaint {
					time.Sleep(500 * time.Millisecond)
				}

				// we request next paint event
				// in order to draw as fast as possible
				a.Send(paint.Event{})
			case mouse.Event:
				game.input(t.X, t.Y)
			case touch.Event:
				game.input(t.X, t.Y)
			case size.Event:
				game.resize(t.WidthPx, t.HeightPx)
			}
		}

		log.Print("app.Main end")
	})

	log.Print("main end")
}

func (game *gameState) resize(w, h int) {
	if game.width != w || game.height != h {
		log.Printf("resize: %d,%d", w, h)
	}
	game.width = w
	game.height = h
}

func (game *gameState) input(x, y float32) {
	log.Printf("input: %f,%f (%d x %d)", x, y, game.width, game.height)
}

func (game *gameState) start(glc gl.Context) {
	log.Printf("start")
	game.gl = glc
}

func (game *gameState) stop() {
	log.Printf("stop")
	game.gl = nil
}

func (game *gameState) paint() {
	//log.Printf("paint: call OpenGL here")
}
