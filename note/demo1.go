package main

import (
	"fmt"

	pdf "github.com/adrg/go-wkhtmltopdf"
	"github.com/pschlump/filelib"
)

func GenPdf(in, out string) error {

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
	converter.SetOption("documentTitle", "Sample document") // xyzzy fix
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

	fmt.Fprintf(fp, "%s", output)
	return nil
}

func main() {
	pdf.Init()
	defer pdf.Destroy()
	GenPdf("https://en.wikipedia.org/wiki/Secure_Remote_Password_protocol", ",a.pdf")
	GenPdf("https://www.google.com", ",b.pdf")
}
