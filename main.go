package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"
	"os"
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
	6, 5, 4, 4, 7, 6, // Back
	8, 9, 10, 10, 11, 8, // Top
	14, 13, 12, 12, 15, 14, // Bottom
	16, 17, 18, 18, 19, 16, // Right
	22, 21, 20, 20, 23, 22, // Left
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

type Player struct {
	Position mgl32.Vec3
	Velocity mgl32.Vec3
	Yaw      float64
	Pitch    float64
	OnGround bool
	IsDead   bool
}

const (
	gravity       = 25.0
	jumpSpeed     = 8.0
	moveSpeed     = 10.0
	rotationSpeed = 100.0
)

var (
	player = Player{
		Position: mgl32.Vec3{0, 10, 0}, // Start higher to avoid terrain
		Velocity: mgl32.Vec3{0, 0, 0},
		Yaw:      -90.0, // Face forward (-Z)
		Pitch:    0.0,
	}

	lastMouseX = 0.0
	lastMouseY = 0.0
	firstMouse = true

	// Map stores Type ID (1-based)
	blocks = make(map[BlockPos]int)

	currentBlockType = 1
	blockColors      = []mgl32.Vec4{
		{1.0, 1.0, 1.0, 1.0}, // 1: Sand
		{1.0, 1.0, 1.0, 1.0}, // 2: Rock
		{1.0, 1.0, 1.0, 1.0}, // 3: Grass
		{1.0, 1.0, 1.0, 1.0}, // 4: Dirt
		{1.0, 1.0, 1.0, 1.0}, // 5: Water
	}

	gameOverTexture uint32
	texDirt         uint32
	texGrassTop     uint32
	texGrassSide    uint32
	texRock         uint32
	texWater        uint32
	texSand         uint32
)

func main() {
	fmt.Println("LOLOLOL")

	runtime.LockOSThread()
	runtime.LockOSThread()

	// Generate Irregular Terrain
	waterLevel := -1

	for x := -50; x <= 50; x++ {
		for z := -50; z <= 50; z++ {
			// Simple wave function
			h := int(float64(4.0) * (math.Sin(float64(x)*0.1) + math.Cos(float64(z)*0.1)))

			// Fill from bottom up to height
			for y := -5; y <= h; y++ {
				// Determine color based on height
				blockType := 1 // Sand default
				if y == h {
					// Surface block
					if y <= waterLevel+1 {
						blockType = 1 // Sand (Shore/Seabed)
					} else {
						blockType = 3 // Grass
					}
				} else if y < -2 {
					blockType = 2 // Rock deep down
				} else {
					blockType = 4 // Dirt in between
				}
				blocks[BlockPos{x, y, z}] = blockType
			}

			// Fill Water
			for y := h + 1; y <= waterLevel; y++ {
				blocks[BlockPos{x, y, z}] = 5 // Water
			}
		}
	}

	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.True)
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
	// Enable VSync
	glfw.SwapInterval(1)

	program, err := newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		panic(err)
	}
	gl.UseProgram(program)

	// Texture
	// texture, err := newTexture() // Generated texture unused
	// if err != nil {
	// 	panic(err)
	// }

	// Load Textures
	texDirt, err = loadTexture("dirt.png")
	if err != nil {
		fmt.Printf("Failed to load dirt.png: %v\n", err)
	}
	texGrassTop, err = loadTexture("grass_top.png")
	if err != nil {
		fmt.Printf("Failed to load grass_top.png: %v\n", err)
	}
	texGrassSide, err = loadTexture("grass_side.png")
	if err != nil {
		fmt.Printf("Failed to load grass_side.png: %v\n", err)
	}
	texRock, err = loadTexture("rock.png")
	if err != nil {
		fmt.Printf("Failed to load rock.png: %v\n", err)
	}
	texWater, err = loadTexture("water.png")
	if err != nil {
		fmt.Printf("Failed to load water.png: %v\n", err)
	}
	texSand, err = loadTexture("sand.png")
	if err != nil {
		fmt.Printf("Failed to load sand.png: %v\n", err)
	}

	// Game Over Texture
	gameOverTexture, err = loadTexture("game_over.png")
	if err != nil {
		// Fallback or panic? Let's log and proceed or panic.
		fmt.Printf("Failed to load game_over.png: %v\n", err)
	}

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texDirt)
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
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	gl.Enable(gl.BLEND)                  // Enable blending globally
	gl.ClearColor(0.53, 0.81, 0.92, 1.0) // Sky Blue

	// Time tracking
	lastTime := glfw.GetTime()
	frameCount := 0

	for !window.ShouldClose() {
		// Dynamic Window Size
		fbWidth, fbHeight := window.GetFramebufferSize()
		gl.Viewport(0, 0, int32(fbWidth), int32(fbHeight))

		projection3D := mgl32.Perspective(mgl32.DegToRad(45.0), float32(fbWidth)/float32(fbHeight), 0.1, 100.0)
		projection2D := mgl32.Ortho(0, float32(fbWidth), 0, float32(fbHeight), -1, 1)

		currentTime := glfw.GetTime()
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.UseProgram(program)

		// --- 3D Pass ---
		// --- Physics & Movement ---
		dt := currentTime - lastTime
		if dt > 0.1 {
			dt = 0.1 // Cap dt to avoid large jumps
		}

		// Death Check
		if player.Position.Y() < -100.0 && !player.IsDead {
			player.IsDead = true
		}

		// Input Handling
		// Convert Yaw to Radians for math.Cos/Sin (takes float64)
		radYaw := player.Yaw * (math.Pi / 180.0)
		radPitch := player.Pitch * (math.Pi / 180.0)

		forward := mgl32.Vec3{
			float32(math.Cos(radYaw)),
			0,
			float32(math.Sin(radYaw)),
		}
		// right vector removed as unused for input

		// Horizontal Movement
		vel := mgl32.Vec3{0, 0, 0}
		if !player.IsDead {
			if window.GetKey(glfw.KeyW) == glfw.Press {
				vel = vel.Add(forward)
			}
			if window.GetKey(glfw.KeyS) == glfw.Press {
				vel = vel.Sub(forward)
			}
			if window.GetKey(glfw.KeyD) == glfw.Press {
				player.Yaw += rotationSpeed * float64(dt)
			}
			if window.GetKey(glfw.KeyA) == glfw.Press {
				player.Yaw -= rotationSpeed * float64(dt)
			}
		} else {
			// Stop horizontal movement when dead
			player.Velocity = mgl32.Vec3{0, player.Velocity.Y(), 0}
		}

		if vel.Len() > 0 {
			vel = vel.Normalize().Mul(moveSpeed)
			player.Velocity = mgl32.Vec3{vel.X(), player.Velocity.Y(), vel.Z()}
		} else if !player.IsDead {
			player.Velocity = mgl32.Vec3{0, player.Velocity.Y(), 0}
		}

		// Jumping
		if !player.IsDead && window.GetKey(glfw.KeySpace) == glfw.Press && player.OnGround {
			player.Velocity = mgl32.Vec3{player.Velocity.X(), jumpSpeed, player.Velocity.Z()}
			player.OnGround = false
		}

		// Gravity
		player.Velocity = player.Velocity.Sub(mgl32.Vec3{0, gravity * float32(dt), 0})

		// Axis Separated Movement & Collision
		// Y Axis
		player.Position = player.Position.Add(mgl32.Vec3{0, player.Velocity.Y() * float32(dt), 0})
		if checkCollision(player.Position) {
			if player.Velocity.Y() < 0 {
				// Falling down, hit floor
				player.OnGround = true
				// Snap to block top?
				// Simple snap: Round position to nearest integer or just step back?
				// To start simple: just undo movement
				// But undoing movement is jittery.
				// Proper way: Find the block we hit and snap to its top.
				// But we are intersecting potentially multiple blocks.
				// Let's iterate find the highest block floor we are in?
				// Doing simple iterative resolution:
				// If we are collided, move back until not collided? No.

				// Let's assume integer grid.
				// Player Height 2. Width 1. Depth 1.
				// Feet at Y. Top at Y+2.
				// If we hit something below, our Y is slightly inside a block.
				// Floor is at int(Y) or similar.
				// For now: reset Y to previous Y (simpler but might get stuck)
				// Or better: round Y up to nearest integer if falling?
				// Blocks are at integer coordinates. Top surface at BlockY + 0.5.

				// Revert Y
				player.Position = player.Position.Sub(mgl32.Vec3{0, player.Velocity.Y() * float32(dt), 0})

				// Snap Y to nice value?
				// If we hit floor at Y=-0.5, our feet should be at -0.5.
				// For now let's just zero velocity and revert position.
				// Improving logic:
				// Snap to top of the block we hit. Which block?
				// checkCollision just returns bool.
				// Let's just Stop.
				player.Velocity = mgl32.Vec3{player.Velocity.X(), 0, player.Velocity.Z()}
			} else if player.Velocity.Y() > 0 {
				// Jumping up, hit ceiling
				player.Position = player.Position.Sub(mgl32.Vec3{0, player.Velocity.Y() * float32(dt), 0})
				player.Velocity = mgl32.Vec3{player.Velocity.X(), 0, player.Velocity.Z()}
			}
		} else {
			player.OnGround = false
		}

		// X Axis
		player.Position = player.Position.Add(mgl32.Vec3{player.Velocity.X() * float32(dt), 0, 0})
		if checkCollision(player.Position) {
			player.Position = player.Position.Sub(mgl32.Vec3{player.Velocity.X() * float32(dt), 0, 0})
			player.Velocity = mgl32.Vec3{0, player.Velocity.Y(), player.Velocity.Z()}
		}

		// Z Axis
		player.Position = player.Position.Add(mgl32.Vec3{0, 0, player.Velocity.Z() * float32(dt)})
		if checkCollision(player.Position) {
			player.Position = player.Position.Sub(mgl32.Vec3{0, 0, player.Velocity.Z() * float32(dt)})
			player.Velocity = mgl32.Vec3{player.Velocity.X(), player.Velocity.Y(), 0}
		}

		// --- 3D Pass ---
		gl.Enable(gl.DEPTH_TEST)
		gl.BindVertexArray(vaoCube)

		// Create Camera Matrix
		// Camera at Position + EyeOffset (0, 1.5, 0)
		eyePos := player.Position.Add(mgl32.Vec3{0, 1.5, 0})

		// Look Direction
		// Reuse radYaw/radPitch
		front := mgl32.Vec3{
			float32(math.Cos(radPitch) * math.Cos(radYaw)),
			float32(math.Sin(radPitch)),
			float32(math.Cos(radPitch) * math.Sin(radYaw)),
		}

		camera := mgl32.LookAtV(eyePos, eyePos.Add(front), mgl32.Vec3{0, 1, 0})
		vp := projection3D.Mul4(camera)

		// Two passes: Opaque then Transparent (Water)
		// Or just render water last.
		// Since we iterate map, order is random.
		// Construct lists first.
		var opaqueBlocks []BlockPos
		var waterBlocks []BlockPos

		for pos, typeID := range blocks {
			_ = typeID
			// if typeID == 5 && false { // Water
			// 	waterBlocks = append(waterBlocks, pos)
			// } else {
			// 	opaqueBlocks = append(opaqueBlocks, pos)
			// }
			opaqueBlocks = append(opaqueBlocks, pos)

		}

		drawBlock := func(pos BlockPos, typeID int) {
			if typeID < 1 || typeID > len(blockColors) {
				typeID = 1
			}

			model := mgl32.Translate3D(float32(pos.X), float32(pos.Y), float32(pos.Z))
			mvp := vp.Mul4(model)
			gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])

			color := blockColors[typeID-1]
			gl.Uniform4fv(tintUniform, 1, &color[0])

			if typeID == 3 { // Grass (Multi-face)
				// Top (Indices 12-17, but verify offset)
				// Cube Indices:
				// Front: 0-5 (0*6)
				// Back: 6-11 (1*6)
				// Top: 12-17 (2*6)
				// Bottom: 18-23 (3*6)
				// Right: 24-29 (4*6)
				// Left: 30-35 (5*6)

				// Top Face
				gl.BindTexture(gl.TEXTURE_2D, texGrassTop)
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(12*4)) // Start at index 12

				// Bottom Face (Dirt)
				gl.BindTexture(gl.TEXTURE_2D, texDirt)
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(18*4))

				// Sides (Grass Side)
				gl.BindTexture(gl.TEXTURE_2D, texGrassSide)
				// Front
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(0))
				// Back
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(6*4))
				// Right
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(24*4))
				// Left
				gl.DrawElements(gl.TRIANGLES, 6, gl.UNSIGNED_INT, gl.PtrOffset(30*4))

			} else {
				// Standard Block
				var typeTex uint32
				switch typeID {
				case 1:
					typeTex = texSand
				case 2:
					typeTex = texRock
				case 4:
					typeTex = texDirt
				case 5:
					typeTex = texWater
				default:
					typeTex = texDirt
				}
				gl.BindTexture(gl.TEXTURE_2D, typeTex)
				gl.DrawElements(gl.TRIANGLES, int32(len(cubeIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
			}
		}

		// Draw Opaque
		for _, pos := range opaqueBlocks {
			drawBlock(pos, blocks[pos])
		}
		// Draw Water
		// Disable Depth Write for water? Usually helps with transparency against itself, but simple back-to-front is better.
		// We don't have sorting, so just drawing last is best effort.
		for _, pos := range waterBlocks {
			drawBlock(pos, blocks[pos])
		}

		// --- 2D UI Pass (Hotbar & Game Over) ---
		gl.Disable(gl.DEPTH_TEST)
		gl.BindVertexArray(vaoQuad)

		if player.IsDead {
			// 1. Red Overlay
			gl.Enable(gl.BLEND)
			gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

			// Unbind texture or bind white texture for solid color overlay?
			// We can use the shader with a white texture (our generated texture has white pixels).
			// Or just disable texture? Our shader expects texture.
			// Let's bind the standard block texture (it has white pixels) and set tint to Red with Alpha.
			// Let's bind the standard block texture (it has white pixels) and set tint to Red with Alpha.
			gl.BindTexture(gl.TEXTURE_2D, texRock) // Use Rock as background overlay pattern?

			redOverlay := mgl32.Vec4{1.0, 0.0, 0.0, 0.5} // Red, 50% opacity
			gl.Uniform4fv(tintUniform, 1, &redOverlay[0])

			// Full screen quad
			// Ortho is 0 to width, 0 to height.
			model := mgl32.Scale3D(float32(fbWidth), float32(fbHeight), 1)
			mvp := projection2D.Mul4(model)
			gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, int32(len(quadIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))

			// 2. "You Died" Text
			gl.BindTexture(gl.TEXTURE_2D, gameOverTexture)
			whiteTint := mgl32.Vec4{1.0, 1.0, 1.0, 1.0}
			gl.Uniform4fv(tintUniform, 1, &whiteTint[0])

			// Center the text
			// Assume Image aspect ratio? Or just arbitrary size.
			// Let's make it 600 wide (or less if screen smaller)
			imgW := float32(600.0)
			imgH := float32(150.0) // Approx aspect ratio 4:1

			if imgW > float32(fbWidth) {
				imgW = float32(fbWidth) * 0.8
				imgH = imgW / 4.0
			}

			x := (float32(fbWidth) - imgW) / 2
			y := (float32(fbHeight) - imgH) / 2

			model = mgl32.Translate3D(x, y, 0).Mul4(mgl32.Scale3D(imgW, imgH, 1))
			mvp = projection2D.Mul4(model)
			gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])
			gl.DrawElements(gl.TRIANGLES, int32(len(quadIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))

			gl.Disable(gl.BLEND)

			// Rebind standard texture for Hotbar (if we draw it? Maybe hide hotbar on death?)
			// Let's hide hotbar.
		} else {
			// Draw 5 squares at bottom
			// gl.BindTexture(gl.TEXTURE_2D, texture) // Rebind standard texture - NO, bind per icon

			boxSize := float32(50.0)
			padding := float32(10.0)
			totalWidth := (boxSize * 5) + (padding * 4)
			startX := (float32(fbWidth) - totalWidth) / 2
			startY := float32(20.0)

			for i := 0; i < 5; i++ {
				typeID := i + 1
				color := blockColors[i]

				// Highlight selection
				scale := float32(1.0)
				if typeID == currentBlockType {
					scale = 1.2
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

				// Bind Icon Texture
				var iconTex uint32
				switch typeID {
				case 1:
					iconTex = texSand
				case 2:
					iconTex = texRock
				case 3:
					iconTex = texGrassSide // Use side for icon
				case 4:
					iconTex = texDirt
				case 5:
					iconTex = texWater
				}
				gl.BindTexture(gl.TEXTURE_2D, iconTex)

				model := mgl32.Translate3D(x, y, 0).Mul4(mgl32.Scale3D(boxSize*scale, boxSize*scale, 1))
				mvp := projection2D.Mul4(model)
				gl.UniformMatrix4fv(mvpUniform, 1, false, &mvp[0])
				gl.DrawElements(gl.TRIANGLES, int32(len(quadIndices)), gl.UNSIGNED_INT, gl.PtrOffset(0))
			}
		}

		window.SwapBuffers()
		// FPS Counter handled by frame counting
		frameCount++
		if currentTime-lastTime >= 1.0 {
			window.SetTitle(fmt.Sprintf("%s - FPS: %d", title, frameCount))
			frameCount = 0
			lastTime = currentTime
		}

		glfw.PollEvents()
	}
}

func performRaycast(w *glfw.Window) {
	xpos, ypos := w.GetCursorPos()
	fbWidth, fbHeight := w.GetFramebufferSize()

	ndcX := (float32(xpos)/float32(fbWidth))*2 - 1
	ndcY := 1 - (float32(ypos)/float32(fbHeight))*2

	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(fbWidth)/float32(fbHeight), 0.1, 100.0)
	// Raycast Update
	// Need to recalculate View Matrix for raycast.
	eyePos := player.Position.Add(mgl32.Vec3{0, 1.5, 0})

	radYaw := player.Yaw * (math.Pi / 180.0)
	radPitch := player.Pitch * (math.Pi / 180.0)

	front := mgl32.Vec3{
		float32(math.Cos(radPitch) * math.Cos(radYaw)),
		float32(math.Sin(radPitch)),
		float32(math.Cos(radPitch) * math.Sin(radYaw)),
	}
	camera := mgl32.LookAtV(eyePos, eyePos.Add(front), mgl32.Vec3{0, 1, 0})

	invVP := projection.Mul4(camera).Inv()

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

func checkCollision(pos mgl32.Vec3) bool {
	// Player Size: 1 width (0.5 radius), 2 height, 1 depth (0.5 radius)
	// AABB relative to Feet Pos:
	// Min: x-0.5, y, z-0.5
	// Max: x+0.5, y+2, z+0.5
	// Padding? Make it slightly smaller to avoid constant collision?
	// Let's use strict size first.

	minX, minY, minZ := pos.X()-0.4, pos.Y(), pos.Z()-0.4
	maxX, maxY, maxZ := pos.X()+0.4, pos.Y()+2.0, pos.Z()+0.4 // Slightly < 0.5 to fit in 1-wide gaps

	// Iterate blocks in this range
	// Blocks are centered at Integer coordinates (e.g. 0,0,0 covers -0.5 to 0.5)
	// We need to check any block whose bounds [-0.5, 0.5] overlap our [min, max].
	// Overlap condition: BlockCenter - 0.5 < Max AND BlockCenter + 0.5 > Min
	// BlockCenter < Max + 0.5 AND BlockCenter > Min - 0.5
	// Closest Integer Logic: Round(Min) to Round(Max) covers the range effectively?
	// Example: Min 0.4 (Round 0). Max 0.6 (Round 1). Block 0 (ends 0.5) overlaps. Block 1 (starts 0.5) overlaps. Correct.
	// Example: Min 0.1 (Round 0). Max 0.3 (Round 0). Block 0 overlaps. Correct.
	// Example: Min -0.6 (Round -1). Max -0.4 (Round 0). Block -1 overlaps. Block 0 overlaps. Correct.

	startX, startY, startZ := int(math.Round(float64(minX))), int(math.Round(float64(minY))), int(math.Round(float64(minZ)))
	endX, endY, endZ := int(math.Round(float64(maxX))), int(math.Round(float64(maxY))), int(math.Round(float64(maxZ)))

	for x := startX; x <= endX; x++ {
		for y := startY; y <= endY; y++ {
			for z := startZ; z <= endZ; z++ {
				if _, exists := blocks[BlockPos{x, y, z}]; exists {
					return true
				}
			}
		}
	}
	return false
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
			// Raycast
			performRaycast(w)
		}
	}
}

// scrollCallback removed/empty as zoom is not needed
func scrollCallback(w *glfw.Window, xoff float64, yoff float64) {
}

func cursorPosCallback(w *glfw.Window, xpos float64, ypos float64) {
	if firstMouse {
		lastMouseX = xpos
		lastMouseY = ypos
		firstMouse = false
	}

	xoffset := xpos - lastMouseX
	yoffset := lastMouseY - ypos // Reversed since y-coordinates go from bottom to top
	lastMouseX = xpos
	lastMouseY = ypos

	sensitivity := 0.1
	xoffset *= sensitivity
	yoffset *= sensitivity

	player.Yaw += xoffset
	player.Pitch += yoffset

	// Constrain pitch
	if player.Pitch > 89.0 {
		player.Pitch = 89.0
	}
	if player.Pitch < -89.0 {
		player.Pitch = -89.0
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

func loadTexture(path string) (uint32, error) {
	file, err := os.Open("textures/" + path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		// Unsupported stride
		// Just draw
	}
	// Copy image to RGBA
	// Simple loop
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	// Or use draw.Draw? "image/draw"
	// Let's stick to manual or simple for now since we didn't import draw.

	// Flip Y? OpenGL textures are bottom-up?
	// Usually images are top-down.
	// But our Quad UVs are 0,0 (bottom-left?) to 1,1 (top-right)?
	// Let's check Quad UVs.
	// 0,0 -> 0,0
	// 1,0 -> 1,0
	// 1,1 -> 1,1
	// 0,1 -> 0,1
	// Wait,
	// 0,0,0, 0,0
	// 1,0,0, 1,0
	// 1,1,0, 1,1
	// 0,1,0, 0,1
	// OpenGL 0,0 is Bottom-Left.
	// PNG 0,0 is Top-Left.
	// So we might render upside down if not flipped.
	// Let's flip it.
	flipped := image.NewRGBA(img.Bounds())
	for y := 0; y < b.Max.Y; y++ {
		for x := 0; x < b.Max.X; x++ {
			flipped.Set(x, b.Max.Y-y-1, img.At(x, y))
		}
	}
	rgba = flipped

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(rgba.Rect.Size().X), int32(rgba.Rect.Size().Y), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(rgba.Pix))

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
