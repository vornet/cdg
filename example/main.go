package main

import (
	"fmt"
	"github.com/vornet/cdg"
)

func main() {
	importer := cdg.NewImporter("O:")
	importer.ImportDisc()

	fmt.Println("Hello World!")
}
