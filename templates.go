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
	"embed"
	"fmt"
	"io/fs"
	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/ui"
	"lied/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ****************************************************************************
// initializeTemplatesFolder()
// ****************************************************************************
func initializeTemplatesFolder() {
	templateDir := filepath.Join(appDir, conf.FOLDER_TEMPLATES)
	_, err := os.Stat(templateDir)
	if os.IsNotExist(err) {
		os.MkdirAll(templateDir, 0755)
		err = extractEmbedFS(conf.TemplatesFS, "templates", templateDir)
		if err != nil {
			ui.SetStatus(fmt.Sprintf("Failed to extract embed FS: %w", err))
		}
	} else {
		ui.SetStatus(fmt.Sprintf("Templates folder already exists: %s", templateDir))
	}
}

// ****************************************************************************
// ShowTemplatesMenu()
// ****************************************************************************
func ShowTemplatesMenu(f any) {
	// Read the directory entry for the "templates" folder
	entries, err := os.ReadDir(filepath.Join(appDir, conf.FOLDER_TEMPLATES))
	ui.SetStatus("Reading templates")
	if err == nil {
		MnuTemplates = MnuTemplates.New(" Templates ", ui.GetCurrentScreen(), edit.CurrentWidget)
		for _, entry := range entries {
			if !entry.IsDir() { // Only list files, not subdirectories
				template := entry.Name()
				MnuTemplates.AddItem(template,
					strings.TrimSuffix(template, filepath.Ext(template)),
					AddTemplate,
					template,
					true,
					false)
			}
		}
		// Popup menu
		ui.PgsApp.AddPage("dlgTemplatesMenu", MnuTemplates.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgTemplatesMenu")
	} else {
		ui.SetStatus("No template found")
	}
}

// ****************************************************************************
// AddTemplate()
// ****************************************************************************
func AddTemplate(f any) {
	d := f.(string)
	DlgNewFile = DlgNewFile.Input("New File", // Title
		fmt.Sprintf("Creating a new file into %s", conf.ConfigGeneral.Workspace), // Message
		d, // Default file name
		func(rc dialog.DlgButton, idx int) {
			if rc == dialog.BUTTON_OK {
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(conf.ConfigGeneral.Workspace, DlgNewFile.Value), filepath.Join(appDir, conf.FOLDER_TEMPLATES, d), func(s1, s2 string) bool {
					// Close the file if it is open
					edit.CloseThisFile(s1)
					ui.SetStatus(fmt.Sprintf("Adding template %s to the current workspace", s2))
					// fileContent, err := conf.TemplatesFS.ReadFile(s2)
					fileContent, err := os.ReadFile(s2)
					if err != nil {
						ui.SetStatus(fmt.Sprintf("Error reading file: %v", err))
						return false
					}
					finalContent := replaceVariablesInTemplate(string(fileContent))
					if err := os.WriteFile(s1, []byte(finalContent), 0644); err != nil { // nolint: gosec
						ui.SetStatus(fmt.Sprintf("Error writing file: %w", err))
						return false
					}
					// and open it
					edit.OpenView(s1, true)
					edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
					return true
				})
			} else {
				ui.SetStatus("Canceling creating new file")
			}
			// Refresh the TrvExplorer
			edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
		},
		0,
		ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
	ui.PgsApp.AddPage("dlgNewFile", DlgNewFile.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgNewFile")
}

// ****************************************************************************
// extractEmbedFS()
// ****************************************************************************
// Source - https://stackoverflow.com/a/79533251
// Posted by guettli
// Retrieved 2026-07-13, License - CC BY-SA 4.0
// ****************************************************************************
func extractEmbedFS(fsys embed.FS, root string, destDir string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o700)
		}

		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		return os.WriteFile(destPath, data, 0o644)
	})
}

// ****************************************************************************
// replaceVariablesInTemplate()
// ****************************************************************************
func replaceVariablesInTemplate(template string) string {
	// %D  : Full directory of current file
	// %P  : Parent directory of current file
	// %F  : Full file name with directory and extension of current file
	// %f  : File name without path and with extension of current file
	// %e  : File name without path nor extension of current file
	// %L  : Line number of current file in editor
	// %T  : Current timestamp
	// %H  : Home directory of current user
	// %U  : Current user name
	// %s  : OS path separator
	// %GU : GitHub user from config file
	// %GK : GitHub key from config file
	// %GE : GitHub email from config file

	out := template
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
