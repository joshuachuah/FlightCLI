/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"github.com/joho/godotenv"
	"github.com/xjosh/flightcli/cmd"
)

func main() {
	godotenv.Load()
	cmd.Execute()
}
