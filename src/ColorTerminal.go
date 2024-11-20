package main

import "fmt"

func printWithColor(color string, text string) {
	fmt.Println(color, text, "\033[0m")
}
