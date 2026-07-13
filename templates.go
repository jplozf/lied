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
	"os"
	"path/filepath"
	"strings"
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
				CreateOrOverwriteIfItAlreadyExists(filepath.Join(conf.ConfigGeneral.Workspace, DlgNewFile.Value), filepath.Join("templates", d), func(s1, s2 string) bool {
					// Close the file if it is open
					edit.CloseThisFile(s1)
					ui.SetStatus(fmt.Sprintf("Adding template %s to the current workspace", s2))
					fileContent, err := conf.TemplatesFS.ReadFile(s2)
					if err != nil {
						ui.SetStatus(fmt.Sprintf("Error reading file: %v", err))
						return false
					}
					if err := os.WriteFile(s1, fileContent, 0644); err != nil { // nolint: gosec
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
