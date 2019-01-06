# cefingo-sample

Cefingo samples

## Usage

To go get github.com/turucrane/cefingo see https://github.com/turutcrane/cefingo/blob/master/README.md

Supported Environmant is Windows 64bit only.

## Sample 1

Open golang.org.

1. Change Directory
    ```bat
    C:> cd simple
    ```
1. Run sample program
    ```bat
    C:> go run main.go -url golang.org
    ```
## Sample 2

Wasm sample with http server

1. Cange Directory
    ```bat
    C:> cd wasm-http
    ```
1. Build wasm module
    ```bat
    C:> cd wasm
    C:> build
    C:> cd ..
    ```
1. Run sample program
    ```bat
    C:> go run main.go
    ```

## Sample 3

Wasm sample without http server

1. Cange Directory
    ```bat
    C:> cd wasm-nohttp
    ```
1. Build wasm module
    ```bat
    C:> cd wasm
    C:> build
    C:> cd ..
    ```
1. Run sample program
    ```bat
    C:> go run main.go
    ```

## Sample 4

A Sample without http server, wasm and JS. Only Go.

1. Cange Directory
    ```bat
    C:> cd onlygo
    ```
1. Run sample program
    ```bat
    C:> go run main.go
    ```