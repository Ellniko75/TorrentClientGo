package main

import (
	"errors"
	"fmt"
)

type Error struct {
	FunctionName string
	ErrorName    string
}

func createError(functionName string, errorDetails string) error {
	return errors.New(fmt.Sprint("Error on ", functionName, ":", errorDetails))
}
