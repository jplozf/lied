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
package conf

import (
	"os"
)

const (
	STATUS_MESSAGE_DURATION = 3
	APP_NAME                = "Lied"
	APP_STRING              = "Lied © jpl@ozf.fr 2024"
	APP_URL                 = "https://github.com/jplozf/lied"
	APP_FOLDER              = ".lied"
	ICON_MODIFIED           = "●"
	NEW_FILE_TEMPLATE       = "noname_"
	FILE_LOG                = "lied.log"
	FILE_INI                = "lied.ini"
	FILE_FIND_HISTORY       = "find"
	FILE_SHELL_HISTORY      = "history"
	FILE_SHELL_OUTPUT       = "output"
	FILE_MACROS             = "macros"
	FILE_MRU                = "mru"
	FKEY_LABELS             = "F1=Help F2=Panel F3=Git F4=Shell F6=Previous F7=Next F8=Settings F10=Menu F12=Exit"
	CKEY_LABELS             = "Ctrl+F=Find… Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+O=Open… Ctrl+T=Close Alt+M=Macros"
	DEFAULT_COLOR_ACCENT    = "#556B2F"
)

var Version string

// var Cwd string
var LogFile *os.File

// var Workspace string

type SConfigGeneral struct {
	Theme         string
	GitUser       string
	GitKey        string
	Workspace     string
	ShowHidden    bool
	ConfirmExit   bool
	CleanUpOnExit bool
	FormatTime    string
	FormatDate    string
	ColorAccent   string
}

type SConfigPrivate struct {
	UISleepUpdate        int
	UIGITUpdate          int
	UIStatusTimeout      int
	ClearNullFilesOnExit bool
}

var ConfigGeneral SConfigGeneral
