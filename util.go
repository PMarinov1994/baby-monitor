package main

import (
	"log"
	"runtime/debug"
)

func checkError(err *error) {
	if *err != nil {
		debug.PrintStack()
		log.Fatalf("Error: %v\n", *err)
	}
}
