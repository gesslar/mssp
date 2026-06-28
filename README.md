# mssp

A small command-line client for the [MSSP](https://tintin.mudhalla.net/protocols/mssp/)
(Mud Server Status Protocol). It connects to a MUD server over Telnet, requests
its MSSP data, and prints the result as JSON — or, with `-value`, the value of a
single variable.

MSSP lets MUD servers advertise metadata about themselves (name, player count,
uptime, codebase, supported features, and so on) so crawlers and listing sites
can poll them. This tool is a one-shot poller for that data.

## Requirements

- [Go](https://go.dev/) 1.26 or newer (per `go.mod`)

## Build

```sh
make build
```

This builds a host-native binary into `build/`, named for the target platform,
e.g. `build/mssp-linux-amd64` (or `…-windows-amd64.exe` on Windows). The build
respects `GOOS`/`GOARCH` overrides, so you can cross-compile:

```sh
GOOS=windows GOARCH=amd64 make build   # -> build/mssp-windows-amd64.exe
```

Or build/run directly with the Go toolchain without the Makefile:

```sh
go build -o mssp .
go run . -host <host> -port <port>
```

## Usage

```sh
mssp -host <host> -port <port> [-value <variable>] [-timeout <seconds>]
```

### Flags

| Flag       | Type   | Default | Description                                              |
|------------|--------|---------|----------------------------------------------------------|
| `-host`    | string | —       | Host to connect to. **Required.**                        |
| `-port`    | int    | —       | Port to connect to. **Required.**                        |
| `-value`   | string | —       | Print only this MSSP variable's value instead of all.    |
| `-timeout` | int    | `5`     | Connection timeout in seconds.                           |

`-host` and `-port` are both required; if either is missing the tool prints a
message and exits with status `1`.

### Output

- **Without `-value`:** the full MSSP result is printed as a single JSON object.
  A variable with one value renders as a string; a variable with multiple values
  renders as an array.

  ```json
  {"NAME":"Oxidus","PLAYERS":"42","PORT":["23","80"],"UPTIME":"1700000000"}
  ```

- **With `-value <variable>`:** only that variable's value is printed. A single
  value is printed bare; multiple values are printed as a bracketed,
  comma-separated list.

  ```sh
  mssp -host example.org -port 23 -value NAME
  # Oxidus

  mssp -host example.org -port 23 -value PORT
  # [23, 80]
  ```

If the requested data is absent (no MSSP data returned, or the named `-value`
variable is not present), the tool prints nothing and exits `0`.

### Examples

```sh
# Dump all advertised MSSP variables as JSON
mssp -host aardwolf.org -port 23

# Read just the player count
mssp -host aardwolf.org -port 23 -value PLAYERS

# Use a longer timeout for a slow server
mssp -host slowmud.example -port 4000 -timeout 15
```

## Exit codes

| Code | Meaning                                                          |
|------|-----------------------------------------------------------------|
| `0`  | Success — data printed, or the requested data was absent.       |
| `1`  | Usage error (missing `-host`/`-port`) or a connection/runtime error. |

## How it works

1. Dials the server over TCP with the configured timeout.
2. Sends the MSSP request handshake (`IAC DO MSSP`).
3. Reads until it receives a complete MSSP subnegotiation, framed by
   `IAC SB MSSP … IAC SE`.
4. Parses the payload into variable/value pairs, following the MSSP rules:
   - One variable followed by several values is a **list** (e.g.
     `PORT -> [23, 80]`).
   - The same variable reported again is an **override** — last one wins.

### Note

`mssp` assumes that all values are strings per the [MSSP spec](https://tintin.mudhalla.net/protocols/mssp/). Therefore, any singular or array value will be printed without quotes. The JSON value is still quoted, per JSON spec.

## Testing

```sh
make test
```

Runs `go test ./...`.

## Cleaning

```sh
make clean
```

Removes the `build/` directory.

## License

`mssp` is released under the [0BSD](LICENSE.txt)