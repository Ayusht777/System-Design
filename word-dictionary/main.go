package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

type Header struct {
	FileName    string
	FileVersion int8
	IndexStart  int64
}

func BuildBaseCsvIndex(filePath string) (map[string]int64, error) {

	baseFile, err := os.Open(filePath)
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to open the file %s : err [%w]", filePath, err)
	}
	defer baseFile.Close()

	cursor := bufio.NewReader(baseFile)

	// By Default This will Zero
	var offset int64
	// Infinite For Loop Until We Will Hit The EOF
	index := map[string]int64{}
	for {
		// Read Tile \n
		line, err := cursor.ReadBytes('\n')

		// Get The Bytes Index For the First Appearance of Given Char
		keyWordEndIndex := bytes.IndexByte(line, ',')
		// Get The Keyword Bytes Until the keyWordEndIndex Bytes And Convert this into String
		keyword := string(line[:keyWordEndIndex])
		index[keyword] = offset

		offset = offset + int64(len(line))

		//Because Last Line Can Be  Skip If we Add the EOF at the Top
		if err == io.EOF {
			break
		}
	}

	return index, nil
}

func main() {
	index, err := BuildBaseCsvIndex("/workspaces/shared-lib/main/data.csv")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(index)
}
