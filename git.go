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
	"fmt"
	"lied/conf"
	"lied/dialog"
	"lied/edit"
	"lied/menu"
	"lied/ui"
	"lied/utils"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ****************************************************************************
// VARS
// ****************************************************************************
var (
	MnuGit *menu.Menu
)

// ****************************************************************************
// IsInsideGitWorkTree()
// ****************************************************************************
func IsInsideGitWorkTree() bool {
	rc := false
	out := XeqOut("git rev-parse --is-inside-work-tree")
	if out[:4] == "true" {
		rc = true
	}
	return rc
}

// ****************************************************************************
// IsFileGitTracked()
// ****************************************************************************
func IsFileGitTracked(f string) bool {
	rc := false
	out := XeqOut("git ls-files --error-unmatch " + utils.EscapeSpaces(f))
	b := filepath.Base(f)
	if out[:len(b)] == b {
		ui.SetStatus(fmt.Sprintf("File [%s] is git-tracked", f))
		rc = true
	} else {
		ui.SetStatus(fmt.Sprintf("File [%s] is NOT git-tracked", f))
	}
	return rc
}

// ****************************************************************************
// DoGitStatus()
// ****************************************************************************
func DoGitStatus(f any) {
	if IsInsideGitWorkTree() {
		out := fmt.Sprintf("Current local commit : %s\n\n", XeqOut("git rev-parse --short HEAD"))
		out += fmt.Sprintf("%s\n", XeqOut("git status"))
		out += fmt.Sprintf("%s\n", XeqOut("git diff"))
		MsgBox = MsgBox.OK("Git Status", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitLog()
// ****************************************************************************
func DoGitLog(f any) {
	if IsInsideGitWorkTree() {
		out := fmt.Sprintf("%s", XeqOut("git log"))
		MsgBox = MsgBox.OK("Git Log", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitBang()
// ****************************************************************************
func DoGitBang(f any) {
	if !IsInsideGitWorkTree() {
		pjname := filepath.Base(filepath.Dir(edit.CurrentView.FName))
		DlgYesNo1 = DlgYesNo1.YesNo("Git Bang", // Title
			fmt.Sprintf("This will initialize and push a Git environment\nfor your project \"%s\".\n\nAre you sure you want to proceed ?", pjname), // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					if conf.ConfigGeneral.GitUser == "" {
						ui.SetStatus("Git User is not yet configured")
					} else {
						DlgYesNo2 = DlgYesNo2.YesNo("Git Bang", // Title
							"The following repository should already exist :\n\nhttps://github.com/"+conf.ConfigGeneral.GitUser+"/"+pjname+"\n\nHas the repository already been created ?", // Message
							func(rc dialog.DlgButton, idx int) {
								if rc == dialog.BUTTON_YES {
									url := "https://" + conf.ConfigGeneral.GitKey + "@github.com/" + conf.ConfigGeneral.GitUser + "/" + pjname
									Xeq("git init")
									Xeq("git add .")
									Xeq("git commit -m \"First Commit\"")
									Xeq("git remote add origin " + url)
									Xeq("git branch -M main")
									Xeq("git pull --rebase origin main")
									Xeq("git push -u origin main")
									currentDir := filepath.Dir(edit.CurrentView.FName)
									if !utils.IsFileExist(filepath.Join(currentDir, "README.md")) {
										createFile(filepath.Join(currentDir, "README.md"), "#"+pjname)
										Xeq("git add README.md")
									}
									if !utils.IsFileExist(filepath.Join(currentDir, ".gitignore")) {
										createFile(filepath.Join(currentDir, ".gitignore"),
											`# This file is used to ignore files which are generated
# ----------------------------------------------------------------------------

github.key
*~
*.autosave
*.a
*.core
*.moc
*.o
*.obj
*.orig
*.rej
*.so
*.so.*
*_pch.h.cpp
*_resource.rc
*.qm
.#*
*.*#
core
!core/
tags
.DS_Store
.directory
*.debug
Makefile*
*.prl
*.app
moc_*.cpp
ui_*.h
qrc_*.cpp
Thumbs.db
*.res
*.rc
/.qmake.cache
/.qmake.stash

# qtcreator generated files
*.pro.user*
*.qbs.user*
CMakeLists.txt.user*

# xemacs temporary files
*.flc

# Vim temporary files
.*.swp

# Visual Studio generated files
*.ib_pdb_index
*.idb
*.ilk
*.pdb
*.sln
*.suo
*.vcproj
*vcproj.*.*.user
*.ncb
*.sdf
*.opensdf
*.vcxproj
*vcxproj.*

# MinGW generated files
*.Debug
*.Release

# Python byte code
*.pyc

# Binaries
# --------
*.dll
*.exe

# Directories with generated files
.moc/
.obj/
.pch/
.rcc/
.uic/
/build*/
`)
										Xeq("git add .gitignore")
									}
									DoGitStatus("dummy")
								} else {
									ui.SetStatus("Aborting Git Bang")
								}
							},
							0,
							ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
						ui.PgsApp.AddPage("dlgYesNo2", DlgYesNo2.Popup(), true, false)
						ui.PgsApp.ShowPage("dlgYesNo2")
					}
				} else {
					ui.SetStatus("Aborting Git Bang")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo1", DlgYesNo1.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo1")
	} else {
		ui.SetStatus("Git repository already created")
	}
}

// ****************************************************************************
// DoGitInit()
// ****************************************************************************
func DoGitInit(f any) {
	if !IsInsideGitWorkTree() {
		pjname := filepath.Base(filepath.Dir(edit.CurrentView.FName))
		DlgYesNo1 = DlgYesNo1.YesNo("Git Init", // Title
			fmt.Sprintf("This will initialize a Git environment\nfor your project \"%s\".\n\nAre you sure you want to proceed ?", pjname), // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					if conf.ConfigGeneral.GitUser == "" {
						ui.SetStatus("Git User is not yet configured")
					} else {
						currentDir := filepath.Dir(edit.CurrentView.FName)
						if !utils.IsFileExist(filepath.Join(currentDir, "README.md")) {
							createFile(filepath.Join(currentDir, "README.md"), "#"+pjname)
						}
						if !utils.IsFileExist(filepath.Join(currentDir, ".gitignore")) {
							createFile(filepath.Join(currentDir, ".gitignore"),
								`# This file is used to ignore files which are generated
# ----------------------------------------------------------------------------

github.key
*~
*.autosave
*.a
*.core
*.moc
*.o
*.obj
*.orig
*.rej
*.so
*.so.*
*_pch.h.cpp
*_resource.rc
*.qm
.#*
*.*#
core
!core/
tags
.DS_Store
.directory
*.debug
Makefile*
*.prl
*.app
moc_*.cpp
ui_*.h
qrc_*.cpp
Thumbs.db
*.res
*.rc
/.qmake.cache
/.qmake.stash

# qtcreator generated files
*.pro.user*
*.qbs.user*
CMakeLists.txt.user*

# xemacs temporary files
*.flc

# Vim temporary files
.*.swp

# Visual Studio generated files
*.ib_pdb_index
*.idb
*.ilk
*.pdb
*.sln
*.suo
*.vcproj
*vcproj.*.*.user
*.ncb
*.sdf
*.opensdf
*.vcxproj
*vcxproj.*

# MinGW generated files
*.Debug
*.Release

# Python byte code
*.pyc

# Binaries
# --------
*.dll
*.exe

# Directories with generated files
.moc/
.obj/
.pch/
.rcc/
.uic/
/build*/
`)
						}
						Xeq("git init")
						// Xeq("git add .")
						DoGitStatus("dummy")
						Xeq("git branch -M main")
						go edit.UpdateStatus()
						edit.ShowTreeDir(conf.ConfigGeneral.Workspace, conf.ConfigGeneral.ShowHidden)
					}
				} else {
					ui.SetStatus("Aborting Git Init")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo1", DlgYesNo1.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo1")
	} else {
		ui.SetStatus("Git repository already created")
	}
}

// ****************************************************************************
// DoGitCommitPush()
// ****************************************************************************
func DoGitCommitPush(f any) {
	if IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Commit & Push", // Title
			"Please, enter the message :", // Message
			"",                            // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Committing...\n%s", XeqOut("git commit -a -m \""+DlgInput.Value+"\""))
					branch := XeqRaw("git rev-parse --abbrev-ref HEAD")
					out += fmt.Sprintf("\n\nPushing...\n%s", XeqOut("git push origin "+branch))

					MsgBox = MsgBox.OK("Git Commit & Push", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Commit & Push")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitConfigure()
// ****************************************************************************
func DoGitConfigure(f any) {
	form := tview.NewForm()
	form.SetTitle("Git Configure")
	form.AddInputField("User", conf.ConfigGeneral.GitUser, 40, nil, nil)
	form.AddInputField("Key", conf.ConfigGeneral.GitKey, 40, nil, nil)
	form.AddInputField("Email", conf.ConfigGeneral.GitEmail, 40, nil, nil)
	form.AddButton("OK", func() {
		conf.ConfigGeneral.GitUser = form.GetFormItem(0).(*tview.InputField).GetText()
		conf.ConfigGeneral.GitKey = form.GetFormItem(1).(*tview.InputField).GetText()
		conf.ConfigGeneral.GitEmail = form.GetFormItem(2).(*tview.InputField).GetText()
		ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
		ui.App.SetFocus(edit.CurrentWidget)
	})
	form.AddButton("Cancel", func() {
		ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
		ui.App.SetFocus(ui.EdtMain)
	})
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
			ui.App.SetFocus(edit.CurrentWidget)
			return nil
		}
		return event
	})

	form.SetButtonsAlign(tview.AlignCenter)
	form.SetButtonBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	form.SetButtonTextColor(tview.Styles.PrimaryTextColor)
	form.SetBackgroundColor(tview.Styles.ContrastBackgroundColor).SetBorderPadding(0, 0, 0, 0)
	form.SetBorder(true).
		SetBackgroundColor(tview.Styles.ContrastBackgroundColor).
		SetBorderPadding(1, 1, 1, 1)

	ui.PgsApp.AddPage("myForm",
		tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(form, 11, 1, true).
				AddItem(nil, 0, 1, false), 49, 1, true).
			AddItem(nil, 0, 1, false),
		true, false)
	ui.PgsApp.ShowPage("myForm")
}

// ****************************************************************************
// DoGitFetch()
// ****************************************************************************
func DoGitFetch(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Fetch", // Title
			"The Git Fetch will fetch the remote version but no merging is processed locally.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Fetching...\n%s", XeqOut("git fetch origin"))
					MsgBox = MsgBox.OK("Git Fetch", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Fetch")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitPull()
// ****************************************************************************
func DoGitPull(f any) {
	if IsInsideGitWorkTree() {
		branch := XeqOut("git rev-parse --abbrev-ref HEAD")
		DlgYesNo = DlgYesNo.YesNo("Git Pull", // Title
			"The Git Pull will fetch the remote version and merge it with you local branch which is currently ["+branch+"].\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Pulling...\n%s", XeqOut("git pull origin "+branch))
					MsgBox = MsgBox.OK("Git Pull", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Pull")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitPush()
// ****************************************************************************
func DoGitPush(f any) {
	if IsInsideGitWorkTree() {
		branch := XeqRaw("git rev-parse --abbrev-ref HEAD")
		out := fmt.Sprintf("Pushing...\n%s", XeqOut("git push origin "+branch))
		MsgBox = MsgBox.OK("Git Push", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
		ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
		ui.PgsApp.ShowPage("msgBox")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitCommit()
// ****************************************************************************
func DoGitCommit(f any) {
	if IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Commit", // Title
			"Please, enter the message :", // Message
			"",                            // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Committing...\n%s", XeqOut("git commit -a -m \""+DlgInput.Value+"\""))
					MsgBox = MsgBox.OK("Git Commit", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Commit")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitClone()
// ****************************************************************************
func DoGitClone(f any) {
	if !IsInsideGitWorkTree() {
		DlgInput = DlgInput.Input("Git Clone", // Title
			"Please, enter the distant repository to clone locally :", // Message
			"https://github.com/repo/project.git",                     // Default value
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_OK {
					out := fmt.Sprintf("Cloning...\n%s", XeqOut("git clone \""+DlgInput.Value+"\""))
					MsgBox = MsgBox.OK("Git Clone", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Clone")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgInput", DlgInput.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgInput")
	} else {
		ui.SetStatus("Already in a Git repository")
	}
}

// ****************************************************************************
// DoGitAdd()
// ****************************************************************************
func DoGitAdd(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Add", // Title
			"This will add the file :\n\n"+f.(string)+"\n\nto the Git tracking.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					b := filepath.Base(f.(string))
					out := fmt.Sprintf("Adding...\n%s", XeqOut("git add ./"+utils.EscapeSpaces(b)))
					MsgBox = MsgBox.OK("Git Add", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Add")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// DoGitAddAll()
// ****************************************************************************
func DoGitAddAll(f any) {
	if IsInsideGitWorkTree() {
		DlgYesNo = DlgYesNo.YesNo("Git Add All", // Title
			"This will add all the files to the Git tracking.\n\nAre you sure you want to proceed ?", // Message
			func(rc dialog.DlgButton, idx int) {
				if rc == dialog.BUTTON_YES {
					out := fmt.Sprintf("Adding...\n%s", XeqOut("git add ."))
					MsgBox = MsgBox.OK("Git Add All", out, nil, 0, ui.GetCurrentScreen(), edit.CurrentWidget)
					ui.PgsApp.AddPage("msgBox", MsgBox.Popup(), true, false)
					ui.PgsApp.ShowPage("msgBox")
				} else {
					ui.SetStatus("Aborting Git Add All")
				}
			},
			0,
			ui.GetCurrentScreen(), edit.CurrentWidget) // Focus return
		ui.PgsApp.AddPage("dlgYesNo", DlgYesNo.Popup(), true, false)
		ui.PgsApp.ShowPage("dlgYesNo")
	} else {
		ui.SetStatus("No Git repository found")
	}
}

// ****************************************************************************
// ShowGitMenu()
// ****************************************************************************
func ShowGitMenu() {
	MnuGit = MnuGit.New(" Git Tracking ", ui.GetCurrentScreen(), edit.CurrentWidget)
	// Menu Options
	MnuGit.AddItem("mnuGitStatus", "Status", DoGitStatus, nil, true, false)
	MnuGit.AddItem("mnuGitLog", "Log", DoGitLog, nil, true, false)
	MnuGit.AddItem("mnuGitAddAll", "Add All (.)", DoGitAddAll, nil, true, false)
	MnuGit.AddItem("mnuGitCommit", "Commit", DoGitCommit, nil, true, false)
	MnuGit.AddItem("mnuGitPush", "Push", DoGitPush, nil, true, false)
	MnuGit.AddItem("mnuGitCommitPush", "Commit & Push", DoGitCommitPush, nil, true, false)
	MnuGit.AddItem("mnuGitFetch", "Fetch", DoGitFetch, nil, true, false)
	MnuGit.AddItem("mnuGitPull", "Pull (Fetch & Merge)", DoGitPull, nil, true, false)
	MnuGit.AddItem("mnuGitInit", "Initialize", DoGitInit, nil, true, false)
	MnuGit.AddItem("mnuGitBang", "Initialize & Push", DoGitBang, nil, true, false)
	MnuGit.AddItem("mnuGitClone", "Clone", DoGitClone, nil, true, false)
	MnuGit.AddItem("mnuGitConfigure", "Configure", DoGitConfigure, nil, true, false)
	// Popup menu
	ui.PgsApp.AddPage("dlgGitMenu", MnuGit.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgGitMenu")
}
