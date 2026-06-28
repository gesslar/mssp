package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"
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
			fmt.Println("Host is required.")
		} else if isZero(*port) && !isZero(*host) {
			fmt.Println("Port is required.")
		} else {
			fmt.Println("Host and port are required.")
		}
		os.Exit(1)
	}

	var config ConnectionConfig
	config.host = *host
	config.port = *port
	config.timeout = *timeout

	result, err := Connect(&config)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if !isZero(*value) {
		if vals, ok := result[*value]; ok {
			// if it's a single value just print the value, otherwise print the array
			if len(vals) == 1 {
				marshaled, err := json.Marshal(vals[0])
				if err != nil {
					fmt.Printf("Error marshaling value: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("%s\n", marshaled)
			} else {
				marshaled, err := json.Marshal(vals)
				if err != nil {
					fmt.Printf("Error marshaling value: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("%s\n", marshaled)
			}
		} else {
			os.Exit(0)
		}
	} else if len(result) == 0 {
		os.Exit(0)
	} else {
		marshaled, err := json.Marshal(result)
		if err != nil {
			fmt.Printf("Error marshaling result: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%s\n", marshaled)
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
