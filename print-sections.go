package main

import "os"
import "fmt"
import "github.com/go-ini/ini"

var ini_file string

func main() {
	if len(os.Args) != 2 {
		fmt.Println("[error] No file")
		os.Exit(1)
	}
	ini_file = os.Args[1]
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, ini_file)
	if err != nil {
		fmt.Println("[error] Could not load", ini_file)
		os.Exit(1)
	}

	for _, section := range cfg.Sections() {
		fmt.Println("Section:", section.Name())
	}
}
