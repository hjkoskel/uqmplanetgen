# wasmplanet

The planet generator as a WebAssembly module, with a single-page front end in
plain HTML, CSS and JavaScript.

## Running

```sh
./serve.sh          # builds main.wasm, then serves on :8000
```

Then open <http://localhost:8000>. The page has to be served over HTTP:
`WebAssembly.instantiateStreaming` needs the `application/wasm` MIME type, so
opening `index.html` from the filesystem will not work.

To build by hand:

```sh
GOOS=js GOARCH=wasm go build -o cmd/wasmplanet/main.wasm ./cmd/wasmplanet
```

`wasm_exec.js` has to come from the same Go toolchain as the build:
`$(go env GOROOT)/lib/wasm/wasm_exec.js` on Go 1.24 and later, or
`$(go env GOROOT)/misc/wasm/wasm_exec.js` before that.

TinyGo also builds this (`tinygo build -o main.wasm -target wasm .`), with its
own `targets/wasm_exec.js`.

## Files

- `wasmplanet.go` — the module; guarded by `//go:build js && wasm`, so a host
  build of the repository skips it
- `index.html` — the page
- `wasm_exec.js` — Go's JavaScript glue
- `serve.sh` — build and serve
- `main.wasm` — build output, not checked in

## Tests

`wasmtest.js` runs the module and `index.html`'s script under Node, with a
small DOM stub, so the whole path can be checked without a browser:

```sh
GOOS=js GOARCH=wasm go build -o main.wasm .
node wasmtest.js
```

## JavaScript API

The module installs these on the global object, then calls
`onPlanetGeneratorReady()` if the page defines it. Define that callback before
starting the module rather than polling for the functions.

| Function | Returns |
| --- | --- |
| `renderWithParams()` | base64 PNG, or a string starting with `error:` |
| `getPresetNames()` | comma-separated preset names |
| `getCurrentParamsJSON()` | current parameters as JSON |
| `setPresetPlanet(name)` | `null`, or a string starting with `error:` |
| `setPlanetName(name)` | `null` |
| `setPlanetSeed(n)` | `null` |
| `setPlanetAlgo(n)` | `null` — 0 topography, 1 cratered, 2 gas giant |
| `setPlanetNumFaults(n)` | `null` — fault passes, or band count for a gas giant |
| `setPlanetFaultDepth(n)` | `null` |
| `setPlanetNumBlemishes(n)` | `null` |
| `setPlanetBaseElevation(n)` | `null` |
| `setPlanetGasGiant(n)` | `null` — 1 for the gas giant colour path |

The setters ignore anything that is not a usable number, so an empty or
half-typed form field leaves the current value alone. `setPlanetAlgo` and
`setPlanetGasGiant` are independent: the algorithm decides how the elevation
map is built and the flag decides how it is coloured. The page sets both from
the algorithm dropdown.

A parameter set the generator cannot use — a gas giant with more bands than the
map has room for, say — comes back as an `error:` string. The module stays
usable afterwards.
