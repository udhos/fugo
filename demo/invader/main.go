// +build darwin linux windows

package main

import (
	"encoding/binary"
	"encoding/gob"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/udhos/goglmath"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/asset"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/mouse"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/exp/f32"
	"golang.org/x/mobile/exp/gl/glutil"
	"golang.org/x/mobile/gl"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/msg"
)

type gameState struct {
	width   int
	height  int
	gl      gl.Context
	program gl.Program
	//bufTriangle    gl.Buffer
	bufSquare              gl.Buffer
	bufSquareWire          gl.Buffer
	bufCannon              gl.Buffer
	bufCannonDown          gl.Buffer
	position               gl.Attrib
	P                      gl.Uniform // projection mat4 uniform
	color                  gl.Uniform
	minX, maxX, minY, maxY float64
	//proj                   goglmath.Matrix4
	shaderVert             string
	shaderFrag             string
	serverAddr             string
	serverOutput           chan msg.Fire
	playerFuel             float32
	playerTeam             int
	updateInterval         time.Duration
	updateLast             time.Time
	missiles               []*msg.Missile
	cannons                []*msg.Cannon
}

func newGame() (*gameState, error) {
	game := &gameState{}

	vert, errVert := loadFull("shader.vert")
	if errVert != nil {
		log.Printf("load vertex shader: %v", errVert)
		return nil, errVert
	}
	game.shaderVert = string(vert)

	frag, errFrag := loadFull("shader.frag")
	if errFrag != nil {
		log.Printf("load fragment shader: %v", errFrag)
		return nil, errFrag
	}
	game.shaderFrag = string(frag)

	server, errServ := loadFull("server.txt")
	if errServ != nil {
		log.Printf("load server: %v", errServ)
		return nil, errServ
	}
	game.serverAddr = strings.TrimSpace(string(server))

	log.Printf("server: [%s]", game.serverAddr)

	game.updateInterval = time.Second
	game.updateLast = time.Now()

	game.serverOutput = make(chan msg.Fire)

	return game, nil
}

func main() {
	log.Print("main begin")

	slowPaint := len(os.Args) > 1
	log.Printf("slowPaint: %v", slowPaint)

	var paintRequests int
	var paints int
	sec := time.Now().Second()
	game, errGame := newGame()
	if errGame != nil {
		log.Printf("main: fatal: %v", errGame)
		return
	}

	gob.Register(msg.Update{})
	gob.Register(msg.Fire{})

	app.Main(func(a app.App) {
		log.Print("app.Main begin")

		go serverHandler(a, game.serverAddr, game.serverOutput)

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

				paintRequests++

				if now := time.Now().Second(); now != sec {
					// once per second event
					log.Printf("requests: %d, paints: %d, team=%d", paintRequests, paints, game.playerTeam)
					paintRequests = 0
					paints = 0
					sec = now
				}

				//if !slowPaint || paintRequests == 0 {
				paints++
				game.paint()
				a.Publish()
				//}

				if slowPaint {
					time.Sleep(200 * time.Millisecond) // slow down paint event request
				}

				// we request next paint event
				// in order to draw as fast as possible
				a.Send(paint.Event{})
			case mouse.Event:
				press := (t.Direction & 1) == 1
				release := (t.Direction & 2) == 2
				game.input(press, release, t.X, t.Y)
			case touch.Event:
				press := t.Type == touch.TypeBegin
				release := t.Type == touch.TypeEnd
				game.input(press, release, t.X, t.Y)
			case size.Event:
				game.resize(t.WidthPx, t.HeightPx)
			case msg.Update:
				//log.Printf("app.Main event update: %v", t)
				game.playerTeam = t.Team
				game.playerFuel = t.Fuel
				game.updateInterval = t.Interval
				game.missiles = t.WorldMissiles
				game.cannons = t.Cannons
				game.updateLast = time.Now()
			}
		}

		log.Print("app.Main end")
	})

	log.Print("main end")
}

func loadFull(name string) ([]byte, error) {
	f, errOpen := asset.Open(name)
	if errOpen != nil {
		return nil, errOpen
	}
	defer f.Close()
	buf, errRead := ioutil.ReadAll(f)
	if errRead != nil {
		return nil, errRead
	}
	log.Printf("loaded: %s (%d bytes)", name, len(buf))
	return buf, nil
}

func (game *gameState) resize(w, h int) {
	if game.width != w || game.height != h {
		log.Printf("resize: %d,%d", w, h)
	}
	game.width = w
	game.height = h

	glc := game.gl // shortcut
	if glc == nil {
		return
	}

	glc.Viewport(0, 0, w, h)

	if h >= w {
		aspect := float64(h) / float64(w)
		game.minX = -1
		game.maxX = 1
		game.minY = -aspect
		game.maxY = aspect
	} else {
		aspect := float64(w) / float64(h)
		game.minX = -aspect
		game.maxX = aspect
		game.minY = -1
		game.maxY = 1
	}

	//goglmath.SetOrthoMatrix(&game.proj, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
}

func (game *gameState) input(press, release bool, x, y float32) {
	log.Printf("input: event press=%v %f,%f (%d x %d)", press, x, y, game.width, game.height)

	if press {
		game.serverOutput <- msg.Fire{}
	}
}

func (game *gameState) start(glc gl.Context) {
	log.Printf("start")

	var err error
	game.program, err = glutil.CreateProgram(glc, game.shaderVert, game.shaderFrag)
	if err != nil {
		log.Printf("start: error creating GL program: %v", err)
		return
	}

	log.Printf("start: shader compiled")

	//game.bufTriangle = glc.CreateBuffer()
	//glc.BindBuffer(gl.ARRAY_BUFFER, game.bufTriangle)
	//glc.BufferData(gl.ARRAY_BUFFER, triangleData, gl.STATIC_DRAW)

	game.bufSquare = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
	glc.BufferData(gl.ARRAY_BUFFER, squareData, gl.STATIC_DRAW)

	game.bufSquareWire = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareWire)
	glc.BufferData(gl.ARRAY_BUFFER, squareWireData, gl.STATIC_DRAW)

	game.bufCannon = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufCannon)
	glc.BufferData(gl.ARRAY_BUFFER, cannonData, gl.STATIC_DRAW)

	game.bufCannonDown = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufCannonDown)
	glc.BufferData(gl.ARRAY_BUFFER, cannonDownData, gl.STATIC_DRAW)

	game.position = glc.GetAttribLocation(game.program, "position")
	game.P = glc.GetUniformLocation(game.program, "P")
	game.color = glc.GetUniformLocation(game.program, "color")

	glc.ClearColor(.5, .5, .5, 1) // gray background

	game.gl = glc

	log.Printf("start: shader initialized")
}

func (game *gameState) stop() {
	log.Printf("stop")

	glc := game.gl // shortcut

	glc.DeleteProgram(game.program)
	//glc.DeleteBuffer(game.bufTriangle)
	glc.DeleteBuffer(game.bufSquare)
	glc.DeleteBuffer(game.bufSquareWire)
	glc.DeleteBuffer(game.bufCannon)
	glc.DeleteBuffer(game.bufCannonDown)

	game.gl = nil

	log.Printf("stop: shader disposed")
}

func (game *gameState) paint() {
	glc := game.gl // shortcut

	now := time.Now()
	elap := time.Since(game.updateLast)

	glc.Clear(gl.COLOR_BUFFER_BIT) // draw ClearColor background

	glc.UseProgram(game.program)
	glc.EnableVertexAttribArray(game.position)

	glc.Uniform4f(game.color, .5, .9, .5, 1) // green

	// Draw orthorgraphic triangle
	/*
		glc.UniformMatrix4fv(game.P, game.proj.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, game.bufTriangle)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.TRIANGLES, 0, triangleVertexCount)
	*/

	// Wire rectangle around fuel bar
	squareWireMVP := goglmath.NewMatrix4Identity()
	squareWireMVP.Translate(-1, -1, .1, 1) // z=.1 put in front of fuel bar
	squareWireMVP.Scale(2, .04, 1, 1)
	glc.UniformMatrix4fv(game.P, squareWireMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareWire)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.LINE_LOOP, 0, squareWireVertexCount)

	// Fuel bar
	glc.Uniform4f(game.color, .9, .9, .9, 1) // white
	squareMVP := goglmath.NewMatrix4Identity()
	squareMVP.Translate(-1, -1, 0, 1)
	fuel := float64(future.Fuel(game.playerFuel, elap))
	squareMVP.Scale(2*fuel/10, .04, 1, 1) // width is fuel, heigh is 5%
	glc.UniformMatrix4fv(game.P, squareMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)

	cannonWidth := .1 // 10%
	cannonHeight := .1 // 10%

	// Cannons
	glc.Uniform4f(game.color, .9, .2, .2, 1) // red
	for _, can := range game.cannons {
		var canBuf gl.Buffer
		var y float64
		if can.Team == game.playerTeam {
			// upward
			y = -.95
			canBuf = game.bufCannon
		} else {
			// downward
			y = 1
			canBuf = game.bufCannonDown
		}
		// x: from 0.0,1.0 to minX,(maxX-width)
		//MVP := goglmath.NewMatrix4Identity()
		var MVP goglmath.Matrix4
		goglmath.SetOrthoMatrix(&MVP, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
		cannonX, _ := future.CannonX(can.CoordX, can.Speed, elap)
		x := float64(cannonX) * (game.maxX - cannonWidth - game.minX) + game.minX
		//MVP.Translate((2-cannonWidth)*float64(cannonX)-1, y, 0, 1)
		MVP.Translate(x, y, 0, 1)
		MVP.Scale(cannonWidth, cannonHeight, 1, 1) // 10% size
		glc.UniformMatrix4fv(game.P, MVP.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, canBuf)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.TRIANGLES, 0, cannonVertexCount)
	}

	// Missiles
	glc.Uniform4f(game.color, .9, .9, .4, 1) // yellow
	for _, miss := range game.missiles {
		missileMVP := goglmath.NewMatrix4Identity()
		width := .01                    // 1%
		height := .07                   // 7%
		x := float64(miss.CoordX)*2 - 1 // FIXME use both cannon and missile widths
		y := float64(future.MissileY(0, miss.Speed, now.Sub(miss.Start)))
		if miss.Team == game.playerTeam {
			// upward
			y = y*2 - 1 // FIXME use heights
		} else {
			// downward
			y = 1 - y*2 // FIXME use heights
		}
		missileMVP.Translate(x, y, 0, 1)
		missileMVP.Scale(width, height, 1, 1)
		glc.UniformMatrix4fv(game.P, missileMVP.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)
	}

	glc.DisableVertexAttribArray(game.position)
}

const (
	coordsPerVertex = 3
	//triangleVertexCount   = 3
	cannonVertexCount     = 3
	squareVertexCount     = 6
	squareWireVertexCount = 4
)

/*
var triangleData = f32.Bytes(binary.LittleEndian,
	0.0, 1.0, 0.0, // top left
	0.0, 0.0, 0.0, // bottom left
	1.0, 0.0, 0.0, // bottom right
)
*/

var cannonData = f32.Bytes(binary.LittleEndian,
	0.5, 1.0, 0.0,
	0.0, 0.0, 0.0,
	1.0, 0.0, 0.0,
)

var cannonDownData = f32.Bytes(binary.LittleEndian,
	0.5, -1.0, 0.0,
	1.0, 0.0, 0.0,
	0.0, 0.0, 0.0,
)

var squareData = f32.Bytes(binary.LittleEndian,
	0.0, 1.0, 0.0,
	0.0, 0.0, 0.0,
	1.0, 0.0, 0.0,
	1.0, 0.0, 0.0,
	1.0, 1.0, 0.0,
	0.0, 1.0, 0.0,
)

var squareWireData = f32.Bytes(binary.LittleEndian,
	0.0, 1.0, 0.0,
	0.0, 0.0, 0.0,
	1.0, 0.0, 0.0,
	1.0, 1.0, 0.0,
)
