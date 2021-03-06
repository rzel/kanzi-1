/*
Copyright 2011-2017 Frederic Langlet
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
you may obtain a copy of the License at

                http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io"
	"kanzi"
	kio "kanzi/io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DECOMP_DEFAULT_BUFFER_SIZE = 32768
)

// Main block decompressor struct
type BlockDecompressor struct {
	verbosity  uint
	overwrite  bool
	inputName  string
	outputName string
	jobs       uint
	listeners  []kanzi.Listener
	cpuProf    string
}

func NewBlockDecompressor(argsMap map[string]interface{}) (*BlockDecompressor, error) {
	this := new(BlockDecompressor)
	this.listeners = make([]kanzi.Listener, 0)

	this.verbosity = argsMap["verbose"].(uint)
	delete(argsMap, "verbose")

	if force, prst := argsMap["overwrite"]; prst == true {
		this.overwrite = force.(bool)
		delete(argsMap, "overwrite")
	} else {
		this.overwrite = false
	}

	this.inputName = argsMap["inputName"].(string)
	delete(argsMap, "inputName")
	this.outputName = argsMap["outputName"].(string)
	delete(argsMap, "outputName")
	this.jobs = argsMap["jobs"].(uint)
	delete(argsMap, "jobs")

	if prof, prst := argsMap["cpuProf"]; prst == true {
		this.cpuProf = prof.(string)
		delete(argsMap, "cpuProf")
	} else {
		this.cpuProf = ""
	}

	if this.verbosity > 2 {
		if listener, err := NewInfoPrinter(this.verbosity, DECODING, os.Stdout); err == nil {
			this.AddListener(listener)
		}
	}

	if this.verbosity > 0 && len(argsMap) > 0 {
		for k, _ := range argsMap {
			bd_printOut("Ignoring invalid option ["+k+"]", this.verbosity > 0)
		}
	}

	return this, nil
}

func (this *BlockDecompressor) CpuProf() string {
	return this.cpuProf
}

func (this *BlockDecompressor) AddListener(bl kanzi.Listener) bool {
	if bl == nil {
		return false
	}

	this.listeners = append(this.listeners, bl)
	return true
}

func (this *BlockDecompressor) RemoveListener(bl kanzi.Listener) bool {
	for i, e := range this.listeners {
		if e == bl {
			this.listeners = append(this.listeners[:i-1], this.listeners[i+1:]...)
			return true
		}
	}

	return false
}

// Return exit code, number of bits written
func (this *BlockDecompressor) Call() (int, uint64) {
	var msg string
	printFlag := this.verbosity > 2
	bd_printOut("Kanzi 1.2 (C) 2017,  Frederic Langlet", this.verbosity >= 1)
	bd_printOut("Input file name set to '"+this.inputName+"'", printFlag)
	bd_printOut("Output file name set to '"+this.outputName+"'", printFlag)
	msg = fmt.Sprintf("Verbosity set to %v", this.verbosity)
	bd_printOut(msg, printFlag)
	msg = fmt.Sprintf("Overwrite set to %t", this.overwrite)
	bd_printOut(msg, printFlag)

	if this.jobs > 0 {
		prefix := ""

		if this.jobs > 1 {
			prefix = "s"
		}

		msg = fmt.Sprintf("Using %d job%s", this.jobs, prefix)
		bd_printOut(msg, printFlag)
	}

	var output io.WriteCloser

	if strings.ToUpper(this.outputName) == "NONE" {
		output, _ = kio.NewNullOutputStream()
	} else if strings.ToUpper(this.outputName) == "STDOUT" {
		output = os.Stdout
	} else {
		var err error

		if output, err = os.OpenFile(this.outputName, os.O_RDWR, 0666); err == nil {
			// File exists
			if this.overwrite == false {
				fmt.Printf("The output file '%v' exists and the 'overwrite' command ", this.outputName)
				fmt.Println("line option has not been provided")
				output.Close()
				return kio.ERR_OVERWRITE_FILE, 0
			}

			path1, _ := filepath.Abs(this.inputName)
			path2, _ := filepath.Abs(this.outputName)

			if path1 == path2 {
				fmt.Print("The input and output files must be different")
				return kio.ERR_CREATE_FILE, 0
			}
		} else {
			// File does not exist, create
			if output, err = os.Create(this.outputName); err != nil {
				fmt.Printf("Cannot open output file '%v' for writing: %v\n", this.outputName, err)
				return kio.ERR_CREATE_FILE, 0
			}
		}
	}

	defer func() {
		output.Close()
	}()

	// Decode
	read := int64(0)
	printFlag = this.verbosity > 1
	bd_printOut("Decoding ...", printFlag)
	var input io.ReadCloser

	if len(this.listeners) > 0 {
		evt := kanzi.NewEvent(kanzi.EVT_DECOMPRESSION_START, -1, 0, 0, false)
		bd_notifyListeners(this.listeners, evt)
	}

	if strings.ToUpper(this.inputName) == "STDIN" {
		input = os.Stdin
	} else {
		var err error

		if input, err = os.Open(this.inputName); err != nil {
			fmt.Printf("Cannot open input file '%v': %v\n", this.inputName, err)
			return kio.ERR_OPEN_FILE, uint64(read)
		}

		defer func() {
			input.Close()
		}()
	}

	ctx := make(map[string]interface{})
	ctx["jobs"] = this.jobs
	cis, err := kio.NewCompressedInputStream(input, ctx)

	if err != nil {
		if err.(*kio.IOError) != nil {
			fmt.Printf("%s\n", err.(*kio.IOError).Message())
			return err.(*kio.IOError).ErrorCode(), uint64(read)
		}

		fmt.Printf("Cannot create compressed stream: %v\n", err)
		return kio.ERR_CREATE_DECOMPRESSOR, uint64(read)
	}

	for _, bl := range this.listeners {
		cis.AddListener(bl)
	}

	buffer := make([]byte, DECOMP_DEFAULT_BUFFER_SIZE)
	decoded := len(buffer)
	before := time.Now()

	// Decode next block
	for decoded == len(buffer) {
		if decoded, err = cis.Read(buffer); err != nil {
			if ioerr, isIOErr := err.(*kio.IOError); isIOErr == true {
				fmt.Printf("%s\n", ioerr.Message())
				return ioerr.ErrorCode(), uint64(read)
			}

			fmt.Printf("An unexpected condition happened. Exiting ...\n%v\n", err)
			return kio.ERR_PROCESS_BLOCK, uint64(read)
		}

		if decoded > 0 {
			_, err = output.Write(buffer[0:decoded])

			if err != nil {
				fmt.Printf("Failed to write decompressed block to file '%v': %v\n", this.outputName, err)
				return kio.ERR_WRITE_FILE, uint64(read)
			}

			read += int64(decoded)
		}
	}

	// Close streams to ensure all data are flushed
	// Deferred close is fallback for error paths
	if err := cis.Close(); err != nil {
		fmt.Printf("%v\n", err)
		return kio.ERR_PROCESS_BLOCK, uint64(read)
	}

	after := time.Now()
	delta := after.Sub(before).Nanoseconds() / 1000000 // convert to ms

	bd_printOut("", this.verbosity >= 1)
	msg = fmt.Sprintf("Decoding:          %d ms", delta)
	bd_printOut(msg, printFlag)
	msg = fmt.Sprintf("Input size:        %d", cis.GetRead())
	bd_printOut(msg, printFlag)
	msg = fmt.Sprintf("Output size:       %d", read)
	bd_printOut(msg, printFlag)
	msg = fmt.Sprintf("Decoding: %v => %v bytes in %v ms", cis.GetRead(), read, delta)
	bd_printOut(msg, this.verbosity == 1)

	if delta > 0 {
		msg = fmt.Sprintf("Throughput (KB/s): %d", ((read*int64(1000))>>10)/int64(delta))
		bd_printOut(msg, printFlag)
	}

	bd_printOut("", this.verbosity >= 1)

	if len(this.listeners) > 0 {
		evt := kanzi.NewEvent(kanzi.EVT_DECOMPRESSION_END, -1, int64(cis.GetRead()), 0, false)
		bd_notifyListeners(this.listeners, evt)
	}

	return 0, cis.GetRead()
}

func bd_printOut(msg string, print bool) {
	if print == true {
		fmt.Println(msg)
	}
}

func bd_notifyListeners(listeners []kanzi.Listener, evt *kanzi.Event) {
	defer func() {
		if r := recover(); r != nil {
			// Ignore exceptions in listeners
		}
	}()

	for _, bl := range listeners {
		bl.ProcessEvent(evt)
	}
}
