package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
)

type Header struct {
	FileVersion int8
	IndexStart  int64
}

const HeaderSize int64 = 256

func BuildBaseCsvIndex(filePath string) (map[string]int64, error) {

	baseFile, err := os.Open(filePath)
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to open the file %s : err [%w]", filePath, err)
	}
	tmpFile, err := os.CreateTemp("/home/ayush/Desktop/SystemDesign/word-dictionary/", "temp-data-*.csv")
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to create temp file : err [%w]", err)
	}

	_, err = tmpFile.Write(make([]byte, HeaderSize))
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to write in temp file : err [%w]", err)
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
				return map[string]int64{}, fmt.Errorf("unable to write in temp file : err [%w]", err)
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
			return map[string]int64{}, fmt.Errorf("unable to write in temp file : err [%w]", err)
		}
	}

	//now write the header to the file

	//move the cursor for the tmpfile to 0
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to seek in temp file : err [%w]", err)
	}

	headerData := Header{
		FileVersion: 1,
		IndexStart:  offset,
	}
	headerString := fmt.Sprintf("%v,%d\n", headerData.FileVersion, headerData.IndexStart)
	_, err = tmpFile.Write([]byte(headerString))
	if err != nil {
		return map[string]int64{}, fmt.Errorf("unable to write in temp file : err [%w]", err)
	}

	//--------------------------------------------------
	// Flush
	//--------------------------------------------------

	err = tmpFile.Sync()
	if err != nil {
		return nil, err
	}

	tmpName := tmpFile.Name()

	baseFile.Close()
	tmpFile.Close()

	//--------------------------------------------------
	// Replace Original File
	//--------------------------------------------------

	err = os.Rename(tmpName, filePath)
	if err != nil {
		return nil, err
	}

	return index, nil
}

func main() {
	index, err := BuildBaseCsvIndex("/home/ayush/Desktop/SystemDesign/word-dictionary/data.csv")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(index)
}
