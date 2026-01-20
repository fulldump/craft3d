package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const (
	width  = 800
	height = 600
	title  = "Craft3D (OpenGL)"
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
		void main() {
			frag_colour = texture(tex, fragTexCoord);
		}
	` + "\x00"
)

// X, Y, Z, U, V
var cubeVertices = []float32{
	// Front face
	-0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, -0.5, 0.5, 1.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 1.0,
	-0.5, 0.5, 0.5, 0.0, 1.0,
	// Back face
	-0.5, -0.5, -0.5, 1.0, 0.0,
	0.5, -0.5, -0.5, 0.0, 0.0,
	0.5, 0.5, -0.5, 0.0, 1.0,
	-0.5, 0.5, -0.5, 1.0, 1.0,
	// Top face
	-0.5, 0.5, 0.5, 0.0, 0.0,
	0.5, 0.5, 0.5, 1.0, 0.0,
	0.5, 0.5, -0.5, 1.0, 1.0,
	-0.5, 0.5, -0.5, 0.0, 1.0,
	// Bottom face
	-0.5, -0.5, 0.5, 0.0, 1.0,
	0.5, -0.5, 0.5, 1.0, 1.0,
	0.5, -0.5, -0.5, 1.0, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0,
	// Right face
	0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, -0.5, -0.5, 1.0, 0.0,
	0.5, 0.5, -0.5, 1.0, 1.0,
	0.5, 0.5, 0.5, 0.0, 1.0,
	// Left face
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

// Global state for interaction
var (
	cameraDistance = 3.0
	rotationX      = 0.0
	rotationY      = 0.0
	lastMouseX     = 0.0
	lastMouseY     = 0.0
	dragging       = false
)

func main() {
	runtime.LockOSThread()

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

	// Callbacks
	window.SetScrollCallback(scrollCallback)
	window.SetMouseButtonCallback(mouseButtonCallback)
	window.SetCursorPosCallback(cursorPosCallback)

	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	// Disable VSync to allow unlimited FPS
	glfw.SwapInterval(0)

	// Compile Shaders
	program, err := newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		panic(err)
	}
	gl.UseProgram(program)

	// Uniforms
	mvpUniform := gl.GetUniformLocation(program, gl.Str("mvp\x00"))
	texUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1i(texUniform, 0) // Texture unit 0

	// Texture
	texture, err := newTexture()
	if err != nil {
		panic(err)
	}
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	// VAO / VBO / EBO
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	var ebo uint32
	gl.GenBuffers(1, &ebo)
	gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, ebo)
	gl.BufferData(gl.ELEMENT_ARRAY_BUFFER, len(cubeIndices)*4, gl.Ptr(cubeIndices), gl.STATIC_DRAW)

	// Attributes (Stride = 5 * 4 bytes)
	stride := int32(5 * 4)

	// Position (offset 0)
	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vp\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))

	// TexCoord (offset 3*4 = 12)
	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, stride, gl.PtrOffset(12))

	// Global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(0.1, 0.1, 0.1, 1.0)

	// Projection Matrix
	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(width)/float32(height), 0.1, 100.0)

	lastFpsTime := glfw.GetTime()
	frameCount := 0

	for !window.ShouldClose() {
		currentTime := glfw.GetTime()

		// FPS Counter Update (every 1 second)
		frameCount++
		if currentTime-lastFpsTime >= 1.0 {
			window.SetTitle(fmt.Sprintf("%s | FPS: %d", title, frameCount))
			frameCount = 0
			lastFpsTime = currentTime
		}

		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		gl.UseProgram(program)

		// Camera
		camera := mgl32.LookAtV(
			mgl32.Vec3{0, 0, float32(cameraDistance)},
			mgl32.Vec3{0, 0, 0},
			mgl32.Vec3{0, 1, 0},
		)

		// Rotation logic
		model := mgl32.HomogRotate3D(float32(rotationX), mgl32.Vec3{1, 0, 0})
		model = model.Mul4(mgl32.HomogRotate3D(float32(rotationY), mgl32.Vec3{0, 1, 0}))

		mvp := projection.Mul4(camera).Mul4(model)
		gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])

		gl.BindVertexArray(vao)
		gl.DrawElements(gl.TRIANGLES, int32(len(cubeIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))

		window.SwapBuffers()
		glfw.PollEvents()
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

func mouseButtonCallback(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	if button == glfw.MouseButtonLeft {
		if action == glfw.Press {
			dragging = true
			lastMouseX, lastMouseY = w.GetCursorPos()
		} else {
			dragging = false
		}
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
	// Checkerboard
	for x := 0; x < 64; x++ {
		for y := 0; y < 64; y++ {
			if (x/8+y/8)%2 == 0 {
				rgba.Set(x, y, color.White)
			} else {
				// Gray
				rgba.Set(x, y, color.Gray{Y: 128})
			}
		}
	}

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

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

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

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

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}
