package main

import (
	"fmt"

	pdf "github.com/adrg/go-wkhtmltopdf"
	"github.com/pschlump/filelib"
)

func GenPDF(title, in, out string) error {

	// Create object from url
	object2, err := pdf.NewObject(in)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}
	object2.SetOption("footer.right", "[page]")

	// Create converter
	converter := pdf.NewConverter()
	defer converter.Destroy()

	// Add created objects to the converter
	converter.AddObject(object2)

	// Add converter options
	converter.SetOption("documentTitle", title)
	converter.SetOption("margin.left", "10mm")
	converter.SetOption("margin.right", "10mm")
	converter.SetOption("margin.top", "10mm")
	converter.SetOption("margin.bottom", "10mm")

	// Convert the objects and get the output PDF document
	output, err := converter.Convert()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}

	fp, err := filelib.Fopen(out, "w")
	if err != nil {
		return err
	}
	defer fp.Close()

	IncPdf()
	fmt.Fprintf(fp, "%s", output)
	return nil
}
