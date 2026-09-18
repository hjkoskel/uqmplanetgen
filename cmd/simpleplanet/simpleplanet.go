// Command simpleplanet renders UQM preset planet surfaces to PNG files.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"uqmplanetgen"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("simpleplanet: ")
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	var (
		outDir     = flag.String("od", "outdir", "output directory")
		allPresets = flag.Bool("d", false, "export every preset with its default parameters")
		presetName = flag.String("ps", "YTTRIC_WORLD", "preset to render, one of ["+
			strings.Join(uqmplanetgen.PresetNames(), ",")+"]")

		name          = flag.String("n", "planet", "output planet name")
		algo          = flag.Int("a", -1, "algorithm: topo=0, cratered=1, gas giant=2; -1 keeps the preset value")
		seed          = flag.Int("s", 1, "generation seed")
		ctabPath      = flag.String("ctab", "", "use this .ct color table instead of the preset's")
		xtabPath      = flag.String("xtab", "", "use this .xlt translation table instead of the preset's")
		numFaults     = flag.Int("nf", -1, "fault passes, or band count for gas giants; -1 keeps the preset value")
		faultDepth    = flag.Int("df", -1, "fault depth; -1 keeps the preset value")
		numBlemishes  = flag.Int("c", -1, "craters, used only by the cratered algorithm; -1 keeps the preset value")
		baseElevation = flag.Int("b", -1, "base elevation; -1 keeps the preset value")
		gasGiant      = flag.Int("gg", -1, "1 gas giant, 0 rocky; -1 keeps the preset value")
	)
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}

	if *allPresets {
		planets := uqmplanetgen.Presets()
		for i := range planets {
			planets[i].Seed = uint32(*seed)
			if err := renderToFile(&planets[i], *outDir); err != nil {
				return err
			}
		}
		fmt.Printf("wrote %d PNGs to %s\n", len(planets), *outDir)
		return nil
	}

	planet, err := uqmplanetgen.PresetPlanet(*presetName)
	if err != nil {
		return err
	}
	planet.Name = *name
	planet.Seed = uint32(*seed)
	if *algo >= 0 {
		planet.Algo = uqmplanetgen.Algo(*algo)
	}
	if *numFaults >= 0 {
		planet.NumFaults = *numFaults
	}
	if *faultDepth >= 0 {
		planet.FaultDepth = *faultDepth
	}
	if *numBlemishes >= 0 {
		planet.NumBlemishes = *numBlemishes
	}
	if *baseElevation >= 0 {
		planet.BaseElevation = *baseElevation
	}
	if *gasGiant >= 0 {
		planet.GasGiant = *gasGiant == 1
	}
	if *ctabPath != "" {
		data, err := os.ReadFile(*ctabPath)
		if err != nil {
			return err
		}
		if planet.CTab, err = uqmplanetgen.LoadCTab(data, 1); err != nil {
			return err
		}
	}
	if *xtabPath != "" {
		data, err := os.ReadFile(*xtabPath)
		if err != nil {
			return err
		}
		x, err := uqmplanetgen.LoadXTab(data, 1)
		if err != nil {
			return err
		}
		planet.XTab = &x
	}

	return renderToFile(&planet, *outDir)
}

func renderToFile(planet *uqmplanetgen.Planet, outDir string) error {
	img, err := planet.Render(uqmplanetgen.DefaultWidth, uqmplanetgen.DefaultHeight)
	if err != nil {
		return err
	}

	path := filepath.Join(outDir, fmt.Sprintf("%s_%d.png", planet.Name, planet.Seed))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	err = png.Encode(f, img)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	fmt.Printf("generated: %s\n", path)
	return nil
}
