package main

import (
	"fmt"
	"openify/Authentication"
	"openify/ConfigurationManager"
	"openify/FilesManager"
	"openify/Handlers"
	"runtime"
)



func main() {
	WhoAmI()
	config := ConfigurationManager.OpenConfiguration()
	_ = FilesManager.ScanFolder(config.DocumentRoot)
	authentication.LoadUsers()
	Handlers.HandleRequests()
}

func WhoAmI() {
	fmt.Println("Openify Server by Alexis Delhaie")
	fmt.Printf("\nVersion: %s\n", ConfigurationManager.GetVersion())
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Build for: %s %s\n---------\n", runtime.GOOS, runtime.GOARCH)
}
