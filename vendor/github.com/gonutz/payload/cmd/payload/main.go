// This program appends a data file to an executable file to create another
// executable file. In the resulting program you can use the
// github.com/gonutz/payload package to load the original data from the end of
// the file by using the Read function:
//     byteSlice, err := payload.Read()
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var (
	exeFile    = flag.String("exe", "", "Executable file to append data to.")
	dataFile   = flag.String("data", "", "Data file to be appended to the executable.")
	outputFile = flag.String("output", "", "Combined output file, defaults to the given exe.")
)

func main() {
	flag.Parse()

	if *exeFile == "" || *dataFile == "" {
		flag.Usage()
		return
	}

	if *outputFile == "" {
		*outputFile = *exeFile
	}

	stat, err := os.Stat(*exeFile)
	if err != nil {
		fmt.Println("error reading file attributes:", err)
		return
	}
	fileMode := stat.Mode()

	exe, err := ioutil.ReadFile(*exeFile)
	if err != nil {
		fmt.Println("error reading exe file:", err)
		return
	}

	data, err := ioutil.ReadFile(*dataFile)
	if err != nil {
		fmt.Println("error reading data file:", err)
		return
	}

	originalSize := bytes.NewBuffer(nil)
	size := uint64(len(exe))
	binary.Write(originalSize, binary.LittleEndian, size)

	output := append(exe, data...)
	output = append(output, []byte("payload ")...)
	output = append(output, originalSize.Bytes()...)

	err = ioutil.WriteFile(*outputFile, output, fileMode)
	if err != nil {
		fmt.Println("error writing output file:", err)
		return
	}
}
