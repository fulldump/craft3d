package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	width  = 800
	height = 600
)

func main() {
	// Connect to the X server (env var DISPLAY)
	X, err := xgb.NewConn()
	if err != nil {
		log.Fatalf("Cannot connect to X server: %v", err)
	}
	defer X.Close()

	// Get the setup info (root window, screens, etc)
	setup := xproto.Setup(X)
	screen := setup.DefaultScreen(X)

	// Create a Window ID
	wid, _ := xproto.NewWindowId(X)

	// Create the window
	xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
		0, 0, uint16(width), uint16(height), 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{
			0xffffffff, // White background
			xproto.EventMaskStructureNotify | xproto.EventMaskKeyPress, // Listen for map notify and keys
		})

	// Create a Graphics Context (GC) for drawing
	gc, _ := xproto.NewGcontextId(X)
	xproto.CreateGC(X, gc, xproto.Drawable(wid), 0, nil)

	// Map (show) the window
	xproto.MapWindow(X, wid)

	// Set window title
	title := "Craft3D (Software Rasterizer)"
	xproto.ChangeProperty(X, xproto.PropModeReplace, wid, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(title)), []byte(title))

	// Create a software framebuffer (RGBA image)
	fb := image.NewRGBA(image.Rect(0, 0, width, height))

	fmt.Println("Window created! Press ESC or 'q' to quit.")

	// Channel to handle X events without blocking the render loop
	events := make(chan xgb.Event)
	go func() {
		for {
			ev, err := X.WaitForEvent()
			if err != nil {
				log.Printf("Error waiting for event: %v", err)
				continue
			}
			if ev == nil {
				log.Println("Connection closed")
				os.Exit(0)
			}
			events <- ev
		}
	}()

	ticker := time.NewTicker(16 * time.Millisecond) // ~60 FPS
	defer ticker.Stop()

	// --- 3D SETUP ---
	// Define the 8 vertices of a cube
	cubeVertices := []Vec3{
		{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
		{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
	}

	// Define the 12 edges (indices into vertices)
	edges := [][2]int{
		{0, 1}, {1, 2}, {2, 3}, {3, 0}, // Back face
		{4, 5}, {5, 6}, {6, 7}, {7, 4}, // Front face
		{0, 4}, {1, 5}, {2, 6}, {3, 7}, // Connecting lines
	}

	angleX, angleY, angleZ := 0.0, 0.0, 0.0
	// ----------------

	for {
		select {
		case ev := <-events:
			switch e := ev.(type) {
			case xproto.KeyPressEvent:
				if e.Detail == 9 || e.Detail == 24 {
					os.Exit(0)
				}
			case xproto.MapNotifyEvent:
				fmt.Println("Window mapped!")
			}

		case <-ticker.C:
			// Update angles
			angleX += 0.01
			angleY += 0.015
			angleZ += 0.005

			// --- RENDER START ---

			// Clear background to black
			for i := 0; i < len(fb.Pix); i += 4 {
				fb.Pix[i] = 0
				fb.Pix[i+1] = 0
				fb.Pix[i+2] = 0
				fb.Pix[i+3] = 255
			}

			// Project and Draw Cube
			var projectedPoints [8]struct{ x, y int }

			// 1. Rotate and Project all vertices
			for i, v := range cubeVertices {
				rotated := v.RotateX(angleX).RotateY(angleY).RotateZ(angleZ)
				// Scale for visibility
				rotated.X *= 100
				rotated.Y *= 100
				rotated.Z *= 100

				x, y := rotated.Project(width, height, 400, 400) // fov=400, dist=400
				projectedPoints[i] = struct{ x, y int }{x, y}
			}

			// 2. Draw Edges
			lineColor := color.RGBA{255, 255, 0, 255} // Yellow lines
			for _, edge := range edges {
				p1 := projectedPoints[edge[0]]
				p2 := projectedPoints[edge[1]]
				DrawLine(fb, p1.x, p1.y, p2.x, p2.y, lineColor)
			}

			// --- RENDER END ---

			// Blit the framebuffer to the window using PutImage
			// X11 has a maximum request size (often 256KB).
			// Our buffer is 800*600*4 = ~1.9MB.
			// We MUST split the sending into smaller chunks (batches of rows).
			batchHeight := 50 // 800 * 50 * 4 = 160KB < 256KB
			for batchY := 0; batchY < height; batchY += batchHeight {
				h := batchHeight
				if batchY+h > height {
					h = height - batchY
				}

				start := fb.PixOffset(0, batchY)
				end := fb.PixOffset(0, batchY+h)

				xproto.PutImage(X, xproto.ImageFormatZPixmap, xproto.Drawable(wid), gc,
					uint16(width), uint16(h), 0, int16(batchY), 0, screen.RootDepth, fb.Pix[start:end])
			}
		}
	}
}
