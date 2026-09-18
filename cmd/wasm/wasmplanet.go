//go:build js && wasm

// Command wasmplanet exposes the planet generator to a web page as a
// WebAssembly module. Build it with:
//
//	GOOS=js GOARCH=wasm go build -o cmd/wasmplanet/main.wasm ./cmd/wasmplanet
//
// index.html drives the functions this installs on the JavaScript global
// object; see README.md for the list.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"math"
	"strings"
	"syscall/js"

	"uqmplanetgen"
)

// defaultPreset is loaded at startup so that a render before the page picks a
// preset still has a colormap to work with.
const defaultPreset = "WATER_WORLD"

// current holds the parameters the page is editing. JavaScript calls arrive on
// a single goroutine, so it needs no locking.
var current uqmplanetgen.Planet

func main() {
	preset, err := uqmplanetgen.PresetPlanet(defaultPreset)
	if err != nil {
		fmt.Println("wasmplanet:", err)
		return
	}
	current = preset

	for name, fn := range map[string]func(js.Value, []js.Value) any{
		"renderWithParams":       renderWithParams,
		"getPresetNames":         getPresetNames,
		"getCurrentParamsJSON":   getCurrentParamsJSON,
		"setPresetPlanet":        setPresetPlanet,
		"setPlanetName":          setPlanetName,
		"setPlanetSeed":          setPlanetSeed,
		"setPlanetAlgo":          setPlanetAlgo,
		"setPlanetNumFaults":     setPlanetNumFaults,
		"setPlanetFaultDepth":    setPlanetFaultDepth,
		"setPlanetNumBlemishes":  setPlanetNumBlemishes,
		"setPlanetBaseElevation": setPlanetBaseElevation,
		"setPlanetGasGiant":      setPlanetGasGiant,
	} {
		js.Global().Set(name, js.FuncOf(fn))
	}

	// Tell the page the functions are installed, so it does not have to poll.
	if ready := js.Global().Get("onPlanetGeneratorReady"); ready.Type() == js.TypeFunction {
		ready.Invoke()
	}

	select {} // keep the module alive for the callbacks
}

// argNum returns args[0] as a float64, reporting whether it is a usable
// number. JavaScript hands over NaN for an empty input field.
func argNum(args []js.Value) (float64, bool) {
	if len(args) == 0 || args[0].Type() != js.TypeNumber {
		return 0, false
	}
	v := args[0].Float()
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

// argInt is argNum clamped into a sane integer range.
func argInt(args []js.Value) (int, bool) {
	v, ok := argNum(args)
	if !ok || v < math.MinInt32 || v > math.MaxInt32 {
		return 0, false
	}
	return int(v), ok
}

func renderWithParams(js.Value, []js.Value) any {
	img, err := current.Render(uqmplanetgen.DefaultWidth, uqmplanetgen.DefaultHeight)
	if err != nil {
		return errorString(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return errorString(err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func getPresetNames(js.Value, []js.Value) any {
	return strings.Join(uqmplanetgen.PresetNames(), ",")
}

func getCurrentParamsJSON(js.Value, []js.Value) any {
	b, err := json.Marshal(struct {
		Name          string `json:"name"`
		Algo          int    `json:"algo"`
		Seed          uint32 `json:"seed"`
		NumFaults     int    `json:"numFaults"`
		FaultDepth    int    `json:"faultDepth"`
		NumBlemishes  int    `json:"numBlemishes"`
		BaseElevation int    `json:"baseElevation"`
		GasGiant      bool   `json:"gasGiant"`
	}{
		Name:          current.Name,
		Algo:          int(current.Algo),
		Seed:          current.Seed,
		NumFaults:     current.NumFaults,
		FaultDepth:    current.FaultDepth,
		NumBlemishes:  current.NumBlemishes,
		BaseElevation: current.BaseElevation,
		GasGiant:      current.GasGiant,
	})
	if err != nil {
		return errorString(err)
	}
	return string(b)
}

func setPresetPlanet(_ js.Value, args []js.Value) any {
	if len(args) == 0 || args[0].Type() != js.TypeString {
		return errorString(fmt.Errorf("setPresetPlanet wants a preset name"))
	}
	preset, err := uqmplanetgen.PresetPlanet(args[0].String())
	if err != nil {
		return errorString(err)
	}
	current = preset
	return nil
}

func setPlanetName(_ js.Value, args []js.Value) any {
	if len(args) > 0 && args[0].Type() == js.TypeString {
		current.Name = args[0].String()
	}
	return nil
}

func setPlanetSeed(_ js.Value, args []js.Value) any {
	// Seeds are 32-bit and wrap, so this takes the raw number rather than
	// going through argInt's narrower range.
	if v, ok := argNum(args); ok {
		current.Seed = uint32(int64(v))
	}
	return nil
}

func setPlanetAlgo(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok && v >= int(uqmplanetgen.AlgoTopo) && v <= int(uqmplanetgen.AlgoGasGiant) {
		current.Algo = uqmplanetgen.Algo(v)
	}
	return nil
}

func setPlanetNumFaults(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok && v >= 0 {
		current.NumFaults = v
	}
	return nil
}

func setPlanetFaultDepth(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok {
		current.FaultDepth = v
	}
	return nil
}

func setPlanetNumBlemishes(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok && v >= 0 {
		current.NumBlemishes = v
	}
	return nil
}

func setPlanetBaseElevation(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok {
		current.BaseElevation = v
	}
	return nil
}

func setPlanetGasGiant(_ js.Value, args []js.Value) any {
	if v, ok := argInt(args); ok {
		current.GasGiant = v == 1
	}
	return nil
}

// errorString is the failure form the page checks for.
func errorString(err error) string { return "error: " + err.Error() }
