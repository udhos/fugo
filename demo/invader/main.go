// +build darwin linux windows

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"image"
	"image/color"
	_ "image/png" // The _ means to import a package purely for its initialization side effects.
	"io/ioutil"
	"log"
	"os"
	"strconv"
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

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/msg"
	"github.com/udhos/fugo/trace"
	"github.com/udhos/fugo/unit"
	"github.com/udhos/fugo/version"
)

type gameState struct {
	width              int
	height             int
	gl                 gl.Context
	program            gl.Program
	programTex         gl.Program
	bufSquare          gl.Buffer
	bufSquareWire      gl.Buffer
	bufCannon          gl.Buffer
	bufCannonDown      gl.Buffer
	bufSquareElemIndex gl.Buffer
	bufSquareElemData  gl.Buffer

	// simple shader
	position gl.Attrib
	P        gl.Uniform // projection mat4 uniform
	color    gl.Uniform

	// texturizing shader
	texPosition     gl.Attrib
	texTextureCoord gl.Attrib
	texSampler      gl.Uniform
	texMVP          gl.Uniform // MVP mat4
	texButtonFire   gl.Texture
	texButtonTurn   gl.Texture
	ship            gl.Texture
	missile         gl.Texture
	brick           gl.Texture

	streamLaser beep.StreamSeekCloser

	cannonWidth   float64
	cannonHeight  float64
	missileWidth  float64
	missileHeight float64
	brickWidth    float64
	brickHeight   float64
	debugBound    bool

	atlas      *fontAtlas
	t1         *fontText
	scoreOur   *fontText
	scoreTheir *fontText

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
	missiles               map[int]*msg.Missile
	cannons                map[int]*msg.Cannon
	bricks                 map[int]*msg.Brick
	tracer                 *trace.Trace
}

func playLaser(game *gameState) {
	game.streamLaser.Seek(0)
	speaker.Play(beep.Seq(game.streamLaser))
}

func loadSound(name string) (beep.StreamSeekCloser, error) {
	f, errSndLaser := asset.Open(name)
	if errSndLaser != nil {
		return nil, errSndLaser
	}

	s, format, errDec := wav.Decode(f)
	if errDec != nil {
		return nil, errDec
	}

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	return s, nil
}

func newGame() (*gameState, error) {
	game := &gameState{
		minX:     -1,
		maxX:     1,
		minY:     -1,
		maxY:     1,
		missiles: map[int]*msg.Missile{},
		cannons:  map[int]*msg.Cannon{},
	}

	sndLaser := "95933__robinhood76__01665-thin-laser-blast.wav"
	var errSndLaser error
	game.streamLaser, errSndLaser = loadSound(sndLaser)
	if errSndLaser != nil {
		log.Printf("laser sound: %v", errSndLaser)
	}

	if errVert := flagStr(&game.shaderVert, "shader.vert"); errVert != nil {
		log.Printf("load vertex shader: %v", errVert)
		return nil, errVert
	}

	if errFrag := flagStr(&game.shaderFrag, "shader.frag"); errFrag != nil {
		log.Printf("load fragment shader: %v", errFrag)
		return nil, errFrag
	}

	if errVert := flagStr(&game.shaderTexVert, "shader_tex.vert"); errVert != nil {
		log.Printf("load vertex tex shader: %v", errVert)
		return nil, errVert
	}

	if errFrag := flagStr(&game.shaderTexFrag, "shader_tex.frag"); errFrag != nil {
		log.Printf("load fragment tex shader: %v", errFrag)
		return nil, errFrag
	}

	if errServ := flagStr(&game.serverAddr, "server.txt"); errServ != nil {
		log.Printf("load server: %v", errServ)
		return nil, errServ
	}

	log.Printf("server: [%s]", game.serverAddr)

	var tracer string
	errTrace := flagStr(&tracer, "trace.txt")
	if errTrace != nil {
		log.Printf("trace file: %v", errTrace)
	} else {
		tracer = strings.TrimSpace(tracer)
		log.Printf("tracer: [%s]", tracer)
		game.tracer, errTrace = trace.New(tracer)
		if errTrace != nil {
			log.Printf("trace sock: %v", errTrace)
		}
	}
	log.Printf("tracer: %v", game.tracer)

	flagBool(&game.debugBound, "box.txt")

	game.tracef("trace, hello from invader app")

	game.updateInterval = 2 * time.Second
	game.updateLast = time.Now()

	game.serverOutput = make(chan msg.Button)

	return game, nil
}

func flipY(name string, img *image.NRGBA) {
	b := img.Bounds()
	midY := (b.Max.Y - b.Min.Y) / 2
	for x := b.Min.X; x < b.Max.X; x++ {
		for y1 := b.Min.Y; y1 < midY; y1++ {
			y2 := b.Max.Y - y1 - 1
			c1 := img.At(x, y1)
			c2 := img.At(x, y2)
			img.Set(x, y1, c2)
			img.Set(x, y2, c1)
		}
	}
	log.Printf("image y-flipped: %s", name)
}

func main() {
	log.Print("main begin - fugo invader version " + version.Version)

	slowPaint := len(os.Args) > 1
	if !slowPaint {
		flagBool(&slowPaint, "slow.txt")
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

				game.updateLast = time.Now()
				elap := time.Since(game.updateLast)

				missiles := map[int]*msg.Missile{}
				for _, m := range t.WorldMissiles {
					old, found := game.missiles[m.ID]
					if found {
						oldY := future.MissileY(old.CoordY, old.Speed, elap)
						newY := future.MissileY(m.CoordY, m.Speed, elap)
						if newY < oldY {
							// refuse to move back in time
							missiles[m.ID] = old // prevent deletion
							continue
						}
					}
					missiles[m.ID] = m
				}
				game.missiles = missiles

				bricks := map[int]*msg.Brick{}
				for _, m := range t.Bricks {
					bricks[m.ID] = m
				}
				game.bricks = bricks

				cannons := map[int]*msg.Cannon{}
				for _, c := range t.Cannons {
					old, found := game.cannons[c.ID]
					if found {
						if old.Speed == c.Speed {
							oldX, _ := future.CannonX(old.CoordX, old.Speed, elap)
							newX, _ := future.CannonX(c.CoordX, c.Speed, elap)
							if (old.Speed >= 0 && newX < oldX) || (old.Speed < 0 && newX > oldX) {
								// refuse to move back in time
								cannons[c.ID] = old // prevent deletion
								continue
							}
						}
					}
					cannons[c.ID] = c
				}
				game.cannons = cannons

				game.t1.write(fmt.Sprintf("%f", t.Fuel))

				var our, their string
				our = strconv.Itoa(t.Scores[t.Team])
				their = strconv.Itoa(t.Scores[1-t.Team])
				game.scoreOur.write(our)
				game.scoreTheir.write(their)

				if t.FireSound {
					playLaser(game)
				}
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
	game.texMVP = getUniformLocation(glc, game.programTex, "MVP")
	game.texSampler = getUniformLocation(glc, game.programTex, "sampler")

	var errLoad error
	game.texButtonFire, _, errLoad = loadTexture(glc, "icon-missile.png", true)
	if errLoad != nil {
		log.Printf("start: texture load: %v", errLoad)
	}
	game.texButtonTurn, _, errLoad = loadTexture(glc, "icon-right-left.png", true)
	if errLoad != nil {
		log.Printf("start: texture load: %v", errLoad)
	}
	var shipImg *image.NRGBA
	game.ship, shipImg, errLoad = loadTexture(glc, "ship.png", true)
	if errLoad != nil {
		log.Printf("start: texture load: %v", errLoad)
	}

	game.cannonWidth, game.cannonHeight = unit.BoxSize(shipImg, unit.ScaleCannon)

	var missImg *image.NRGBA
	game.missile, missImg, errLoad = loadTexture(glc, "rocket.png", true)
	if errLoad != nil {
		log.Printf("start: texture load: %v", errLoad)
	}

	game.missileWidth, game.missileHeight = unit.BoxSize(missImg, unit.ScaleMissile)

	br := "brick.png"
	var brickImg *image.NRGBA
	game.brick, brickImg, errLoad = loadTexture(glc, br, true)
	if errLoad != nil {
		log.Printf("start: texture load: %v", errLoad)
	}
	game.brickWidth, game.brickHeight = unit.BoxSize(brickImg, unit.ScaleBrick)
	log.Printf("brick: %s scale=%v =>%vx%v", br, unit.ScaleBrick, game.brickWidth, game.brickHeight)

	game.bufSquareElemData = glc.CreateBuffer()
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareElemData)
	glc.BufferData(gl.ARRAY_BUFFER, squareElemData, gl.STATIC_DRAW)

	game.bufSquareElemIndex = glc.CreateBuffer()
	glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufSquareElemIndex)
	glc.BufferData(gl.ELEMENT_ARRAY_BUFFER, squareElemIndex, gl.STATIC_DRAW)

	var errFont error
	game.atlas, errFont = newAtlas(glc, color.NRGBA{128, 230, 128, 255}, game.texPosition, game.texTextureCoord)
	if errFont != nil {
		log.Printf("start: font: %v", errFont)
	}

	game.t1 = newText(game.atlas)
	game.t1.write("invader")
	game.scoreOur = newText(game.atlas)
	game.scoreOur.write("?")
	game.scoreTheir = newText(game.atlas)
	game.scoreTheir.write("?")

	glc.ClearColor(.5, .5, .5, 1) // gray background
	glc.ClearDepthf(1)            // default
	glc.Enable(gl.DEPTH_TEST)     // enable depth testing
	glc.DepthFunc(gl.LEQUAL)      // gl.LESS is default depth test
	glc.DepthRangef(0, 1)         // default

	game.gl = glc

	log.Printf("start: shaders initialized")
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
	// FIXME 1000 is a hack to detect a bad location.Value, since it can't represent -1
	if location.Value > 1000 {
		log.Printf("bad attribute '%s' location: %d", attr, location.Value)
	}
	return location
}

func (game *gameState) stop() {
	log.Printf("stop")

	glc := game.gl // shortcut

	if game.scoreOur != nil {
		game.scoreOur.delete()
		game.scoreOur = nil
	}

	if game.scoreTheir != nil {
		game.scoreTheir.delete()
		game.scoreTheir = nil
	}

	if game.t1 != nil {
		game.t1.delete()
		game.t1 = nil
	}

	if game.atlas != nil {
		game.atlas.delete()
		game.atlas = nil
	}

	glc.DeleteProgram(game.program)
	glc.DeleteProgram(game.programTex)
	glc.DeleteTexture(game.texButtonFire)
	glc.DeleteTexture(game.texButtonTurn)
	glc.DeleteTexture(game.ship)
	glc.DeleteTexture(game.missile)
	glc.DeleteTexture(game.brick)
	glc.DeleteBuffer(game.bufSquareElemIndex)
	glc.DeleteBuffer(game.bufSquareElemData)

	glc.DeleteBuffer(game.bufSquare)
	glc.DeleteBuffer(game.bufSquareWire)
	glc.DeleteBuffer(game.bufCannon)
	glc.DeleteBuffer(game.bufCannonDown)

	game.gl = nil

	log.Printf("stop: shader disposed")
}

func (game *gameState) setOrtho(m *goglmath.Matrix4) {
	// near=1 far=-1 -> keep Z
	// near=-1 far=1 -> flip Z
	goglmath.SetOrthoMatrix(m, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
}

const buttons = 5

func (game *gameState) buttonEdge() float64 {
	screenWidth := game.maxX - game.minX
	return screenWidth / float64(buttons)
}

const (
	coordsPerVertex       = 3
	squareVertexCount     = 6
	squareWireVertexCount = 4
)

var cannonData = f32.Bytes(binary.LittleEndian,
	0.5, 1.0, 0.0,
	0.0, 0.0, 0.0,
	1.0, 0.0, 0.0,
)

var cannonDownData = f32.Bytes(binary.LittleEndian,
	0.5, 0.0, 0.0,
	1.0, 1.0, 0.0,
	0.0, 1.0, 0.0,
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

const squareElemIndexCount = 6

var squareElemIndex = intsToBytes([]uint32{
	0, 1, 2,
	2, 3, 0,
})

func intsToBytes(s []uint32) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, s)
	b := buf.Bytes()
	return b
}

var squareElemData = f32.Bytes(binary.LittleEndian,
	// pos         tex
	// ----------  --------
	0.0, 1.0, 0.0, 0.0, 1.0, // 0
	0.0, 0.0, 0.0, 0.0, 0.0, // 1
	1.0, 0.0, 0.0, 1.0, 0.0, // 2
	1.0, 1.0, 0.0, 1.0, 1.0, // 3
)
