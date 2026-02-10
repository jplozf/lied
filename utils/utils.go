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
	"math/big"
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
	// This method reads only the first line, it's faster but could failed sometimes
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
// IsTextFile2()
// ****************************************************************************
// ****************************************************************************
// IsBinaryFile()
// ****************************************************************************
func IsBinaryFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return true // Assume binary if we can't open it
	}
	defer file.Close()

	// Read the first 1024 bytes
	buffer := make([]byte, 1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return true // Assume binary if there's an error reading
	}

	// Check for null bytes or a high percentage of non-printable characters
	nonPrintableCount := 0
	for i := 0; i < n; i++ {
		if buffer[i] == 0 {
			return true // Contains null byte, definitely binary
		}
		if !unicode.IsPrint(rune(buffer[i])) && !unicode.IsSpace(rune(buffer[i])) {
			nonPrintableCount++
		}
	}

	// If more than 10% of characters are non-printable (excluding whitespace), consider it binary
	if n > 0 && float64(nonPrintableCount)/float64(n) > 0.10 {
		return true
	}

	return false
}

// ****************************************************************************
// BytesToHexAndASCII()
// ****************************************************************************
func BytesToHexAndASCII(data []byte) string {
	var output strings.Builder

	bytesPerLine := 16

	for i := 0; i < len(data); i += bytesPerLine {
		// Offset
		fmt.Fprintf(&output, "[yellow]%08X[white]   ", i)

		lineBytes := data[i:If(i+bytesPerLine > len(data), len(data), i+bytesPerLine)]

		// Hexadecimal representation
		for j, b := range lineBytes {
			fmt.Fprintf(&output, "%02X", b)
			if (j+1)%2 == 0 {
				output.WriteString(" ")
			}
		}
		// Pad with spaces if the line is shorter than bytesPerLine
		if len(lineBytes) < bytesPerLine {
			output.WriteString(strings.Repeat("   ", bytesPerLine-len(lineBytes)))
		}
		output.WriteString(" ")

		// ASCII representation
		for _, b := range lineBytes {
			if unicode.IsPrint(rune(b)) {
				output.WriteRune(rune(b))
			} else {
				output.WriteRune('.') // Replace non-printable with a dot
			}
		}
		output.WriteString("\n")
	}
	return output.String()
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

// ****************************************************************************
// EscapeSpaces()
// ****************************************************************************
func EscapeSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "\\ ")
}

// ****************************************************************************
// IsDir()
// ****************************************************************************
func IsDir(p string) bool {
	fi, err := os.Stat(p)
	if err != nil {
		return false
	}
	return fi.Mode().IsDir()
}

// ****************************************************************************
// ZipIt()
// ****************************************************************************
// Zips "./input" into "./output.zip"
func ZipIt(source string, target string) error {
	// 1. Create a ZIP file and zip.Writer
	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := zip.NewWriter(f)
	defer writer.Close()

	// 2. Go through all the files of the source
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 3. Create a local file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// set compression
		header.Method = zip.Deflate

		// 4. Set relative path of a file as the header name
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		// 5. Create writer for the file header and save content of the file
		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}

var (
	adjectives = []string{
		"agile", "ancient", "arctic", "arid", "atomic", "azure", "bold", "bright",
		"broken", "bronze", "calm", "celestial", "chill", "clear", "cold", "cosmic",
		"crimson", "cryptic", "crystal", "dark", "dawn", "deep", "digital", "divine",
		"dusty", "eager", "electric", "emerald", "epic", "eternal", "faint", "fancy",
		"fast", "feral", "fierce", "flat", "floral", "flying", "formal", "frozen",
		"gentle", "giant", "glass", "global", "glossy", "golden", "grand", "gray",
		"green", "hidden", "hollow", "honest", "huge", "humble", "icy", "inner",
		"iron", "ivory", "jade", "jolly", "jumpy", "keen", "kind", "large",
		"light", "liquid", "little", "lucky", "lunar", "magic", "misty", "modern",
		"mystic", "narrow", "neon", "noble", "oceanic", "old", "opal", "outer",
		"pale", "patient", "plain", "prime", "proud", "pure", "quiet", "rapid",
		"rare", "red", "royal", "rugged", "secret", "shining", "silent", "silver",
		"smooth", "solar", "stark", "steady", "stellar", "swift", "vibrant",
	}

	nouns = []string{
		"anchor", "apple", "arrow", "atlas", "atom", "axis", "beacon", "beam",
		"bird", "bison", "boulder", "breeze", "bridge", "cactus", "canyon", "castle",
		"cliff", "cloud", "comet", "crag", "crane", "crest", "crystal", "desert",
		"door", "dune", "eagle", "earth", "echo", "edge", "ember", "field",
		"flame", "flower", "forest", "forge", "fossil", "fountain", "fox", "galaxy",
		"garden", "gate", "glacier", "glass", "grove", "harbor", "hawk", "heart",
		"hill", "island", "jungle", "lake", "leaf", "light", "lion", "marsh",
		"maze", "meadow", "mirror", "mist", "moon", "mountain", "nebula", "night",
		"node", "ocean", "orbit", "owl", "path", "peak", "pebble", "pine",
		"planet", "plateau", "pond", "prism", "pulse", "rain", "reef", "ridge",
		"river", "rock", "root", "sand", "shadow", "shell", "shield", "sky",
		"snow", "spark", "spire", "star", "stone", "storm", "stream", "sun",
		"temple", "thistle", "thorn", "thunder", "tiger", "tower", "trail", "tree",
		"tundra", "valley", "vessel", "voice", "vortex", "wave", "wind", "wolf",
		"zenith", "zone",
	}
)

// secureInt returns a random integer in the range [0, max)
func secureInt(max int) int {
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(nBig.Int64())
}

func GenerateSecureFilename() string {
	adj := adjectives[secureInt(len(adjectives))]
	noun := nouns[secureInt(len(nouns))]
	num := secureInt(9000) + 1000

	// Returns format like "cosmic-nebula-4029"
	return fmt.Sprintf("%s-%s-%d", adj, noun, num)
}

// ****************************************************************************
// IsSQLite3()
// ****************************************************************************
func IsSQLite3(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// The magic string is exactly 16 bytes long
	header := make([]byte, 16)
	_, err = f.Read(header)
	if err != nil {
		return false
	}

	sqliteHeader := []byte("SQLite format 3\x00")
	return bytes.Equal(header, sqliteHeader)
}

var compactDigits = map[rune][3]string{
	'0': {
		"▄▀▀▄",
		"█  █",
		"▀▄▄▀",
	},
	'1': {
		" ▄  ",
		" █  ",
		" █  ",
	},
	'2': {
		"▀▀▀▄",
		"▄▀▀ ",
		"▀▀▀▀",
	},
	'3': {
		"▀▀▀▄",
		" ▀▀▄",
		"▀▀▀▀",
	},
	'4': {
		"█  █",
		"▀▀▀█",
		"   ▀",
	},
	'5': {
		"█▀▀▀",
		"▀▀▀▄",
		"▀▀▀▀",
	},
	'6': {
		"▄▀▀▀",
		"█▀▀▄",
		"▀▄▄▀",
	},
	'7': {
		"▀▀▀█",
		"  █ ",
		"  ▀ ",
	},
	'8': {
		"▄▀▀▄",
		"▄▀▀▄",
		"▀▄▄▀",
	},
	'9': {
		"▄▀▀▄",
		"▀▀▀█",
		"▀▀▀▀",
	},
	':': {
		" ▄ ",
		"   ",
		" ▀ ",
	},
}

// ****************************************************************************
// PrintCompactNumber()
// ****************************************************************************
func PrintCompactNumber(input string) {
	for row := 0; row < 3; row++ {
		line := ""
		for _, char := range input {
			if val, ok := compactDigits[char]; ok {
				line += val[row] + " "
			}
		}
		fmt.Println(line)
	}
}
