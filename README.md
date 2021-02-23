# cefingo-sample

Cefingo samples

Require go version 1.16 or later. 

## Usage

To go get github.com/turucrane/cefingo see https://github.com/turutcrane/cefingo/blob/master/README.md

Supported Environmant is Windows 64bit only.

## Sample 1

Open golang.org.

1. Change Directory.
    ```bat
    C:> cd simple
    ```
1. Run sample program.
    ```bat
    C:> go run . -url golang.org
    ```
## Sample 2

Wasm sample with http server

1. Cange Directory
    ```bat
    C:> cd wasm-http
    ```
1. Build wasm module.
    ```bat
    C:> cd wasm
    C:> build
    C:> cd ..
    ```
1. Run sample program.
    ```bat
    C:> go run .
    ```

## Sample 3

Wasm sample without http server

1. Cange Directory
    ```bat
    C:> cd wasm-nohttp
    ```
1. Build wasm module.
    ```bat
    C:> cd wasm
    C:> build
    C:> cd ..
    ```
1. Run sample program.
    ```bat
    C:> go run .
    ```

## Sample 4

A Sample without http server, wasm and JS. Only Go.

1. Cange Directory.
    ```bat
    C:> cd onlygo
    ```
1. Run sample program.
    ```bat
    C:> go run .
    ```

## Sample 5

Implements this [Monaco-editor](
https://github.com/microsoft/monaco-editor/blob/master/docs/integrate-amd.md) sample.

1. Cange Directory.
    ```bat
    C:> cd monaco-editor
    ```
1. Download monaco-editor tgz file from [direct download link](https://registry.npmjs.org/monaco-editor/-/monaco-editor-0.17.0.tgz)

1. Expand tgz file. 
    ```bat
    C:> tar xf monaco-editor-0.17.0.tgz
    ```

1. Run sample program.
    ```bat
    C:> go run .
    ```

