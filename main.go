package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	width  = 800
	height = 600
	title  = "Craft3D (Hotbar)"
)

var (
	vertexShaderSource = `
		#version 410
		in vec3 vp;
		in vec2 vertTexCoord;
		out vec2 fragTexCoord;
		uniform mat4 mvp;
		void main() {
			fragTexCoord = vertTexCoord;
			gl_Position = mvp * vec4(vp, 1.0);
		}
	` + "\x00"

	fragmentShaderSource = `
		#version 410
		in vec2 fragTexCoord;
		out vec4 frag_colour;
		uniform sampler2D tex;
		uniform vec4 colorTint; 
		void main() {
			vec4 texColor = texture(tex, fragTexCoord);
			frag_colour = texColor * colorTint;
		}
	` + "\x00"
)

// X, Y, Z, U, V
var cubeVertices = []float32{
	// Front
	-0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, -0.5, 0.5, 1.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1.0,
	// Back
	-0.5, -0.5, -0.5, 1.0, 0.0,
	0.5, -0.5, -0.5, 0.0, 0.0,
	0.5, 0.5, -0.5, 0.0, 1.0,
	-0.5, 0.5, -0.5, 1.0, 1.0,
	// Top
	-0.5, 0.5, 0.5, 0.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 0.0,
	0.5, 0.5, -0.5, 1.0, 1.0,
	-0.5, 0.5, -0.5, 0.0, 1.0,
	// Bottom
	-0.5, -0.5, 0.5, 0.0, 1.0,
	0.5, -0.5, 0.5, 1.0, 1.0,
	0.5, -0.5, -0.5, 1.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0,
	// Right
	0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, -0.5, -0.5, 1.0, 0.0,
	0.5, 0.5, -0.5, 1.0, 1.0,
	0.5, 0.5, 0.5, 0.0, 1.0,
	// Left
	-0.5, -0.5, 0.5, 1.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0,
	-0.5, 0.5, -0.5, 0.0, 1.0,
	-0.5, 0.5, 0.5, 1.0, 1.0,
}

var cubeIndices = []uint32{
	0, 1, 2, 2, 3, 0, // Front
	4, 5, 6, 6, 7, 4, // Back
	8, 9, 10, 10, 11, 8, // Top
	12, 13, 14, 14, 15, 12, // Bottom
	16, 17, 18, 18, 19, 16, // Right
	20, 21, 22, 22, 23, 20, // Left
}

// 2D Quad for UI
var quadVertices = []float32{
	// x, y, z, u, v
	0.0, 0.0, 0.0, 0.0, 0.0,
	1.0, 0.0, 0.0, 1.0, 0.0,
	1.0, 1.0, 0.0, 1.0, 1.0,
	0.0, 1.0, 0.0, 0.0, 1.0,
}
var quadIndices = []uint32{0, 1, 2, 2, 3, 0}

type BlockPos struct {
	X, Y, Z int
}

var (
	cameraDistance = 5.0
	rotationX      = 0.0
	rotationY      = 0.0
	lastMouseX     = 0.0
	lastMouseY     = 0.0
	dragging       = false

	// Map stores Type ID (1-based)
	blocks = make(map[BlockPos]int)

	currentBlockType = 1
	blockColors      = []mgl32.Vec4{
		{1.0, 1.0, 1.0, 1.0}, // 1: White
		{1.0, 0.2, 0.2, 1.0}, // 2: Red
		{0.2, 1.0, 0.2, 1.0}, // 3: Green
		{0.2, 0.2, 1.0, 1.0}, // 4: Blue
		{1.0, 1.0, 0.0, 1.0}, // 5: Yellow
	}
)

func main() {
	runtime.LockOSThread()
	blocks[BlockPos{0, 0, 0}] = 1 // Start with white block

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	window.SetScrollCallback(scrollCallback)
	window.SetMouseButtonCallback(mouseButtonCallback)
	window.SetCursorPosCallback(cursorPosCallback)
	window.SetKeyCallback(keyCallback)

	if err := gl.Init(); err != nil {
		panic(err)
	}
	glfw.SwapInterval(0)

	program, err := newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		panic(err)
	}
	gl.UseProgram(program)

	// Texture
	texture, err := newTexture()
	if err != nil {
		panic(err)
	}
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.Uniform1i(gl.GetUniformLocation(program, gl.Str("tex\x00")), 0)

	// Cube Mesh
	var vaoCube, vboCube, eboCube uint32
	gl.GenVertexArrays(1, &vaoCube)
	gl.BindVertexArray(vaoCube)
	gl.GenBuffers(1, &vboCube)
	gl.BindBuffer(gl.ARRAY_BUFFER, vboCube)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)
	gl.GenBuffers(1, &eboCube)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, eboCube)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(cubeIndices)*4, gl.Ptr(cubeIndices), gl.STATIC_DRAW)

	stride := int32(5 * 4)
	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vp\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, stride, gl.PtrOffset(12))

	// UI Quad Mesh
	var vaoQuad, vboQuad, eboQuad uint32
	gl.GenVertexArrays(1, &vaoQuad)
	gl.BindVertexArray(vaoQuad)
	gl.GenBuffers(1, &vboQuad)
	gl.BindBuffer(gl.ARRAY_BUFFER, vboQuad)
	gl.BufferData(gl.ARRAY_BUFFER, len(quadVertices)*4, gl.Ptr(quadVertices), gl.STATIC_DRAW)
	gl.GenBuffers(1, &eboQuad)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, eboQuad)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(quadIndices)*4, gl.Ptr(quadIndices), gl.STATIC_DRAW)

	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, stride, gl.PtrOffset(12))

	// Shared Uniforms
	mvpUniform := gl.GetUniformLocation(program, gl.Str("mvp\x00"))
	tintUniform := gl.GetUniformLocation(program, gl.Str("colorTint\x00"))

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.CULL_FACE)
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)

	projection3D := mgl32.Perspective(mgl32.DegToRad(45.0), float32(width)/float32(height), 0.1, 100.0)
	projection2D := mgl32.Ortho(0, float32(width), 0, float32(height), -1, 1)

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(program)

		// --- 3D Pass ---
		gl.Enable(gl.DEPTH_TEST)
		gl.BindVertexArray(vaoCube)

		camera := mgl32.LookAtV(mgl32.Vec3{0, 0, float32(cameraDistance)}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
		viewRotation := mgl32.HomogRotate3D(float32(rotationX), mgl32.Vec3{1, 0, 0}).Mul4(mgl32.HomogRotate3D(float32(rotationY), mgl32.Vec3{0, 1, 0}))
		vp := projection3D.Mul4(camera.Mul4(viewRotation))

		for pos, typeID := range blocks {
			if typeID < 1 || typeID > len(blockColors) {
				typeID = 1
			}
			color := blockColors[typeID-1]
			gl.Uniform4fv(tintUniform, 1, &color[0])

			model := mgl32.Translate3D(float32(pos.X), float32(pos.Y), float32(pos.Z))
			mvp := vp.Mul4(model)
			gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, int32(len(cubeIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
		}

		// --- 2D UI Pass (Hotbar) ---
		gl.Disable(gl.DEPTH_TEST)
		gl.BindVertexArray(vaoQuad)

		// Draw 5 squares at bottom
		boxSize := float32(50.0)
		padding := float32(10.0)
		totalWidth := (boxSize * 5) + (padding * 4)
		startX := (float32(width) - totalWidth) / 2
		startY := float32(20.0)

		for i := 0; i < 5; i++ {
			typeID := i + 1
			color := blockColors[i]

			// Highlight selection
			scale := float32(1.0)
			if typeID == currentBlockType {
				scale = 1.2
				// Make selected slightly brighter or just rely on scale?
				// We already simply use base color. Scale is good.
			}

			gl.Uniform4fv(tintUniform, 1, &color[0])

			x := startX + float32(i)*(boxSize+padding)
			y := startY

			// Center scaling
			if scale > 1.0 {
				diff := (boxSize*scale - boxSize) / 2
				x -= diff
				y -= diff
			}

			model := mgl32.Translate3D(x, y, 0).Mul4(mgl32.Scale3D(boxSize*scale, boxSize*scale, 1))
			mvp := projection2D.Mul4(model)
			gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, int32(len(quadIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
		}

		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func performRaycast(w *glfw.Window) {
	xpos, ypos := w.GetCursorPos()
	ndcX := (float32(xpos)/float32(width))*2 - 1
	ndcY := 1 - (float32(ypos)/float32(height))*2

	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(width)/float32(height), 0.1, 100.0)
	camera := mgl32.LookAtV(mgl32.Vec3{0, 0, float32(cameraDistance)}, mgl32.Vec3{0, 0, 0}, mgl32.Vec3{0, 1, 0})
	viewRotation := mgl32.HomogRotate3D(float32(rotationX), mgl32.Vec3{1, 0, 0}).Mul4(mgl32.HomogRotate3D(float32(rotationY), mgl32.Vec3{0, 1, 0}))
	invVP := projection.Mul4(camera.Mul4(viewRotation)).Inv()

	start := invVP.Mul4x1(mgl32.Vec4{ndcX, ndcY, -1, 1})
	start = start.Mul(1.0 / start.W())
	end := invVP.Mul4x1(mgl32.Vec4{ndcX, ndcY, 1, 1})
	end = end.Mul(1.0 / end.W())

	rayOrigin := mgl32.Vec3{start.X(), start.Y(), start.Z()}
	rayDir := mgl32.Vec3{end.X() - start.X(), end.Y() - start.Y(), end.Z() - start.Z()}.Normalize()

	step := 0.05
	dist := 0.0
	maxDist := 100.0

	for dist < maxDist {
		point := rayOrigin.Add(rayDir.Mul(float32(dist)))
		bx, by, bz := int(math.Round(float64(point.X()))), int(math.Round(float64(point.Y()))), int(math.Round(float64(point.Z())))
		hitPos := BlockPos{bx, by, bz}

		if _, exists := blocks[hitPos]; exists {
			if w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press {
				delete(blocks, hitPos)
				return // Destroyed
			} else if w.GetMouseButton(glfw.MouseButtonRight) == glfw.Press {
				prevPoint := rayOrigin.Add(rayDir.Mul(float32(dist - step*2)))
				nx, ny, nz := int(math.Round(float64(prevPoint.X()))), int(math.Round(float64(prevPoint.Y()))), int(math.Round(float64(prevPoint.Z())))
				newPos := BlockPos{nx, ny, nz}
				if _, exists := blocks[newPos]; !exists {
					blocks[newPos] = currentBlockType // Use selected type
				}
				return // Placed
			}
		}
		dist += step
	}
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		if key >= glfw.Key1 && key <= glfw.Key5 {
			currentBlockType = int(key - glfw.Key1 + 1)
		}
	}
}

func mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if action == glfw.Press {
		if button == glfw.MouseButtonLeft || button == glfw.MouseButtonRight {
			// Try Raycast first
			// Optimization: Don't raycast if clicking on UI? (For now ignore UI clicks)
			performRaycast(w)

			// If Left click didn't hit anything, it becomes drag
			// But for now let's keep it simple: Click is logic + Drag starts
			if button == glfw.MouseButtonLeft {
				dragging = true
				lastMouseX, lastMouseY = w.GetCursorPos()
			}
		}
	} else if action == glfw.Release {
		dragging = false
	}
}

func scrollCallback(w *glfw.Window, xoff float64, yoff float64) {
	cameraDistance -= yoff * 0.5
	if cameraDistance < 1.0 {
		cameraDistance = 1.0
	}
	if cameraDistance > 20.0 {
		cameraDistance = 20.0
	}
}

func cursorPosCallback(w *glfw.Window, xpos float64, ypos float64) {
	if dragging {
		dx := xpos - lastMouseX
		dy := ypos - lastMouseY
		rotationY += dx * 0.01
		rotationX += dy * 0.01
		lastMouseX = xpos
		lastMouseY = ypos
	}
}

func newTexture() (uint32, error) {
	rgba := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			// Make texture white/gray so it multiplies well with Color Tint
			if (x/8+y/8)%2 == 0 {
				rgba.Set(x, y, color.White)
			} else {
				// Lighter gray for better tinting
				rgba.Set(x, y, color.Gray{Y: 200})
			}
		}
	}
	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, 64, 64, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))
	return texture, nil
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}
	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)
	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)
		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))
		return 0, fmt.Errorf("compile error: %v", log)
	}
	return shader, nil
}
