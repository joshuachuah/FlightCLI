/*
Copyright © 2026 Joshua Chuah jchuah07@gmail.com

*/
package main

import (
	"github.com/joho/godotenv"
	"github.com/joshuachuah/flightcli/cmd"
)

func main() {
	godotenv.Load()
	cmd.Execute()
}
