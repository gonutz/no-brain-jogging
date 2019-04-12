package main

import (
	"flag"
	"fmt"
	"github.com/gonutz/blob"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	inPath  = flag.String("path", "", "File or folder to be blobbed")
	outPath = flag.String("out", "", "Output path")
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `blob takes a file or folder and creates a binary blob file of it.

If you blob a file, its ID will be the file name without the directory.

If you blob a folder, it will be traversed recursively and all regular files in
the tree will be blobbed. The IDs are the relative file names with respect to
the given root folder. The path separator is always slash.
Example: the following file structure 
  folder
  ---> index.html
  ---> static
       ---> favicon.ico
       ---> logo.png
results in the following IDs: "index.html", "static/favicon.ico",
"static/logo.png".

Usage of blob:
`)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *inPath == "" {
		errln("input path not specified")
		flag.Usage()
		return 1
	}
	if *outPath == "" {
		errln("output path not specified")
		flag.Usage()
		return 1
	}

	f, err := os.Lstat(*inPath)
	if err != nil {
		errln("cannot find input path '" + *inPath + "': " + err.Error())
		return 1
	}

	var b blob.Blob
	if f.IsDir() {
		err := filepath.Walk(*inPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				// TODO add this file
				data, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}
				relPath, _ := filepath.Rel(*inPath, path)
				id := filepath.ToSlash(relPath)
				b.Append(id, data)
			}
			return nil
		})
		if err != nil {
			errln("unable to traverse input directory: " + err.Error())
			return 1
		}
	} else {
		data, err := ioutil.ReadFile(*inPath)
		if err != nil {
			errln("unable to read input file: " + err.Error())
			return 1
		}
		b.Append(filepath.Base(*inPath), data)
	}

	outFile, err := os.Create(*outPath)
	if err != nil {
		errln("unable to create output file: " + err.Error())
		return 1
	}
	defer outFile.Close()

	if err := b.Write(outFile); err != nil {
		errln("unable to write output file: " + err.Error())
		return 1
	}

	return 0
}

func errln(msg string) {
	fmt.Fprintln(os.Stderr, "ERROR "+msg)
}
