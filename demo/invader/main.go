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

	"github.com/udhos/fugo/msg"
	"github.com/udhos/fugo/trace"
)

type gameState struct {
	width         int
	height        int
	gl            gl.Context
	program       gl.Program
	programTex    gl.Program
	bufSquare     gl.Buffer
	bufSquareWire gl.Buffer
	bufCannon     gl.Buffer
	bufCannonDown gl.Buffer

	position gl.Attrib
	P        gl.Uniform // projection mat4 uniform
	color    gl.Uniform

	texPosition     gl.Attrib
	texTextureCoord gl.Attrib

	minX, maxX, minY, maxY float64
	shaderVert             string
	shaderFrag             string
	shaderTexVert          string
	shaderTexFrag          string
	serverAddr             string
	serverOutput           chan msg.Button
	playerFuel             float32
	playerTeam             int
	updateInterval         time.Duration
	updateLast             time.Time
	missiles               []*msg.Missile
	cannons                []*msg.Cannon
	tracer                 *trace.Trace
}

func newGame() (*gameState, error) {
	game := &gameState{
		minX: -1,
		maxX: 1,
		minY: -1,
		maxY: 1,
	}

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

	texVert, errVert := loadFull("shader_tex.vert")
	if errVert != nil {
		log.Printf("load vertex shader: %v", errVert)
		return nil, errVert
	}
	game.shaderTexVert = string(texVert)

	texFrag, errFrag := loadFull("shader_tex.frag")
	if errFrag != nil {
		log.Printf("load fragment shader: %v", errFrag)
		return nil, errFrag
	}
	game.shaderTexFrag = string(texFrag)

	server, errServ := loadFull("server.txt")
	if errServ != nil {
		log.Printf("load server: %v", errServ)
		return nil, errServ
	}
	game.serverAddr = strings.TrimSpace(string(server))

	log.Printf("server: [%s]", game.serverAddr)

	tracer, errTrace := loadFull("trace.txt")
	if errTrace != nil {
		log.Printf("trace file: %v", errTrace)
	} else {
		tracer := strings.TrimSpace(string(tracer))
		log.Printf("tracer: [%s]", tracer)
		game.tracer, errTrace = trace.New(tracer)
		if errTrace != nil {
			log.Printf("trace sock: %v", errTrace)
		}
	}
	log.Printf("tracer: %v", game.tracer)

	game.tracef("trace, hello from invader app")

	game.updateInterval = time.Second
	game.updateLast = time.Now()

	game.serverOutput = make(chan msg.Button)

	return game, nil
}

func main() {
	log.Print("main begin")

	slowPaint := len(os.Args) > 1

	if !slowPaint {
		_, errSlow := loadFull("slow.txt")
		if errSlow == nil {
			slowPaint = true
		}
	}

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
	gob.Register(msg.Button{})

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

func (game *gameState) tracef(format string, v ...interface{}) {
	if game.tracer == nil {
		return
	}
	game.tracer.Printf(format, v...)
}

func (game *gameState) resize(w, h int) {
	if game.width != w || game.height != h {
		log.Printf("resize: %d,%d", w, h)
	}
	game.width = w
	game.height = h

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

	log.Printf("resize: %v,%v,%v,%v", game.minX, game.maxX, game.minY, game.maxY)

	glc := game.gl // shortcut
	if glc == nil {
		return
	}

	glc.Viewport(0, 0, w, h)
}

func (game *gameState) input(press, release bool, pixelX, pixelY float32) {
	log.Printf("input: event press=%v %f,%f (%d x %d)", press, pixelX, pixelY, game.width, game.height)

	if press {
		//x := float64(pixelX) / float64(game.width-1) * (game.maxX - game.minX) + game.minX
		y := float64(pixelY)/float64(game.height-1)*(game.minY-game.maxY) + game.maxY

		if y < (game.minY + game.buttonEdge()) {
			// might hit button
			pixelsPerButton := float32(game.width) / float32(buttons)
			b := pixelX / pixelsPerButton
			game.serverOutput <- msg.Button{ID: int(b)}
		}
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

	var errTex error
	game.programTex, errTex = glutil.CreateProgram(glc, game.shaderTexVert, game.shaderTexFrag)
	if errTex != nil {
		log.Printf("start: error creating GL texturizer program: %v", errTex)
		return
	}
	log.Printf("start: texturizing shader compiled")

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

	game.position = getAttribLocation(glc, game.program, "position")
	game.P = getUniformLocation(glc, game.program, "P")
	game.color = getUniformLocation(glc, game.program, "color")

	game.texPosition = getAttribLocation(glc, game.programTex, "position")
	game.texTextureCoord = getAttribLocation(glc, game.programTex, "textureCoord")

	glc.ClearColor(.5, .5, .5, 1) // gray background

	game.gl = glc

	log.Printf("start: shader initialized")
}

func getUniformLocation(glc gl.Context, prog gl.Program, uniform string) gl.Uniform {
	location := glc.GetUniformLocation(prog, uniform)
	if location.Value < 0 {
		log.Printf("bad uniform '%s' location: %d", uniform, location.Value)
	}
	return location
}

func getAttribLocation(glc gl.Context, prog gl.Program, attr string) gl.Attrib {
	location := glc.GetAttribLocation(prog, attr)
	if location.Value < 0 {
		log.Printf("bad attribute '%s' location: %d", attr, location.Value)
	}
	return location
}

func (game *gameState) stop() {
	log.Printf("stop")

	glc := game.gl // shortcut

	glc.DeleteProgram(game.program)
	glc.DeleteBuffer(game.bufSquare)
	glc.DeleteBuffer(game.bufSquareWire)
	glc.DeleteBuffer(game.bufCannon)
	glc.DeleteBuffer(game.bufCannonDown)

	game.gl = nil

	log.Printf("stop: shader disposed")
}

func (game *gameState) setOrtho(m *goglmath.Matrix4) {
	goglmath.SetOrthoMatrix(m, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
}

const buttons = 5

func (game *gameState) buttonEdge() float64 {
	screenWidth := game.maxX - game.minX
	return screenWidth / float64(buttons)
}

const (
	coordsPerVertex       = 3
	cannonVertexCount     = 3
	squareVertexCount     = 6
	squareWireVertexCount = 4
)

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
