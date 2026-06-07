package main

import (
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"sort"
	"time"
)

type KeyMapper struct {
	FileInfo         string
	ValueSize        int
	ValueStartCursor int
}

type FileInfoMapper struct {
	FileName string
	FileSize int64
	FilePath string
	ModeTime time.Time
	IsActive bool
}

type StorageLoggerObj struct {
	CheckSumHash uint32
	TimeStamp    uint32
	KeySize      uint32
	ValueSize    uint32
	Key          string
	Value        string
}

func GenrateHash(value string) uint32 {
	hash := crc32.ChecksumIEEE([]byte(value))
	return hash
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
			FileSize: fileInfo.Size(),
			FilePath: baseDirPath + "/" + file.Name(),
			ModeTime: fileInfo.ModTime(),
			IsActive: false,
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

func AddKeyValue(activeFileInfo FileInfoMapper, inMemMapper *map[string]KeyMapper, key, data string) error {
	//Step 1 Open The File for writing the data only
	writeFile, err := os.OpenFile(activeFileInfo.FilePath,
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
		CheckSumHash: GenrateHash(data),
		TimeStamp:    uint32(time.Now().Unix()),
		KeySize:      uint32(len(key)),
		ValueSize:    uint32(len(data)),
		Key:          key,
		Value:        data,
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
   
	//#Todo need to use write bytes in code

	writeFile.Sync() // Ensure data is flushed to disk
	//step 3 add to inMemMapper

	(*inMemMapper)[key] = KeyMapper{
		FileInfo:         activeFileInfo.FilePath,
		ValueSize:        int(appendObject.ValueSize),
		ValueStartCursor: int(recordStart) + writeLines + int(appendObject.KeySize),
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
			FileName: "data-0001.dat",
			FileSize: 0,
			FilePath: "/home/ayush/Desktop/SystemDesign/bitcask/db/data-0001.dat",
			ModeTime: time.Now(),
			IsActive: true,
		}
		fileInfoMapper = append(fileInfoMapper, newFileInfo)
	}

	activeFileInfo := GetFileForAppend(fileInfoMapper)
	lookUpMapper := make(map[string]KeyMapper)
	err = AddKeyValue(activeFileInfo, &lookUpMapper, "key1", "value1")

	value, err := GetValueByKey("key1", &lookUpMapper)
	if err != nil {
		fmt.Println("Error getting value by key:", err)
		return
	}

	fmt.Println("Value for key1:", value)
}
