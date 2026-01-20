package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

// RotateX rotates the vector around the X axis
func (v Vec3) RotateX(angle float64) Vec3 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vec3{
		X: v.X,
		Y: v.Y*cos - v.Z*sin,
		Z: v.Y*sin + v.Z*cos,
	}
}

// RotateY rotates the vector around the Y axis
func (v Vec3) RotateY(angle float64) Vec3 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vec3{
		X: v.X*cos + v.Z*sin,
		Y: v.Y,
		Z: -v.X*sin + v.Z*cos,
	}
}

// RotateZ rotates the vector around the Z axis
func (v Vec3) RotateZ(angle float64) Vec3 {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	return Vec3{
		X: v.X*cos - v.Y*sin,
		Y: v.X*sin + v.Y*cos,
		Z: v.Z,
	}
}

// Project projects the 3D point to 2D screen coordinates
// fov: field of view factor (e.g., 200-400)
// viewerDistance: distance from camera to object center
func (v Vec3) Project(width, height int, fov, viewerDistance float64) (x, y int) {
	factor := fov / (viewerDistance + v.Z)
	x = int(v.X*factor) + width/2
	y = int(v.Y*factor) + height/2
	return x, y
}
