package uqmplanetgen

import (
	"fmt"
	"image"
)

// Planet is a complete surface recipe: the generation parameters plus the
// colormap pair used to shade the result. The tables are read-only and may be
// shared between planets.
type Planet struct {
	Name string
	Algo Algo
	Seed uint32

	CTab CTab
	XTab *XTab

	NumFaults     int // fault passes for rocky worlds, band count for gas giants
	FaultDepth    int
	NumBlemishes  int // craters; used only by AlgoCratered
	BaseElevation int // sea-level offset; ignored for gas giants
	GasGiant      bool
}

// Render generates a surface and shades it. Planets are independent, so
// several may be rendered concurrently.
func (p *Planet) Render(width, height int) (*image.RGBA, error) {
	elev := NewElevation(width, height)
	if err := elev.Generate(p.Algo, p.Seed, p.NumFaults, p.FaultDepth, p.NumBlemishes); err != nil {
		return nil, fmt.Errorf("%s: generate: %w", p.Name, err)
	}
	img, err := Render(elev, p.GasGiant, p.BaseElevation, p.XTab, p.CTab)
	if err != nil {
		return nil, fmt.Errorf("%s: render: %w", p.Name, err)
	}
	return img, nil
}
