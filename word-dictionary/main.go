package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Header struct {
	FileVersion int8
	IndexStart  int64
}

const HeaderSize int64 = 256

var indexMapper map[string]int64

func BuildBaseCsvIndex(filePath string) error {

	baseFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open the file %s : err [%w]", filePath, err)
	}
	tmpFile, err := os.CreateTemp("/workspaces/shared-lib/", "temp-data-*.csv")
	if err != nil {
		return fmt.Errorf("unable to create temp file : err [%w]", err)
	}

	_, err = tmpFile.Write(make([]byte, HeaderSize))
	if err != nil {
		return fmt.Errorf("unable to write in temp file : err [%w]", err)
	}

	cursor := bufio.NewReader(baseFile)

	// By Default This will Header Size Preallocation
	offset := HeaderSize
	// Infinite For Loop Until We Will Hit The EOF
	index := map[string]int64{}
	for {
		// Read Tile \n
		line, err := cursor.ReadBytes('\n')

		if len(line) > 0 {
			// Get The Bytes Index For the First Appearance of Given Char
			keyWordEndIndex := bytes.IndexByte(line, ',')
			// Get The Keyword Bytes Until the keyWordEndIndex Bytes And Convert this into String
			keyword := string(line[:keyWordEndIndex])
			index[keyword] = offset

			numberOfBytesWritten, err := tmpFile.Write(line)
			if err != nil {
				return fmt.Errorf("unable to write in temp file : err [%w]", err)
			}
			offset += int64(numberOfBytesWritten)
		}

		//Because Last Line Can Be  Skip If we Add the EOF at the Top
		if err == io.EOF {
			break
		}
	}
	//now we need to write the header files
	for word, offset := range index {
		indexData := fmt.Sprintf("%s,%d\n", word, offset)
		_, err := tmpFile.Write([]byte(indexData))
		if err != nil {
			return fmt.Errorf("unable to write in temp file : err [%w]", err)
		}
	}

	//now write the header to the file

	//move the cursor for the tmpfile to 0
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("unable to seek in temp file : err [%w]", err)
	}

	headerData := Header{
		FileVersion: 1,
		IndexStart:  offset,
	}
	headerString := fmt.Sprintf("%v,%d\n", headerData.FileVersion, headerData.IndexStart)
	_, err = tmpFile.Write([]byte(headerString))
	if err != nil {
		return fmt.Errorf("unable to write in temp file : err [%w]", err)
	}
	// Sync forces all buffered writes to be flushed from the OS page cache
	// to stable storage (disk) before we rename the temp file.
	//
	// Although tmpFile.Write() succeeds, the data may still reside only in
	// the operating system's memory cache. If the process, OS, or machine
	// crashes before those cached pages are persisted, the generated
	// dictionary file could be incomplete or corrupted.
	//
	// Calling Sync() gives us a durability checkpoint: once it returns
	// successfully, the data and index sections have been committed to disk.
	// This follows the classic storage-engine pattern:
	//
	//     Write Data
	//     Write Index
	//     Sync
	//     Close
	//     Rename
	//
	// ensuring we never replace the original file with a partially written one.
	err = tmpFile.Sync()
	if err != nil {
		return err
	}

	baseFile.Close()
	tmpFile.Close()

	err = os.Rename(tmpFile.Name(), filePath)
	if err != nil {
		return err
	}

	return nil
}

func LoadCustomCsv(filePath string) error {
	baseFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("unable to open the file %s : err [%w]", filePath, err)
	}

	defer baseFile.Close()

	headerBytes := make([]byte, HeaderSize)

	_, err = baseFile.Read(headerBytes)
	if err != nil {
		return fmt.Errorf("unable to read header from the file  : err [%w]", err)
	}
	// Get The first new line so we can read the header
	firstNewLine := bytes.IndexByte(headerBytes, '\n')

	if firstNewLine == 0 {
		return fmt.Errorf("no header found %v", firstNewLine)
	}
	// Read Until the \n get the str from headerBytes array
	headerString := string(headerBytes[:firstNewLine])

	headerParts := strings.Split(headerString, ",")
	version := headerParts[0]
	startOfIndex, err := strconv.ParseInt(headerParts[1], 10, 64)
	fmt.Println(headerString, version, startOfIndex)

	// Now Build The Index Move The Cursor To Index Start Position in the file
	baseFile.Seek(startOfIndex, io.SeekStart)

	fileCursor := bufio.NewReader(baseFile)

	indexMapper = make(map[string]int64)

	for {
		// Read Tile \n
		line, err := fileCursor.ReadBytes('\n')

		if len(line) > 0 {
			// the go strings.TrimSpace Can Remove \n
			//' '   space
			// '\t'  tab
			// '\n'  newline
			// '\r'  carriage return

			lineString := strings.TrimSpace(string(line))
			parts := strings.SplitN(lineString, ",", 2)
			indexKey := parts[0]

			offset, err := strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return fmt.Errorf("error while converting str into int : err [%w]", err)
			}

			indexMapper[indexKey] = offset

		}

		//Because Last Line Can Be  Skip If we Add the EOF at the Top
		if err == io.EOF {
			break
		}
	}
	fmt.Println(indexMapper)
	return nil
}

func main() {
	// err := BuildBaseCsvIndex("/workspaces/shared-lib/main/data.csv")
	// if err != nil {
	// 	fmt.Println(err)
	// }
	err := LoadCustomCsv("/workspaces/shared-lib/main/data.csv")
	if err != nil {
		fmt.Println(err)
	}

}
