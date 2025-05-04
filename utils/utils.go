// ****************************************************************************
//
//	 _____ _____ _____ _____
//	|   __|     |   __|  |  |
//	|  |  |  |  |__   |     |
//	|_____|_____|_____|__|__|
//
// ****************************************************************************
// G O S H   -   Copyright © JPL 2023
// ****************************************************************************
package utils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	suffixes [5]string
	CpuUsage float64
)

// ****************************************************************************
// Round()
// ****************************************************************************
func Round(val float64, roundOn float64, places int) (newVal float64) {
	var round float64
	pow := math.Pow(10, float64(places))
	digit := pow * val
	_, div := math.Modf(digit)
	if div >= roundOn {
		round = math.Ceil(digit)
	} else {
		round = math.Floor(digit)
	}
	newVal = round / pow
	return
}

// ****************************************************************************
// HumanFileSize()
// ****************************************************************************
func HumanFileSize(size float64) string {
	if size == 0 {
		return "0 B"
	} else {
		suffixes[0] = "B"
		suffixes[1] = "KB"
		suffixes[2] = "MB"
		suffixes[3] = "GB"
		suffixes[4] = "TB"

		base := math.Log(size) / math.Log(1024)
		getSize := Round(math.Pow(1024, base-math.Floor(base)), .5, 2)
		getSuffix := suffixes[int(math.Floor(base))]
		return strconv.FormatFloat(getSize, 'f', -1, 64) + " " + string(getSuffix)
	}
}

// ****************************************************************************
// IsTextFile()
// ****************************************************************************
func IsTextFile(fName string) bool {
	readFile, err := os.Open(fName)
	if err != nil {
		return false
	}
	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	fileScanner.Scan()

	return (utf8.ValidString(string(fileScanner.Text())))
}

// ****************************************************************************
// GetMimeType()
// ****************************************************************************
func GetMimeType(fName string) string {
	readFile, err := os.Open(fName)
	if err != nil {
		return "NIL"
	}
	defer readFile.Close()
	// Read the response body as a byte slice
	bytes, err := ioutil.ReadAll(readFile)
	if err != nil {
		return "NIL"
	}
	mimeType := http.DetectContentType(bytes)
	return mimeType
}

// ****************************************************************************
// NumberOfFilesAndFolders()
// ****************************************************************************
func NumberOfFilesAndFolders(path string) (int, int, error) {
	nFiles := 0
	nFolders := 0

	files, err := os.ReadDir(path)
	if err != nil {
		return 0, 0, err
	}
	for _, file := range files {
		if file.IsDir() {
			nFolders++
		} else {
			nFiles++
		}
	}
	return nFiles, nFolders, nil
}

// ****************************************************************************
// GetSha256()
// ****************************************************************************
func GetSha256(fName string) (string, error) {
	file, err := os.Open(fName)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hashSHA256 := sha256.New()
	if _, err := io.Copy(hashSHA256, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hashSHA256.Sum(nil)), nil
}

// ****************************************************************************
// GetCPUSample()
// ****************************************************************************
func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val // tally up all the numbers to get total ticks
				if i == 4 {  // idle is the 5th field in the cpu line
					idle = val
				}
			}
			return
		}
	}
	return
}

// ****************************************************************************
// GetCpuUsage()
// ****************************************************************************
func GetCpuUsage() {
	for {
		idle0, total0 := getCPUSample()
		time.Sleep(3 * time.Second)
		idle1, total1 := getCPUSample()
		idleTicks := float64(idle1 - idle0)
		totalTicks := float64(total1 - total0)
		CpuUsage = 100 * (totalTicks - idleTicks) / totalTicks
	}
}

// ****************************************************************************
// DirSize()
// ****************************************************************************
func DirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

// ****************************************************************************
// FilenameWithoutExtension()
// ****************************************************************************
func FilenameWithoutExtension(fileName string) string {
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

// ****************************************************************************
// GetAllFilesFromFolder()
// ****************************************************************************
func GetAllFilesFromFolder(folder string) ([]string, error) {
	var files []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, nil
}

// ****************************************************************************
// ZipFile()
// ****************************************************************************
func ZipFile(fArchive string, fName string) {
	arc, err := os.Create(fArchive)
	if err != nil {
		log.Fatal(err)
	} else {
		defer arc.Close()
		zipWriter := zip.NewWriter(arc)
		f1, err := os.Open(fName)
		if err != nil {
			log.Fatal(err)
		} else {
			w1, err := zipWriter.Create(filepath.Base(fName))
			if err != nil {
				log.Fatal(err)
			} else {
				if _, err := io.Copy(w1, f1); err != nil {
					log.Fatal(err)
				} else {
					zipWriter.Close()
				}
			}
		}
	}
}

// ****************************************************************************
// ZipFolder()
// ****************************************************************************
func ZipFolder(fArchive string, fName string) {
	zipFile, err := os.Create(fArchive)
	if err != nil {
		log.Fatal(err)
	}
	zipWriter := zip.NewWriter(zipFile)
	err = filepath.Walk(fName, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if path == fName {
			return nil
		}
		pathInZip := strings.Replace(path, strings.Replace(fName, "./", "", 1)+"/", "", 1)
		if info.IsDir() {
			_, err := zipWriter.Create(pathInZip + "/")
			if err != nil {
				log.Fatal(err)
			}
			return nil
		}
		zipFileWriter, err := zipWriter.Create(pathInZip)
		if err != nil {
			log.Fatal(err)
		}

		fileDescriptor, err := os.Open(path)
		if err != nil {
			log.Fatal(err)
		}

		_, err = io.Copy(zipFileWriter, fileDescriptor)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	err = zipWriter.Close()
	if err != nil {
		log.Fatal(err)
	}

	err = zipFile.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// ****************************************************************************
// GetFilenameWhichDoesntExist()
// ****************************************************************************
func GetFilenameWhichDoesntExist(fName string) string {
	if IsFileExist(fName) {
		f := fName
		i := 1
		for IsFileExist(f) {
			f = FilenameWithoutExtension(fName) + fmt.Sprintf("(%d)", i) + filepath.Ext(fName)
			i++
		}
		return f
	} else {
		return fName
	}
}

// ****************************************************************************
// IsFileExist()
// ****************************************************************************
func IsFileExist(fName string) bool {
	if _, err := os.Stat(fName); err == nil {
		return true
	} else {
		return false
	}
}

// ****************************************************************************
// CopyFile()
// ****************************************************************************
func CopyFile(source string, dest string) (err error) {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return
}

// ****************************************************************************
// CopyDir()
// ****************************************************************************
func CopyDir(source string, dest string) (err error) {

	// get properties of source dir
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	// create dest dir

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)

	objects, err := directory.Readdir(-1)

	for _, obj := range objects {

		sourcefilepointer := source + "/" + obj.Name()

		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			// create sub-directories - recursively
			err = CopyDir(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			// perform copy
			err = CopyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return
}

// ****************************************************************************
// CopyFileIntoFolder()
// ****************************************************************************
func CopyFileIntoFolder(source string, dest string) (err error) {
	destFile := filepath.Join(dest, filepath.Base(source))
	return CopyFile(source, destFile)
}

// ****************************************************************************
// CopyFileIntoFolder()
// ****************************************************************************
func CopyFolderIntoFolder(source string, dest string) (err error) {
	destFolder := filepath.Join(dest, filepath.Base(source))
	return CopyDir(source, destFolder)
}

// ****************************************************************************
// IsAsciiPrintable()
// ****************************************************************************
func IsAsciiPrintable(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// ****************************************************************************
// RandomHex()
// ****************************************************************************
func RandomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ****************************************************************************
// If() Ternary Operator
// ****************************************************************************
func If[T any](cond bool, vtrue, vfalse T) T {
	if cond {
		return vtrue
	}
	return vfalse
}

// ****************************************************************************
// Xeq()
// ****************************************************************************
func Xeq(dir string, args ...string) (string, string) {
	baseCmd := args[0]
	cmdArgs := args[1:]
	xeq := exec.Command(baseCmd, cmdArgs...)
	xeq.Dir = dir
	var outb, errb bytes.Buffer
	xeq.Stdout = &outb
	xeq.Stderr = &errb
	xeq.Run()
	return outb.String(), errb.String()
}
