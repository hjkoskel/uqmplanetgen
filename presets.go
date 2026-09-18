package uqmplanetgen

import (
	"embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed presetdata/*.ct presetdata/*.xlt
var presetData embed.FS

// presetVariant is the colormap variant the presets use. Shipped .xlt files
// hold a single item, and variant 1 is the temperate .ct palette.
const presetVariant = 1

// presetSpec describes one UQM world type. Columns:
//
//	name, algo, .ct file, .xlt file, faults/bands, fault depth, craters, base elevation, gas giant
type presetSpec struct {
	name          string
	algo          Algo
	ctab, xtab    string
	numFaults     int
	faultDepth    int
	numBlemishes  int
	baseElevation int
	gasGiant      bool
}

var presetSpecs = []presetSpec{
	{"OOLITE_WORLD", AlgoCratered, "oolite.ct", "oolite.xlt", 230, 2, 200, 150, false},
	{"YTTRIC_WORLD", AlgoCratered, "vinyl.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"QUASI_DEGENERATE_WORLD", AlgoTopo, "primordial.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"LANTHANIDE_WORLD", AlgoCratered, "yellow.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"TREASURE_WORLD", AlgoCratered, "treasure.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"UREA_WORLD", AlgoCratered, "yellow.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"METAL_WORLD", AlgoCratered, "metal.ct", "metal.xlt", 230, 2, 200, 150, false},
	{"RADIOACTIVE_WORLD", AlgoCratered, "orange.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"OPALESCENT_WORLD", AlgoCratered, "opalescent.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"CYANIC_WORLD", AlgoCratered, "copper.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"ACID_WORLD", AlgoCratered, "rusted.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"ALKALI_WORLD", AlgoCratered, "banded.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"HALIDE_WORLD", AlgoCratered, "redux.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"GREEN_WORLD", AlgoCratered, "green.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"COPPER_WORLD", AlgoTopo, "copper.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"CARBIDE_WORLD", AlgoCratered, "orange.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"ULTRAMARINE_WORLD", AlgoCratered, "ultramarine.ct", "dented.xlt", 200, 2, 100, 100, false},
	{"NOBLE_WORLD", AlgoCratered, "noble.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"AZURE_WORLD", AlgoCratered, "azure.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"CHONDRITE_WORLD", AlgoCratered, "chrondrite.ct", "chondrite.xlt", 500, 1, 100, 190, false},
	{"PURPLE_WORLD", AlgoCratered, "purple.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"SUPER_DENSE_WORLD", AlgoTopo, "superdense.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"PELLUCID_WORLD", AlgoCratered, "pellucid.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"DUST_WORLD", AlgoCratered, "dust.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"CRIMSON_WORLD", AlgoCratered, "dust.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"CIMMERIAN_WORLD", AlgoTopo, "cimmerian.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"INFRARED_WORLD", AlgoCratered, "dust.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"SELENIC_WORLD", AlgoCratered, "selenic.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"AURIC_WORLD", AlgoTopo, "yellow.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"FLUORESCENT_WORLD", AlgoCratered, "purple.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"ULTRAVIOLET_WORLD", AlgoCratered, "purple.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"PLUTONIC_WORLD", AlgoCratered, "plutonic.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"RAINBOW_WORLD", AlgoTopo, "rainbow.ct", "rainbow.xlt", 500, 1, 20, 100, false},
	{"SHATTERED_WORLD", AlgoCratered, "shattered.ct", "shattered.xlt", 500, 1, 0, 185, false},
	{"SAPPHIRE_WORLD", AlgoTopo, "sapphire.ct", "crystal.xlt", 80, 1, 0, 128, false},
	{"ORGANIC_WORLD", AlgoTopo, "banded.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"XENOLITHIC_WORLD", AlgoCratered, "xenolithic.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"REDUX_WORLD", AlgoCratered, "redux.ct", "redux.xlt", 500, 1, 0, 190, false},
	{"PRIMORDIAL_WORLD", AlgoCratered, "primordial.ct", "rusted.xlt", 250, 2, 10, 200, false},
	{"EMERALD_WORLD", AlgoTopo, "emerald.ct", "crystal.xlt", 80, 1, 0, 128, false},
	{"CHLORINE_WORLD", AlgoCratered, "rusted.ct", "continents.xlt", 500, 1, 0, 190, false},
	{"MAGNETIC_WORLD", AlgoCratered, "redux.ct", "ice.xlt", 400, 1, 100, 190, false},
	{"WATER_WORLD", AlgoCratered, "water.ct", "continents.xlt", 500, 1, 0, 190, false},
	{"TELLURIC_WORLD", AlgoCratered, "water.ct", "rusted.xlt", 250, 2, 80, 200, false},
	{"HYDROCARBON_WORLD", AlgoTopo, "azure.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"IODINE_WORLD", AlgoCratered, "primordial.ct", "dented.xlt", 230, 2, 200, 150, false},
	{"VINYLOGOUS_WORLD", AlgoCratered, "vinyl.ct", "dented.xlt", 400, 1, 100, 190, false},
	{"RUBY_WORLD", AlgoTopo, "ruby.ct", "crystal.xlt", 80, 1, 0, 128, false},
	{"MAGMA_WORLD", AlgoTopo, "orange.ct", "banded.xlt", 500, 1, 0, 160, false},
	{"MAROON_WORLD", AlgoCratered, "maroon.ct", "dented.xlt", 230, 2, 200, 150, false},

	{"BLUE_GAS_GIANT", AlgoGasGiant, "bluegasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"CYAN_GAS_GIANT", AlgoGasGiant, "cyangasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"GREEN_GAS_GIANT", AlgoGasGiant, "greengasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"GRAY_GAS_GIANT", AlgoGasGiant, "greengasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"ORANGE_GAS_GIANT", AlgoGasGiant, "orangegasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"PURPLE_GAS_GIANT", AlgoGasGiant, "purplegasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"RED_GAS_GIANT", AlgoGasGiant, "redgasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"VIOLET_GAS_GIANT", AlgoGasGiant, "violetgasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
	{"YELLOW_GAS_GIANT", AlgoGasGiant, "yellowgasgiant.ct", "gasgiant.xlt", 10, 2, 8, 29, true},
}

// presets is built once at init. Worlds that share a .ct or .xlt file share the
// parsed table: they are read-only.
var presets = buildPresets()

func buildPresets() []Planet {
	xtabs := make(map[string]*XTab)
	ctabs := make(map[string]CTab)

	out := make([]Planet, len(presetSpecs))
	for i, s := range presetSpecs {
		xtab, ok := xtabs[s.xtab]
		if !ok {
			x, err := LoadXTab(mustReadPresetData(s.xtab), presetVariant)
			if err != nil {
				panic(fmt.Sprintf("uqmplanetgen: preset %s: %v", s.name, err))
			}
			xtab = &x
			xtabs[s.xtab] = xtab
		}

		ctab, ok := ctabs[s.ctab]
		if !ok {
			var err error
			if ctab, err = LoadCTab(mustReadPresetData(s.ctab), presetVariant); err != nil {
				panic(fmt.Sprintf("uqmplanetgen: preset %s: %v", s.name, err))
			}
			ctabs[s.ctab] = ctab
		}

		out[i] = Planet{
			Name:          s.name,
			Algo:          s.algo,
			Seed:          1,
			CTab:          ctab,
			XTab:          xtab,
			NumFaults:     s.numFaults,
			FaultDepth:    s.faultDepth,
			NumBlemishes:  s.numBlemishes,
			BaseElevation: s.baseElevation,
			GasGiant:      s.gasGiant,
		}
	}
	return out
}

// mustReadPresetData panics on failure: the files are embedded, so a miss is a
// build error rather than something a caller can handle.
func mustReadPresetData(name string) []byte {
	data, err := presetData.ReadFile("presetdata/" + name)
	if err != nil {
		panic("uqmplanetgen: embedded preset data: " + err.Error())
	}
	return data
}

// Presets returns a copy of every preset world, ready to render. The colormaps
// they point at are shared and must not be modified.
func Presets() []Planet { return slices.Clone(presets) }

// PresetNames returns the name of every preset, in table order.
func PresetNames() []string {
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.Name
	}
	return names
}

// PresetPlanet returns a copy of the named preset. The lookup is
// case-insensitive.
func PresetPlanet(name string) (Planet, error) {
	for _, p := range presets {
		if strings.EqualFold(p.Name, name) {
			return p, nil
		}
	}
	return Planet{}, fmt.Errorf("uqmplanetgen: no preset named %q", name)
}
