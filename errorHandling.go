package main

import (
	"errors"
	"fmt"
)

func createError(functionName string, errorDetails string) error {
	return errors.New(fmt.Sprint("Error on ", functionName, ":", errorDetails))
}
