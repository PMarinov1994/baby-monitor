package util

import (
	"log"
	"runtime/debug"
)

func CheckError(err *error) {
	if *err != nil {
		debug.PrintStack()
		log.Fatalf("Error: %v\n", *err)
	}
}
