// Command mssp is a one-shot command-line client for the MSSP (Mud Server
// Status Protocol). It connects to a server, requests its MSSP data, and prints
// the result as JSON — or, with -value, the value of a single variable.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/gesslar/mssp"
)

// main parses command-line flags (-host, -port, -value, -timeout), connects to
// the server, and prints the MSSP data as JSON. With -value it prints only that
// variable's value; otherwise it prints the full result map. It exits with
// status 1 on a usage or runtime error, and status 0 (printing nothing) when
// the requested data is absent.
func main() {
	host := flag.String("host", zeroFor[string](), "The host to connect to")
	port := flag.Int("port", zeroFor[int](), "The port to connect to")
	value := flag.String("value", zeroFor[string](), "The value to send (optional)")
	timeout := flag.Int("timeout", 5, "Connection timeout in seconds")

	flag.Parse()

	if isZero(*host) || isZero(*port) {
		if isZero(*host) && !isZero(*port) {
			fmt.Fprintf(os.Stderr, "Host is required.\n\n")
		} else if isZero(*port) && !isZero(*host) {
			fmt.Fprintf(os.Stderr, "Port is required.\n\n")
		} else {
			fmt.Fprintf(os.Stderr, "Host and port are required.\n\n")
		}
		flag.Usage()
		os.Exit(1)
	}

	config := mssp.NewConnectionConfig(*host, *port, *timeout)

	result, err := mssp.Connect(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !isZero(*value) {
		if vals, ok := result[*value]; ok {
			// if it's a single value just print the value, otherwise print the array
			if len(vals) == 1 {
				fmt.Println(string(vals[0]))
			} else if len(vals) > 1 {
				// Get the marshaled value, but don't include the quotes around the array values.
				joined := "[" + strings.Join(vals, ", ") + "]"
				fmt.Println(joined)
			} else {
				os.Exit(0)
			}
		} else {
			os.Exit(0)
		}
	} else if len(result) == 0 {
		os.Exit(0)
	} else {
		marshaled, err := json.Marshal(result)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(marshaled))
	}
}

// zeroFor returns the zero value for type T. It is handy as a default when
// declaring flags whose "unset" state should be the type's zero value.
func zeroFor[T any]() T {
	var zero T
	return zero
}

// isZero reports whether value equals the zero value for its type T.
func isZero[T any](value T) bool {
	var zero T
	return reflect.DeepEqual(value, zero)
}
