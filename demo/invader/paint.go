// +build darwin linux windows

package main

import (
	//"log"
	"time"

	"github.com/udhos/goglmath"

	"golang.org/x/mobile/gl"

	"github.com/udhos/fugo/future"
	"github.com/udhos/fugo/unit"
)

func (game *gameState) paint() {
	glc := game.gl // shortcut

	elap := time.Since(game.updateLast)

	glc.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	glc.UseProgram(game.program)
	glc.EnableVertexAttribArray(game.position)

	glc.Uniform4f(game.color, .5, .9, .5, 1) // green

	screenWidth := game.maxX - game.minX
	screenHeight := game.maxY - game.minY
	statusBarHeight := .05
	scoreTop := game.maxY - statusBarHeight
	scoreBarHeight := .06
	fieldTop := scoreTop - scoreBarHeight

	buttonWidth := game.buttonEdge()
	buttonHeight := buttonWidth

	// clamp height
	maxH := .3 * screenHeight
	if buttonHeight > maxH {
		buttonHeight = maxH
	}

	for i := 0; i < buttons; i++ {
		//squareWireMVP := goglmath.NewMatrix4Identity()
		var squareWireMVP goglmath.Matrix4
		game.setOrtho(&squareWireMVP)
		x := game.minX + float64(i)*buttonWidth
		squareWireMVP.Translate(x, game.minY, .1, 1) // z=.1 put in front of fuel bar
		squareWireMVP.Scale(buttonWidth, buttonHeight, 1, 1)
		glc.UniformMatrix4fv(game.P, squareWireMVP.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareWire)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.LINE_LOOP, 0, squareWireVertexCount)
	}

	fuelBottom := game.minY + buttonHeight
	fuelHeight := .04 * screenHeight

	// Wire rectangle around fuel bar
	fuelBarR := unit.Rect{X1: game.minX, Y1: fuelBottom, X2: game.minX + screenWidth, Y2: fuelBottom + fuelHeight}
	game.drawWireRect(fuelBarR, .5, .9, .5, 1, .1)

	// Fuel bar
	fuel := float64(future.Fuel(game.playerFuel, elap))
	fuelR := unit.Rect{X1: game.minX, Y1: fuelBottom, X2: game.minX + screenWidth*fuel/10, Y2: fuelBottom + fuelHeight}
	game.drawRect(fuelR, .9, .9, .9, 1, 0)

	cannonBottom := fuelBottom + fuelHeight + .01

	// Cannons
	for _, can := range game.cannons {
		switch {
		case can.Life <= 0:
			glc.Uniform4f(game.color, .9, .2, .2, 1) // red - dead
		case can.Player:
			glc.Uniform4f(game.color, .2, .2, .8, 1) // blue - player
		default:
			glc.Uniform4f(game.color, .5, .9, .5, 1) // green - other
		}

		cannonX, _ := future.CannonX(can.CoordX, can.Speed, elap)

		/*
			var canBuf gl.Buffer
			if up {
				// upward
				canBuf = game.bufCannon
			} else {
				// downward
				canBuf = game.bufCannonDown
			}
		*/

		up := can.Team == game.playerTeam

		r := unit.CannonBox(game.minX, game.maxX, float64(cannonX), fieldTop, cannonBottom, up)

		/*
			var MVP goglmath.Matrix4
			game.setOrtho(&MVP)
			MVP.Translate(r.X1, r.Y1, 0, 1)
			MVP.Scale(unit.CannonWidth, unit.CannonHeight, 1, 1)
			glc.UniformMatrix4fv(game.P, MVP.Data())
			glc.BindBuffer(gl.ARRAY_BUFFER, canBuf)
			glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
			glc.DrawArrays(gl.TRIANGLES, 0, cannonVertexCount)
		*/

		// life bar
		lifeBarH := .02
		lifeR := r
		lifeR.X2 = lifeR.X1 + unit.CannonWidth*float64(can.Life)
		lifeR2 := r
		lifeR2.X1 = lifeR.X2
		if up {
			lifeR.Y2 = lifeR.Y1 + lifeBarH
			lifeR2.Y2 = lifeR.Y2
		} else {
			lifeR.Y1 = lifeR.Y2 - lifeBarH
			lifeR2.Y1 = lifeR.Y1
		}
		game.drawRect(lifeR, .5, .5, .8, 1, .05)
		game.drawRect(lifeR2, .9, .5, .5, 1, .05)

		//game.drawWireRect(r, 1, 1, 1, 1, .1) // debug-only
	}

	// Missiles
	for _, miss := range game.missiles {
		up := miss.Team == game.playerTeam
		y := float64(future.MissileY(miss.CoordY, miss.Speed, elap))

		r := unit.MissileBox(game.minX, game.maxX, float64(miss.CoordX), y, fieldTop, cannonBottom, up)

		game.drawRect(r, .9, .9, .4, 1, 0)

		//game.drawWireRect(r, 1, 1, 1, 1, .1) // debug-only
	}

	//game.debugZ(glc)

	glc.DisableVertexAttribArray(game.position)

	game.paintTex(glc, elap, buttonWidth, buttonHeight, scoreTop, scoreBarHeight, fieldTop, cannonBottom) // another shader
}

func (game *gameState) drawRect(rect unit.Rect, r, g, b, a float32, z float64) {
	glc := game.gl // shortcut

	glc.Uniform4f(game.color, r, g, b, a)

	var squareMVP goglmath.Matrix4
	game.setOrtho(&squareMVP)
	squareMVP.Translate(rect.X1, rect.Y1, z, 1)
	squareMVP.Scale(rect.X2-rect.X1, rect.Y2-rect.Y1, 1, 1)
	glc.UniformMatrix4fv(game.P, squareMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)
}

func (game *gameState) drawWireRect(rect unit.Rect, r, g, b, a float32, z float64) {
	glc := game.gl // shortcut

	glc.Uniform4f(game.color, r, g, b, a)

	var squareWireMVP goglmath.Matrix4
	game.setOrtho(&squareWireMVP)
	squareWireMVP.Translate(rect.X1, rect.Y1, z, 1)
	squareWireMVP.Scale(rect.X2-rect.X1, rect.Y2-rect.Y1, 1, 1)
	glc.UniformMatrix4fv(game.P, squareWireMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareWire)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.LINE_LOOP, 0, squareWireVertexCount)
}

/*
func (game *gameState) debugZ(glc gl.Context) {
	var MVP goglmath.Matrix4
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)

	glc.Uniform4f(game.color, .9, .9, .9, 1) // white
	game.setOrtho(&MVP)
	MVP.Translate(0, 0, .1, 1) // white z=.1 front - closer to eye
	MVP.Scale(.1, .1, 1, 1)
	glc.UniformMatrix4fv(game.P, MVP.Data())
	glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)

	p1x, p1y, p1z, p1w := MVP.Transform(0, 0, 0, 1)

	glc.Uniform4f(game.color, .9, .5, .5, 1) // red
	game.setOrtho(&MVP)
	MVP.Translate(.05, .05, -.1, 1) // red z=-.1 back - farther from eye
	MVP.Scale(.1, .1, 1, 1)
	glc.UniformMatrix4fv(game.P, MVP.Data())
	glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)

	p2x, p2y, p2z, p2w := MVP.Transform(0, 0, 0, 1)

	log.Printf("white=%v,%v,%v,%v red=%v,%v,%v,%v", p1x, p1y, p1z, p1w, p2x, p2y, p2z, p2w)
	time.Sleep(time.Second)
}
*/

func (game *gameState) paintTex(glc gl.Context, elap time.Duration, buttonWidth, buttonHeight, scoreTop, scoreHeight, fieldTop, cannonBottom float64) {

	glc.Enable(gl.BLEND)
	glc.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	glc.UseProgram(game.programTex)
	glc.EnableVertexAttribArray(game.texPosition)
	glc.EnableVertexAttribArray(game.texTextureCoord)

	tunit := 0
	glc.ActiveTexture(gl.TEXTURE0 + gl.Enum(tunit))
	glc.Uniform1i(game.texSampler, tunit)

	// draw button - fire

	/*
		glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareElemData)
		glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufSquareElemIndex)

		// square geometry
		elemFirst := 0
		elemCount := squareElemIndexCount // 6
		elemType := gl.Enum(gl.UNSIGNED_INT)
		elemSize := 4

		strideSize := 5 * 4 // 5 x 4 bytes
		itemsPosition := 3
		itemsTexture := 2
		offsetPosition := 0
		offsetTexture := itemsPosition * 4 // 3 x 4 bytes
		glc.VertexAttribPointer(game.texPosition, itemsPosition, gl.FLOAT, false, strideSize, offsetPosition)
		glc.VertexAttribPointer(game.texTextureCoord, itemsTexture, gl.FLOAT, false, strideSize, offsetTexture)

		fireIndex := 0
		var MVPfire goglmath.Matrix4
		game.setOrtho(&MVPfire)
		scaleButtonFire := buttonHeight // FIXME using square -- should use image aspect?
		xFire := game.minX + float64(fireIndex)*buttonWidth
		MVPfire.Translate(xFire, game.minY, 0, 1)
		MVPfire.Scale(scaleButtonFire, scaleButtonFire, 1, 1)
		glc.UniformMatrix4fv(game.texMVP, MVPfire.Data())

		glc.BindTexture(gl.TEXTURE_2D, game.texButtonFire)

		glc.DrawElements(gl.TRIANGLES, elemCount, elemType, elemFirst*elemSize)
	*/

	fireIndex := 0
	scaleButtonFire := buttonHeight // FIXME using square -- should use image aspect?
	xFire := game.minX + float64(fireIndex)*buttonWidth
	game.drawImage(game.texButtonFire, xFire, game.minY, scaleButtonFire, scaleButtonFire)

	// draw button - turn

	/*
		turnIndex := 1
		var MVPturn goglmath.Matrix4
		game.setOrtho(&MVPturn)
		scaleButtonTurn := buttonHeight // FIXME using square -- should use image aspect?
		xTurn := game.minX + float64(turnIndex)*buttonWidth
		MVPturn.Translate(xTurn, game.minY, 0, 1)
		MVPturn.Scale(scaleButtonTurn, scaleButtonTurn, 1, 1)
		glc.UniformMatrix4fv(game.texMVP, MVPturn.Data())

		glc.BindTexture(gl.TEXTURE_2D, game.texButtonTurn)

		glc.DrawElements(gl.TRIANGLES, elemCount, elemType, elemFirst*elemSize)
	*/

	turnIndex := 1
	scaleButtonTurn := buttonHeight // FIXME using square -- should use image aspect?
	xTurn := game.minX + float64(turnIndex)*buttonWidth
	game.drawImage(game.texButtonTurn, xTurn, game.minY, scaleButtonTurn, scaleButtonTurn)

	// Cannons
	for _, can := range game.cannons {

		cannonX, _ := future.CannonX(can.CoordX, can.Speed, elap)

		up := can.Team == game.playerTeam

		r := unit.CannonBox(game.minX, game.maxX, float64(cannonX), fieldTop, cannonBottom, up)

		/*
			var MVP goglmath.Matrix4
			game.setOrtho(&MVP)
			MVP.Translate(r.X1, r.Y1, 0, 1)
			MVP.Scale(unit.CannonWidth, unit.CannonHeight, 1, 1)
			glc.UniformMatrix4fv(game.P, MVP.Data())
			glc.BindBuffer(gl.ARRAY_BUFFER, canBuf)
			glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
			glc.DrawArrays(gl.TRIANGLES, 0, cannonVertexCount)
		*/

		game.drawImage(game.ship, r.X1, r.Y1, unit.CannonWidth, unit.CannonHeight)
	}

	// font

	var MVPfont goglmath.Matrix4
	game.setOrtho(&MVPfont)
	MVPfont.Translate(0, 0, 0, 1)
	MVPfont.Scale(.1, .1, 1, 1)
	glc.UniformMatrix4fv(game.texMVP, MVPfont.Data())

	game.t1.draw()

	// score
	var MVP goglmath.Matrix4
	scaleFont := scoreHeight
	scoreY := scoreTop - scaleFont

	game.setOrtho(&MVP)
	MVP.Translate(game.minX, scoreY, 0, 1)
	MVP.Scale(scaleFont, scaleFont, 1, 1)
	glc.UniformMatrix4fv(game.texMVP, MVP.Data())
	game.scoreOur.draw()

	game.setOrtho(&MVP)
	MVP.Translate(0, scoreY, 0, 1) // FIXME coord X
	MVP.Scale(scaleFont, scaleFont, 1, 1)
	glc.UniformMatrix4fv(game.texMVP, MVP.Data())
	game.scoreTheir.draw()

	// clean-up

	glc.DisableVertexAttribArray(game.texPosition)
	glc.DisableVertexAttribArray(game.texTextureCoord)

	glc.Disable(gl.BLEND)
}

func (game *gameState) drawImage(tex gl.Texture, x, y, width, height float64) {
	glc := game.gl // shortcut

	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareElemData)
	glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufSquareElemIndex)

	// square geometry
	elemFirst := 0
	elemCount := squareElemIndexCount // 6
	elemType := gl.Enum(gl.UNSIGNED_INT)
	elemSize := 4

	strideSize := 5 * 4 // 5 x 4 bytes
	itemsPosition := 3
	itemsTexture := 2
	offsetPosition := 0
	offsetTexture := itemsPosition * 4 // 3 x 4 bytes

	glc.VertexAttribPointer(game.texPosition, itemsPosition, gl.FLOAT, false, strideSize, offsetPosition)
	glc.VertexAttribPointer(game.texTextureCoord, itemsTexture, gl.FLOAT, false, strideSize, offsetTexture)

	var MVP goglmath.Matrix4
	game.setOrtho(&MVP)
	MVP.Translate(x, y, 0, 1)
	MVP.Scale(width, height, 1, 1)
	glc.UniformMatrix4fv(game.texMVP, MVP.Data())

	glc.BindTexture(gl.TEXTURE_2D, tex)
	glc.DrawElements(gl.TRIANGLES, elemCount, elemType, elemFirst*elemSize)
}
