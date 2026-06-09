package main

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type KeyMapper struct {
	FileInfo         string
	ValueSize        int
	ValueStartCursor int
}

type FileInfoMapper struct {
	FileName     string
	FileSize     int64
	FileBasePath string
	ModeTime     time.Time
	IsActive     bool
}

type StorageLoggerObj struct {
	CheckSumHash uint32
	TimeStamp    uint32
	KeySize      uint32
	ValueSize    uint32
	Key          string
	Value        string
}

const (
	HeaderSize  = 16
	MaxFileSize = 1  // 1mb
)

func GenrateHash(value string) uint32 {
	hash := crc32.ChecksumIEEE([]byte(value))
	return hash
}

func CalculateRecordSize(key, value string) int {
	return HeaderSize + len(key) + len(value)
}

func IncrementFileName(baseFileName *string) (string, error) {
	updateName := strings.TrimPrefix(*baseFileName, "data-")
	updateName = strings.TrimSuffix(updateName, ".dat")

	updateNameInNum, err := strconv.Atoi(updateName)
	if err != nil {
		return "", err
	}

	updateName = strconv.Itoa(updateNameInNum + 1)
	return "data-" + updateName + ".dat", nil
}

func FileLoader(baseDirPath string) ([]FileInfoMapper, error) {
	baseDir, err := os.ReadDir(baseDirPath)
	if err != nil {
		return nil, err
	}

	fileMapper := []FileInfoMapper{}

	for _, file := range baseDir {
		if file.IsDir() {
			fmt.Println("this base path has sub directory")
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			return nil, err
		}

		fileMapper = append(fileMapper, FileInfoMapper{
			FileName: file.Name(),
			// it can be dervie by filepath but for simplicity we are using file.Name() here
			FileSize:     fileInfo.Size(),
			FileBasePath: baseDirPath + "/" + file.Name(),
			ModeTime:     fileInfo.ModTime(),
			IsActive:     false,
			// by deafult it will be false because we are not able to determine the active status of the file at this point
			//, we will update it later when we will read the file and check if it is active or not
		})
	}
	if len(fileMapper) == 0 {
		return nil, nil
	}

	sort.Slice(fileMapper, func(i, j int) bool {
		return fileMapper[i].FileName < fileMapper[j].FileName
	})

	fileMapper[len(fileMapper)-1].IsActive = true

	return fileMapper, nil
}

func GetFileForAppend(fileInfoMapper []FileInfoMapper) FileInfoMapper {
	for _, currFileInfo := range fileInfoMapper {
		if currFileInfo.IsActive {
			return currFileInfo
		}
	}
	return FileInfoMapper{} // Return a default FileInfoMapper if no active file is found
}

func AddKeyValue(activeFileInfo *FileInfoMapper, inMemMapper *map[string]KeyMapper, key, data string) error {

	//Step 0 check for current file size
	if activeFileInfo.FileSize+int64(CalculateRecordSize(key, data)) >= MaxFileSize {
		updateActiveFileName, err := IncrementFileName(&activeFileInfo.FileName)
		if err != nil {
			return err
		}
		activeFileInfo.FileName = updateActiveFileName
	}

	//Step 1 Open The File for writing the data only
	writeFile, err := os.OpenFile(activeFileInfo.FileBasePath+activeFileInfo.FileName,
		os.O_CREATE|
			os.O_APPEND|
			os.O_RDWR,
		0644)

	if err != nil {
		return err
	}
	defer writeFile.Close()

	// Move cursor to end of the file to get the correct position for appending new data
	recordStart, _ := writeFile.Seek(0, io.SeekEnd)

	//Step 2 Write The Data To The File
	appendObject := StorageLoggerObj{
		CheckSumHash: GenrateHash(data), // 32 byte = 4 bytes
		TimeStamp:    uint32(time.Now().Unix()),
		KeySize:      uint32(len(key)),
		ValueSize:    uint32(len(data)),
		Key:          key,  // N bytes
		Value:        data, // M bytes
		// Total Fixed Size Header = 4 + 4  + 4 + 4 = 16
	}
	// writeLines, err := writeFile.Write([]byte(
	// 	fmt.Sprintf("%d%d%d%d%s%s",
	// 		appendObject.CheckSumHash,
	// 		appendObject.TimeStamp,
	// 		appendObject.KeySize,
	// 		appendObject.ValueSize,
	// 		appendObject.Key,
	// 		appendObject.Value,
	// 	)))

	// need to use write bytes in code because string has issue

	ValueStartCursor := recordStart + int64(HeaderSize) + int64(appendObject.KeySize)

	binary.Write(writeFile, binary.LittleEndian, appendObject.CheckSumHash)
	binary.Write(writeFile, binary.LittleEndian, appendObject.TimeStamp)
	binary.Write(writeFile, binary.LittleEndian, appendObject.KeySize)
	binary.Write(writeFile, binary.LittleEndian, appendObject.ValueSize)

	writeFile.Write([]byte(appendObject.Key))
	writeFile.Write([]byte(appendObject.Value))

	//Need To Update File Size So Rotaion Can work
	activeFileInfo.FileSize = int64(CalculateRecordSize(key, data))

	writeFile.Sync() // Ensure data is flushed to disk
	//step 3 add to inMemMapper

	(*inMemMapper)[key] = KeyMapper{
		FileInfo:         activeFileInfo.FileBasePath,
		ValueSize:        int(appendObject.ValueSize),
		ValueStartCursor: int(ValueStartCursor),
	}

	return nil

}

func GetValueByKey(key string, inMemMapper *map[string]KeyMapper) (string, error) {
	keyInfo, exists := (*inMemMapper)[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}
	fileSeek, err := os.Open(keyInfo.FileInfo)
	if err != nil {
		return "", err
	}
	defer fileSeek.Close()

	_, err = fileSeek.Seek(int64(keyInfo.ValueStartCursor), io.SeekStart)
	if err != nil {
		return "", err
	}
	binaryValue := make([]byte, keyInfo.ValueSize)

	_, err = io.ReadFull(fileSeek, binaryValue)
	if err != nil {
		return "", err
	}

	return string(binaryValue), nil
}

func main() {
	fileInfoMapper, err := FileLoader("/home/ayush/Desktop/SystemDesign/bitcask/db")
	if err != nil {
		fmt.Println("Error loading files:", err)
		return
	}
	// Mean The App Is Started First Time so no
	// file is there in the directory so
	//  we need to create a new file and add it to the fileInfoMapper

	if fileInfoMapper == nil {
		file, err := os.OpenFile("/home/ayush/Desktop/SystemDesign/bitcask/db/data-0001.dat", os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
		defer file.Close()

		newFileInfo := FileInfoMapper{
			FileName:     "data-1.dat",
			FileSize:     0,
			FileBasePath: "/home/ayush/Desktop/SystemDesign/bitcask/db/",
			ModeTime:     time.Now(),
			IsActive:     true,
		}
		fileInfoMapper = append(fileInfoMapper, newFileInfo)
	}

	activeFileInfo := GetFileForAppend(fileInfoMapper)
	lookUpMapper := make(map[string]KeyMapper)
	err = AddKeyValue(&activeFileInfo, &lookUpMapper, "key1", "value1")

	value, err := GetValueByKey("key1", &lookUpMapper)
	if err != nil {
		fmt.Println("Error getting value by key:", err)
		return
	}

	fmt.Println("Value for key1:", value)
}
