// +build darwin linux windows

package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"
)

type gameState struct {
	width    int
	height   int
	gl       gl.Context
	program  gl.Program
	bufData  gl.Buffer
	bufIndex gl.Buffer
	position gl.Attrib
}

func main() {
	log.Print("main begin")

	slowPaint := len(os.Args) > 1
	log.Printf("slowPaint: %v", slowPaint)

	game := &gameState{}
	var requests int
	var paints int
	sec := time.Now().Second()

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

				requests++ // events

				if now := time.Now().Second(); now != sec {
					log.Printf("requests=%d paints=%d", requests, paints)
					requests = 0
					paints = 0
					sec = now
				}

				//if !slowPaint || frames == 0 {
				paints++ // draws
				game.paint()
				a.Publish()
				//}

				if slowPaint {
					time.Sleep(250 * time.Millisecond)
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

	var err error
	game.program, err = glutil.CreateProgram(glc, vertexShader, fragmentShader)
	if err != nil {
		log.Printf("start: error creating GL program: %v", err)
		return
	}

	log.Printf("start: shader compiled")

	game.bufData = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufData)
	glc.BufferData(gl.ARRAY_BUFFER, squareData, gl.STATIC_DRAW)

	game.bufIndex = glc.CreateBuffer()
	glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufIndex)
	glc.BufferData(gl.ELEMENT_ARRAY_BUFFER, squareIndex, gl.STATIC_DRAW)

	game.position = glc.GetAttribLocation(game.program, "position")

	game.gl = glc
}

func (game *gameState) stop() {
	log.Printf("stop")

	glc := game.gl // shortcut

	glc.DeleteProgram(game.program)
	glc.DeleteBuffer(game.bufData)
	glc.DeleteBuffer(game.bufIndex)

	game.gl = nil
}

func (game *gameState) paint() {
	glc := game.gl // shortcut

	glc.ClearColor(.5, .5, .5, 1) // gray background
	glc.Clear(gl.COLOR_BUFFER_BIT)

	glc.UseProgram(game.program)
	glc.EnableVertexAttribArray(game.position)

	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufData)
	glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufIndex)

	strideSize := 5 * 4
	offset := 0
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, strideSize, offset)

	//glc.DrawArrays(gl.TRIANGLES, 0, vertexCount)
	elemFirst := 0
	elemCount := vertexCount
	elemType := gl.Enum(gl.UNSIGNED_INT) // 32-bit int = 4 bytes
	elemSize := 4                        // 32-bit int = 4 bytes
	glc.DrawElements(gl.TRIANGLES, elemCount, elemType, elemFirst*elemSize)

	glc.DisableVertexAttribArray(game.position)
}

const (
	coordsPerVertex = 3
	vertexCount     = 6
)

var squareData = f32.Bytes(binary.LittleEndian,
	// position    texture
	// ----------  --------
	0.0, 0.4, 0.0, 0.0, 0.4, // top left
	0.0, 0.0, 0.0, 0.0, 0.0, // bottom left
	0.4, 0.0, 0.0, 0.4, 0.0, // bottom right
	0.4, 0.4, 0.0, 0.4, 0.4, // top right
)

var squareIndex = intsToBytes([]uint32{
	0, 1, 2,
	2, 3, 0,
})

func intsToBytes(s []uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, s)
	b := buf.Bytes()
	log.Printf("intsToBytes: ints=%d bytes=%d: %v", len(s), len(b), b)
	return b
}

const vertexShader = `#version 100
attribute vec4 position;
void main() {
	gl_Position = position;
}`

const fragmentShader = `#version 100
precision mediump float;
void main() {
	gl_FragColor = vec4(0.8,0.8,0.8,1.0); // white
}`
