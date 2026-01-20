package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
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
	// We need to specify the parent window (root), visual, and geometry
	xproto.CreateWindow(X, screen.RootDepth, wid, screen.Root,
		0, 0, 800, 600, 0,
		xproto.WindowClassInputOutput, screen.RootVisual,
		xproto.CwBackPixel|xproto.CwEventMask,
		[]uint32{ // Values for the masks above
			0xffffffff, // White background
			xproto.EventMaskStructureNotify | xproto.EventMaskKeyPress, // Listen for map notify and keys
		})

	// Map (show) the window
	xproto.MapWindow(X, wid)

	// Set window title
	title := "Craft3D (Pure Go X11)"
	xproto.ChangeProperty(X, xproto.PropModeReplace, wid, xproto.AtomWmName, xproto.AtomString, 8, uint32(len(title)), []byte(title))

	fmt.Println("Window created! Press ESC or 'q' to quit.")

	// Event loop
	for {
		ev, xerr := X.WaitForEvent()
		if xerr != nil {
			log.Printf("Error waiting for event: %v", xerr)
			continue
		}

		if ev == nil {
			log.Println("Connection closed")
			return
		}

		switch e := ev.(type) {
		case xproto.KeyPressEvent:
			// Check for 'q' or 'Esc' to quit.
			// This is a very raw check of keycodes usually map to these on standard keyboards.
			// For a robust app we'd need XKB, but for now we just want to verify window works.
			// 9 = ESC, 24 = q (on many layouts, but let's just print the code first or check detail)
			fmt.Printf("Key pressed: %v\n", e.Detail)
			// Simply quit on any key for the first test to ensure interaction works
			// checking for ESC (usually 9) or q (usually 24)
			if e.Detail == 9 || e.Detail == 24 {
				os.Exit(0)
			}
		case xproto.MapNotifyEvent:
			fmt.Println("Window mapped (visible)!")
		}
	}
}
