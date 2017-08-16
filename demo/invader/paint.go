// +build darwin linux windows

package main

import (
	"log"
	"time"

	"github.com/udhos/goglmath"

	"golang.org/x/mobile/gl"

	"github.com/udhos/fugo/future"
)

func (game *gameState) paint() {
	glc := game.gl // shortcut

	elap := time.Since(game.updateLast)

	glc.Clear(gl.COLOR_BUFFER_BIT) // draw ClearColor background

	glc.UseProgram(game.program)
	glc.EnableVertexAttribArray(game.position)

	glc.Uniform4f(game.color, .5, .9, .5, 1) // green

	screenWidth := game.maxX - game.minX

	//buttonWidth := screenWidth / float64(buttons)
	buttonWidth := game.buttonEdge()
	//buttonHeight := .2 * (game.maxY - game.minY)
	buttonHeight := buttonWidth

	// clamp height
	maxH := .3 * (game.maxY - game.minY)
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
	fuelHeight := .04

	// Wire rectangle around fuel bar
	//squareWireMVP := goglmath.NewMatrix4Identity()
	var squareWireMVP goglmath.Matrix4
	game.setOrtho(&squareWireMVP)
	squareWireMVP.Translate(game.minX, fuelBottom, .1, 1) // z=.1 put in front of fuel bar
	squareWireMVP.Scale(screenWidth, fuelHeight, 1, 1)
	glc.UniformMatrix4fv(game.P, squareWireMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareWire)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.LINE_LOOP, 0, squareWireVertexCount)

	// Fuel bar
	glc.Uniform4f(game.color, .9, .9, .9, 1) // white
	//squareMVP := goglmath.NewMatrix4Identity()
	var squareMVP goglmath.Matrix4
	game.setOrtho(&squareMVP)
	squareMVP.Translate(game.minX, fuelBottom, 0, 1)
	fuel := float64(future.Fuel(game.playerFuel, elap))
	squareMVP.Scale(screenWidth*fuel/10, fuelHeight, 1, 1) // width is fuel
	glc.UniformMatrix4fv(game.P, squareMVP.Data())
	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
	glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
	glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)

	cannonWidth := .1  // 10%
	cannonHeight := .1 // 10%

	cannonBottom := fuelBottom + fuelHeight + .01

	// Cannons
	for _, can := range game.cannons {
		if can.Player {
			glc.Uniform4f(game.color, .2, .2, .8, 1) // blue
		} else {
			//glc.Uniform4f(game.color, .9, .2, .2, 1) // red
			glc.Uniform4f(game.color, .5, .9, .5, 1) // green
		}

		var canBuf gl.Buffer
		var y float64
		if can.Team == game.playerTeam {
			// upward
			y = cannonBottom
			canBuf = game.bufCannon
		} else {
			// downward
			y = game.maxY
			canBuf = game.bufCannonDown
		}
		var MVP goglmath.Matrix4
		//goglmath.SetOrthoMatrix(&MVP, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
		game.setOrtho(&MVP)
		cannonX, _ := future.CannonX(can.CoordX, can.Speed, elap)
		x := float64(cannonX)*(game.maxX-cannonWidth-game.minX) + game.minX
		MVP.Translate(x, y, 0, 1)
		MVP.Scale(cannonWidth, cannonHeight, 1, 1) // 10% size
		glc.UniformMatrix4fv(game.P, MVP.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, canBuf)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.TRIANGLES, 0, cannonVertexCount)
	}

	missileBottom := cannonBottom + cannonHeight
	missileWidth := .03
	missileHeight := .07

	// Missiles
	glc.Uniform4f(game.color, .9, .9, .4, 1) // yellow
	for _, miss := range game.missiles {
		//missileMVP := goglmath.NewMatrix4Identity()
		var missileMVP goglmath.Matrix4
		//goglmath.SetOrthoMatrix(&missileMVP, game.minX, game.maxX, game.minY, game.maxY, -1, 1)
		game.setOrtho(&missileMVP)
		minX := game.minX + .5*cannonWidth - .5*missileWidth
		maxX := game.maxX - .5*cannonWidth - .5*missileWidth
		x := float64(miss.CoordX)*(maxX-minX) + minX
		y := float64(future.MissileY(miss.CoordY, miss.Speed, elap))
		if miss.Team == game.playerTeam {
			// upward
			minY := missileBottom
			maxY := game.maxY - missileHeight
			y = y*(maxY-minY) + minY
		} else {
			// downward
			minY := cannonBottom
			maxY := game.maxY - cannonHeight
			y = y*(minY-maxY) + maxY

		}
		missileMVP.Translate(x, y, 0, 1)
		missileMVP.Scale(missileWidth, missileHeight, 1, 1)
		glc.UniformMatrix4fv(game.P, missileMVP.Data())
		glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquare)
		glc.VertexAttribPointer(game.position, coordsPerVertex, gl.FLOAT, false, 0, 0)
		glc.DrawArrays(gl.TRIANGLES, 0, squareVertexCount)
	}

	glc.DisableVertexAttribArray(game.position)

	game.paintTex(glc) // another shader
}

func (game *gameState) paintTex(glc gl.Context) {

	glc.UseProgram(game.programTex)
	glc.EnableVertexAttribArray(game.texPosition)
	glc.EnableVertexAttribArray(game.texTextureCoord)

	//MVP := goglmath.NewMatrix4Identity()
	var MVP goglmath.Matrix4
	game.setOrtho(&MVP)
	scale := .5
	MVP.Scale(scale, scale, 1, 1)
	glc.UniformMatrix4fv(game.texMVP, MVP.Data())

	strideSize := 5 * 4 // 5 x 4 bytes
	itemsPosition := 3
	itemsTexture := 2
	offsetPosition := 0
	offsetTexture := itemsPosition * 4 // 3 x 4 bytes
	glc.VertexAttribPointer(game.texPosition, itemsPosition, gl.FLOAT, false, strideSize, offsetPosition)
	glc.VertexAttribPointer(game.texTextureCoord, itemsTexture, gl.FLOAT, false, strideSize, offsetTexture)

	unit := 0
	glc.ActiveTexture(gl.TEXTURE0 + gl.Enum(unit))
	glc.BindTexture(gl.TEXTURE_2D, game.texTexture)
	glc.Uniform1i(game.texSampler, unit)

	glc.BindBuffer(gl.ARRAY_BUFFER, game.bufSquareElemData)
	glc.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, game.bufSquareElemIndex)

	elemFirst := 0
	elemCount := squareElemIndexCount // 6
	elemType := gl.Enum(gl.UNSIGNED_INT)
	elemSize := 4
	glc.DrawElements(gl.TRIANGLES, elemCount, elemType, elemFirst*elemSize)

	if status := glc.CheckFramebufferStatus(gl.FRAMEBUFFER); status != gl.FRAMEBUFFER_COMPLETE {
		log.Printf("paintTex: bad framebuffer status: %d", status)
	}

	glc.DisableVertexAttribArray(game.texPosition)
	glc.DisableVertexAttribArray(game.texTextureCoord)
}
