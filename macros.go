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
package main

// ****************************************************************************
// IMPORTS
// ****************************************************************************
import (
	"bufio"
	"fmt"
	"lied/conf"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"lied/utils"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pgavlin/femto"
)

// ****************************************************************************
// VARS
// ****************************************************************************
var (
	Macros         map[string]string
	MnuMacros      *menu.Menu
	macroEntries   []menu.MenuItem
	snippetEntries []menu.MenuItem
	commandEntries []menu.MenuItem
)

// ****************************************************************************
// SaveMacros()
// ****************************************************************************
func SaveMacros() {
	fMac, err := os.Create(filepath.Join(appDir, conf.FILE_MACROS))
	if err == nil {
		defer fMac.Close()
		wMac := bufio.NewWriter(fMac)
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "# Macros file generated on "+time.Now().Format("20060201-150405"))
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "# These placeholders can be used into macros :")
		fmt.Fprintln(wMac, "# %D : Full directory of current file")
		fmt.Fprintln(wMac, "# %P : Parent directory of current file")
		fmt.Fprintln(wMac, "# %W : Full directory of current workspace")
		fmt.Fprintln(wMac, "# %F : Full file name with directory and extension of current file")
		fmt.Fprintln(wMac, "# %f : File name without path and with extension of current file")
		fmt.Fprintln(wMac, "# %e : File name without path nor extension of current file")
		fmt.Fprintln(wMac, "# %L : Line number of current file in editor")
		fmt.Fprintln(wMac, "# %T : Current timestamp")
		fmt.Fprintln(wMac, "# %H : Home directory of current user")
		fmt.Fprintln(wMac, "# %s : OS path separator")
		fmt.Fprintln(wMac, "# %U : Current user name")
		fmt.Fprintln(wMac, "# %GU : GitHub user from config file")
		fmt.Fprintln(wMac, "# %GK : GitHub key from config file")
		fmt.Fprintln(wMac, "# %GE : GitHub email from config file")
		fmt.Fprintln(wMac, "################################################################################")
		fmt.Fprintln(wMac, "")
		for k, v := range Macros {
			fmt.Fprintln(wMac, k+" : "+v)
		}
		wMac.Flush()
	}
}

// ****************************************************************************
// ReadMacros()
// ****************************************************************************
func ReadMacros() {
	fMac, err := os.Open(filepath.Join(appDir, conf.FILE_MACROS))
	if err == nil {
		for k := range Macros {
			delete(Macros, k)
		}
		defer fMac.Close()
		sMac := bufio.NewScanner(fMac)
		for sMac.Scan() {
			if strings.TrimSpace(string(sMac.Text())) != "" {
				if strings.TrimSpace(string(sMac.Text()[0])) != "#" {
					m := strings.Split(sMac.Text(), ":")
					Macros[strings.TrimSpace(m[0])] = strings.TrimSpace(strings.Join(m[1:], ":"))
				}
			}
		}
	}
}

// ****************************************************************************
// XeqMacro()
// ****************************************************************************
func XeqMacroORIG(k any) {
	ui.SetStatus("Executing macro : [" + k.(string) + "]")
	out := fmt.Sprintf("%s\n", XeqOutErr(replaceVariablesInMacro(k.(string))))
	edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
	MsgBox = MsgBox.OK("Macro : "+k.(string), out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// XeqMacro()
// ****************************************************************************
func XeqMacro(k any) {
	macroName := k.(string)
	ui.SetStatus("Executing macro : [" + macroName + "]")

	macroContent := Macros[macroName]
	if strings.HasPrefix(macroContent, "insert:") {
		if edit.CurrentView.ReadWrite == false {
			ui.SetStatus("Cannot insert snippet : file is read-only")
			return
		}
		snippet := replaceVariablesInMacro(macroName)
		finalSnippet := strings.TrimPrefix(snippet, "insert:")
		finalSnippet = strings.ReplaceAll(finalSnippet, "\\n", "\n")
		finalSnippet = strings.ReplaceAll(finalSnippet, "\\t", "\t")
		finalSnippet = strings.ReplaceAll(finalSnippet, "\\r", "\r")

		startPos := ui.EdtMain.Buf.Cursor.Loc
		cursorOffset := strings.Index(finalSnippet, "$0")
		if cursorOffset != -1 {
			finalSnippet = strings.Replace(finalSnippet, "$0", "", 1)
		}

		pos := ui.EdtMain.Buf.Cursor.Loc
		ui.EdtMain.Buf.Insert(pos, finalSnippet)
		if cursorOffset != -1 {
			newLoc := calculateNewLocation(finalSnippet[:cursorOffset], startPos)
			ui.EdtMain.Buf.Cursor.Loc = newLoc
		}
		ui.EdtMain.Buf.Cursor.Relocate()
		ui.SetStatus("Snippet inserted")
		return
	}

	if edit.CurrentView.FemtoBuffer.Modified() {
		edit.SaveFile()
	}

	infoBefore, errBefore := os.Stat(edit.CurrentView.FName)

	cmdString := replaceVariablesInMacro(macroName)
	output := XeqOutErr(cmdString)

	infoAfter, errAfter := os.Stat(edit.CurrentView.FName)

	if errBefore == nil && errAfter == nil {
		if infoAfter.ModTime().After(infoBefore.ModTime()) {
			ui.SetStatus("Refreshing document since it has been modified")
			edit.CurrentView.Reload()
			ui.EdtMain.Buf = edit.CurrentView.FemtoBuffer
		}
	}
	out := fmt.Sprintf("%s\n", output)
	MsgBox = MsgBox.OK("Macro : "+macroName, out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
	ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
	ui.PgsApp.ShowPage("msgBox")
}

// ****************************************************************************
// calculateNewLocation()
// ****************************************************************************
func calculateNewLocation(textBeforeCursor string, start femto.Loc) femto.Loc {
	lines := strings.Split(textBeforeCursor, "\n")
	numLines := len(lines) - 1

	newLoc := start
	newLoc.Y += numLines

	if numLines > 0 {
		// Si on a changé de ligne, la colonne repart de l'index du dernier fragment
		newLoc.X = len(lines[numLines])
	} else {
		// Si on est sur la même ligne, on ajoute simplement le nombre de caractères
		newLoc.X += len(lines[0])
	}

	return newLoc
}

// ****************************************************************************
// replaceVariablesInMacro()
// ****************************************************************************
func replaceVariablesInMacro(k string) string {
	// %D : Full directory of current file
	// %P : Parent directory of current file
	// %W : Full directory of current workspace
	// %F : Full file name with directory and extension of current file
	// %f : File name without path and with extension of current file
	// %e : File name without path nor extension of current file
	// %L : Line number of current file in editor
	// %T : Current timestamp
	// %H : Home directory of current user
	// %U  : Current user name
	// %s  : OS path separator
	// %GU : GitHub user from config file
	// %GK : GitHub key from config file
	// %GE : GitHub email from config file

	out := Macros[k]
	userDir, _ := os.UserHomeDir()
	r := strings.NewReplacer(
		"%D", utils.EscapeSpaces(filepath.Dir(edit.CurrentView.FName)),
		"%P", utils.EscapeSpaces(filepath.Base(filepath.Dir(edit.CurrentView.FName))),
		"%W", utils.EscapeSpaces(conf.ConfigGeneral.Workspace),
		"%F", utils.EscapeSpaces(edit.CurrentView.FName),
		"%f", utils.EscapeSpaces(filepath.Base(edit.CurrentView.FName)),
		"%e", utils.EscapeSpaces(filepath.Base(strings.TrimSuffix(filepath.Base(edit.CurrentView.FName), filepath.Ext(edit.CurrentView.FName)))),
		"%T", time.Now().Format("20060102-150405"),
		"%L", strconv.Itoa(edit.CurrentView.FemtoBuffer.Cursor.Y+1),
		"%s", string(os.PathSeparator),
		"%H", userDir,
		"%U", os.Getenv("USER"),
		"%GU", conf.ConfigGeneral.GitUser,
		"%GK", conf.ConfigGeneral.GitKey,
		"%GE", conf.ConfigGeneral.GitEmail,
	)
	out = r.Replace(out)
	return out
}

// ****************************************************************************
// editMacrosFile()
// ****************************************************************************
func editMacrosFile(f any) {
	edit.OpenView(filepath.Join(appDir, conf.FILE_MACROS), true)
}

// ****************************************************************************
// ShowMacrosMenu()
// ****************************************************************************
func ShowMacrosMenu() {
	ReadMacros()
	MnuMacros = MnuMacros.New(" Macros ", ui.GetCurrentScreen(), edit.CurrentWidget)
	// Sort macros
	keys := make([]string, 0, len(Macros))
	for k := range Macros {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// Read macros
	for _, k := range keys {
		MnuMacros.AddItem(k,
			k,
			XeqMacro,
			k,
			true,
			false)
	}
	// Fixed options
	if len(Macros) == 0 {
		MnuMacros.AddItem("mnuEmpty", "Empty", nil, nil, false, false)
	}
	MnuMacros.AddSeparator()
	MnuMacros.AddItem("mnuEditMacros", "Edit", editMacrosFile, nil, edit.CurrentView.ReadWrite, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgMacrosMenu", MnuMacros.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgMacrosMenu")
}
