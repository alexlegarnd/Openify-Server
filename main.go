package main

import (
	"Openify/handlers"
	"Openify/managers"
	"fmt"
	"runtime"
)

var version = "1.0.0"

func main() {
	WhoAmI()
	config:= managers.OpenConfiguration()
	_ = managers.ScanFolder(config.DocumentRoot)
	handlers.HandleRequests()
}

func WhoAmI() {
	fmt.Println("Openify Server by Alexis Delhaie")
	fmt.Println(fmt.Sprintf("\nVersion: %s", version))
	fmt.Println(fmt.Sprintf("Go Version: %s\n---------", runtime.Version()))
}
