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

	// Set table headers
	TblOutline.SetCell(0, 0, tview.NewTableCell("Property").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft).SetSelectable(false))
	TblOutline.SetCell(0, 1, tview.NewTableCell("Value").SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignLeft).SetSelectable(false))

	// Populate the table with EXIF data
	for i, entry := range exifData {
		row := i + 1 // Start from row 1 for data after headers
		TblOutline.SetCell(row, 0, tview.NewTableCell(entry.Key).SetTextColor(tcell.ColorLightCyan).SetAlign(tview.AlignLeft))
		TblOutline.SetCell(row, 1, tview.NewTableCell(entry.Value).SetTextColor(tcell.ColorWhite).SetAlign(tview.AlignLeft))
	}

	SetStatus(fmt.Sprintf("Displayed EXIF information for %s", strings.Split(filePath, "/")[len(strings.Split(filePath, "/"))-1]))
}
