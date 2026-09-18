# uqmplanetgen

A Go port of the planet surface texture generator from *Star Control II* / The
Ur-Quan Masters. Output is bit-identical to the original C: given the same
seed and parameters, `uqmplanetgen` produces the same elevation map and the
same rendered pixels.

```go
planet, err := uqmplanetgen.PresetPlanet("WATER_WORLD")
if err != nil {
    log.Fatal(err)
}
planet.Seed = 12345

img, err := planet.Render(uqmplanetgen.DefaultWidth, uqmplanetgen.DefaultHeight)
if err != nil {
    log.Fatal(err)
}
png.Encode(out, img)
```

## How it works

An [`Elevation`](elevation.go) is a row-major map of signed 8-bit heights,
with 0 as sea level. `Elevation.Generate` fills one using one of three
algorithms ([`Algo`](doc.go)):

- **`AlgoTopo`** — fault-displacement terrain only
- **`AlgoCratered`** — faults, craters and dithering, for rocky worlds
- **`AlgoGasGiant`** — banded storms, for gas giants

`Render` turns an elevation map into an `image.RGBA` using a matching
colormap pair: an [`XTab`](xtab.go) maps each elevation level to a palette
index, and a [`CTab`](ctab.go) holds the color at each palette index.
[`Planet`](planet.go) bundles a generation recipe with its colormap pair, and
[`Presets`](presets.go) returns the ~59 world types UQM ships, loaded from
embedded `.ct`/`.xlt` files.

Generation draws on a caller-owned [`Rand`](rand.go) rather than a
package-level stream, so planets can be rendered concurrently.

See the [package doc](doc.go) for the full API, and
[`MIGRATION.md`](MIGRATION.md) if you're updating code written against an
earlier version of this package.

## Layout

```
uqmplanetgen/            the library
presetdata/               embedded .ct/.xlt files the preset table loads (bring your own — see below)
cmd/simpleplanet/         CLI: renders presets to PNG files
cmd/wasmplanet/           WebAssembly module + browser UI
```

### presetdata

`Presets()` loads its colormaps from `presetdata/*.ct` and `presetdata/*.xlt`
via `//go:embed`. That directory is not included here — copy your own
`.ct`/`.xlt` files into `presetdata/` at the repository root before building
anything that calls `Presets()` or `PresetPlanet()`; the build fails without
it. Code that only uses `Elevation`, `Render`, `LoadXTab`/`LoadCTab`, or
`NewCTab` directly doesn't need it.

## cmd/simpleplanet

Renders one preset, or all of them, to PNG files.

```sh
go run ./cmd/simpleplanet -ps WATER_WORLD -s 12345 -od out/
go run ./cmd/simpleplanet -d -od out/          # every preset, default params
```

Flags: `-ps` preset name, `-n` output name, `-s` seed, `-a` algorithm
(0 topo, 1 cratered, 2 gas giant), `-nf` fault passes or band count, `-df`
fault depth, `-c` crater count, `-b` base elevation, `-gg` gas-giant color
path, `-ctab`/`-xtab` to override the preset's colormap files, `-od` output
directory, `-d` export every preset. Run with `-h` for the full list.

## cmd/wasmplanet

The generator as a WebAssembly module with a single-page HTML/CSS/vanilla-JS
front end.

```sh
cd cmd/wasmplanet
./serve.sh                # builds main.wasm and serves on :8000
```

Then open <http://localhost:8000>. See
[`cmd/wasmplanet/README.md`](cmd/wasmplanet/README.md) for the JavaScript API
and how to run its test suite (`node wasmtest.js`) without a browser.

## Testing

```sh
go test ./...
```

Tests cover the RNG, elevation generation (including determinism and
concurrent-use safety), palette loading and rendering, and a golden-file suite
in `golden_test.go` that pins output against reference data captured from the
original C implementation. `presetdata/` is required for the preset-related
tests; the rest run without it.

## Compatibility

This package has no third-party dependencies. It aims to match the original
C's output exactly, including its integer wraparound behavior — see the
[package doc](doc.go) for what that implies about the arithmetic in
`elevation.go`, `crater.go` and `gasgiant.go`.
