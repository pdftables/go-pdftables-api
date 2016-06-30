package main

import (
	"io"
	"log"
	"os"

	"github.com/pdftables/api/go/pkg/client"
)

func usage() {
	log.Printf("Remember to set PDFTABLES_API_KEY.")
	log.Fatal("usage: pdftables-api <filename>")
}

func main() {
	if len(os.Args) != 2 || os.Getenv("PDFTABLES_API_KEY") == "" {
		usage()
	}

	filename := os.Args[1]

	fd, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()

	converted, err := client.Do(fd, client.FormatCSV)
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(os.Stdout, converted)
	if err != nil {
		log.Fatal(err)
	}
}
