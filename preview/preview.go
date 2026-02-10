// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |/ _ \/ _` |
//	| | |  __/ (_| |
//	|_|_|\___|\__,_|
//
// ****************************************************************************
// L I E D   -   Copyright © JPL 2024
// ****************************************************************************
package preview

import (
	"archive/zip"
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"lied/conf"
	"lied/ui"
	"lied/utils"
	"os"
	"os/exec"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	_ "github.com/mattn/go-sqlite3"
)

// ****************************************************************************
// preview is the previewer module
// ****************************************************************************

// ****************************************************************************
// DisplayFilePreview()
// ****************************************************************************
func DisplayFilePreview(fName string) (string, string) {
	mtype := utils.GetMimeType(fName)
	xmtype, _ := mimetype.DetectFile(fName)

	f, err := os.OpenFile(fName, os.O_RDONLY, os.ModePerm)
	if err != nil {
		ui.SetStatus(err.Error())
	}
	defer f.Close()

	ui.TxtFileInfo.Clear()
	var preview = ""
	switch xmtype.String() {
	case "application/zip", "application/jar":
		preview = outZIP(fName)
	case "application/pdf":
		preview = outPDF(fName)
	case "application/vnd.microsoft.portable-executable":
		preview = outEXE(fName)
	case "application/vnd.sqlite3":
		preview = outSQLite3(fName)
	default:
		if xmtype.String()[0:4] == "text" {
			reader := bufio.NewReader(f)
			characters := make([]byte, conf.FILE_MAX_PREVIEW)
			_, err := reader.Read(characters)
			if err != nil {
				ui.SetStatus(err.Error())
			} else {
				preview = string(characters) + "\n"
			}
		} else {
			preview = outDefault(fName)
		}
	}
	ui.TxtFileInfo.SetText(preview)
	return mtype, xmtype.String()
}

// ****************************************************************************
// outPDF()
// ****************************************************************************
func outPDF(fName string) string {
	// See "man pdftotext" for more options.
	args := []string{
		"-layout",  // Maintain (as best as possible) the original physical layout of the text.
		"-nopgbrk", // Don't insert page breaks (form feed characters) between pages.
		fName,      // The input file.
		"-",        // Send the output to stdout.
	}
	out, err := exec.CommandContext(context.Background(), "pdftotext", args...).Output()
	if err != nil {
		return err.Error()
	}
	return string(out)
}

// ****************************************************************************
// outEXE()
// ****************************************************************************
func outEXE(fName string) string {
	args := []string{
		fName,
	}
	out, err := exec.CommandContext(context.Background(), "exiftool", args...).Output()
	if err != nil {
		return err.Error()
	}
	return string(out)
}

// ****************************************************************************
// outZIP()
// ****************************************************************************
func outZIP(fName string) string {
	zipListing, err := zip.OpenReader(fName)
	if err != nil {
		return err.Error()
	}
	defer zipListing.Close()
	zList := ""
	nFiles := 0
	for _, file := range zipListing.File {
		nFiles++
		zList += fmt.Sprintf("#%05d > %s\n", nFiles, file.Name)
	}
	return fmt.Sprintf("Total files in ZIP archive : %d\n", nFiles) + zList
}

// ****************************************************************************
// outDefault()
// ****************************************************************************
func outDefault(fName string) string {
	return outExif(fName)
}

// ****************************************************************************
// outExif()
// ****************************************************************************
func outExif(fName string) string {
	args := []string{
		fName,
	}
	out, err := exec.CommandContext(context.Background(), "exiftool", args...).Output()
	if err != nil {
		return err.Error()
	}
	res := strings.Index(string(out), "\n") // Skip the first line which displays "Exif Tools version..."
	return string(out)[res+1:]
}

// ****************************************************************************
// outSQLite3()
// ****************************************************************************
func outSQLite3(fName string) string {
	db, err := sql.Open("sqlite3", fName)
	if err != nil {
		return err.Error()
	}
	defer db.Close()

	rows, err := db.Query("SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';")
	if err != nil {
		return err.Error()
	}
	defer rows.Close()

	zTables := ""
	nTables := 0
	for rows.Next() {
		var name string
		err = rows.Scan(&name)
		if err != nil {
			return err.Error()
		}
		nTables++
		zTables += fmt.Sprintf("#%05d > %s\n", nTables, name)
	}
	return fmt.Sprintf("Total tables in SQLite3 database : %d\n", nTables) + zTables
}

// ****************************************************************************
// DisplayExif()
// ****************************************************************************
func DisplayExif(fName string) {
	ui.TxtFileInfo.SetText(outExif(fName))
}
