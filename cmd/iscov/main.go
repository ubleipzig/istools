package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/miku/holdings"
	"github.com/miku/holdings/google"
	"github.com/miku/holdings/kbart"
	"github.com/miku/holdings/ovid"
	"github.com/miku/istools"
	"github.com/miku/span/finc"
)

func main() {
	filename := flag.String("file", "", "path to holdings file")
	format := flag.String("format", "kbart", "holding file format, kbart, google, ovid")
	permissiveMode := flag.Bool("permissive", false, "if we cannot check, we allow")
	version := flag.Bool("version", false, "show version")

	flag.Parse()

	if *version {
		fmt.Println(istools.Version)
		os.Exit(0)
	}

	if *filename == "" {
		log.Fatal("holding -file required")
	}

	var r *bufio.Reader
	if flag.NArg() == 0 {
		r = bufio.NewReader(os.Stdin)
	} else {
		file, err := os.Open(flag.Arg(0))
		if err != nil {
			log.Fatal(err)
		}
		r = bufio.NewReader(file)
	}

	hfile, err := os.Open(*filename)
	if err != nil {
		log.Fatal(err)
	}

	var hr holdings.File

	switch *format {
	case "kbart":
		hr = kbart.NewReader(hfile)
	case "ovid":
		hr = ovid.NewReader(hfile)
	case "google":
		hr = google.NewReader(hfile)
	default:
		log.Fatalf("invalid holding file format: %s", *format)
	}

	entries, err := hr.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for {
		b, err := r.ReadBytes('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		var is finc.IntermediateSchema
		if err := json.Unmarshal(b, &is); err != nil {
			log.Fatal(err)
		}
		signature := holdings.Signature{
			Date:   is.Date.Format("2006-01-02"),
			Volume: is.Volume,
			Issue:  is.Issue,
		}

		// validate record, if at least one license allows this item
		var valid bool

	LOOP:
		for _, issn := range append(is.ISSN, is.EISSN...) {
			licenses := entries.Licenses(issn)

			if len(licenses) == 0 && *permissiveMode {
				valid = true
				break LOOP
			}

			for _, license := range licenses {
				coverage := license.Covers(signature)
				wall := license.TimeRestricted(is.Date)
				if coverage == nil && wall == nil {
					valid = true
					break LOOP
				}
			}
		}

		if len(is.ISSN) == 0 && len(is.EISSN) == 0 && *permissiveMode {
			valid = true
		}

		fmt.Printf("%s\t%v\n", is.RecordID, valid)
	}
}
