package ui

import (
	"fmt"
	"lied/exifinfo"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// DisplayExifInfo()
// ****************************************************************************
// DisplayExifInfo fetches EXIF information for the given file path
// and displays it in the TblOutline table.
func DisplayExifInfo(filePath string) {
	PleaseWait()
	defer JobsDone()

	TblOutline.Clear()
	TblOutline.SetTitle("Outline")

	exifData, err := exifinfo.GetSortedExifInfo(filePath)
	if err != nil {
		SetStatus(fmt.Sprintf("Error getting EXIF info: %v", err))
		return
	}

	if len(exifData) == 0 {
		SetStatus("No EXIF information found.")
		return
	}

	// Populate the table with EXIF data
	var row = 0
	for _, entry := range exifData {
		if !strings.HasPrefix(entry.Key, "Exif") {
			TblOutline.SetCell(row, 0, tview.NewTableCell(entry.Key).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
			TblOutline.SetCell(row, 1, tview.NewTableCell(entry.Value).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
			row++
		}
	}

	SetStatus(fmt.Sprintf("Displayed EXIF information for %s", strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1]))
}
