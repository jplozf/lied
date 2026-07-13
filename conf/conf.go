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
	"embed"
	"os"

	"github.com/gdamore/tcell/v2"
)

//go:embed licenses/*.md
var LicensesFS embed.FS

//go:embed templates/*
var TemplatesFS embed.FS

const (
	STATUS_MESSAGE_DURATION = 3
	APP_NAME                = "Lied"
	APP_STRING              = "Lied © jpl@ozf.fr 2024-2026"
	APP_URL                 = "https://github.com/jplozf/lied"
	APP_FOLDER              = ".lied"
	ICON_MODIFIED           = "●"
	ICON_DATABASE           = "⛁"
	ICON_EXPLORER           = "🗁"
	NEW_FILE_TEMPLATE       = "noname_"
	NEW_DATABASE_TEMPLATE   = "db_"
	FILE_LOG                = "lied.log"
	FILE_INI                = "lied.ini"
	FILE_FIND_HISTORY       = "find"
	FILE_SHELL_HISTORY      = "history"
	FILE_SQL_HISTORY        = "sql"
	FILE_SHELL_OUTPUT       = "output"
	FILE_MACROS             = "macros"
	FILE_MRU                = "mru"
	FKEY_LABELS             = "F1=Help F2=Panel F3=Git F4=Command F6=Previous F7=Next F8=Settings F9=Context F10=Menu F12=Exit"
	CKEY_LABELS             = "Ctrl+E=Explorer Ctrl+F=Find… Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+O=Open… Ctrl+T=Close Alt+M=Macros Alt+Q=Shell"
	DEFAULT_COLOR_ACCENT    = "#556B2F"
	FILE_MAX_PREVIEW        = 1024
	HASH_THRESHOLD_SIZE     = 1_073_741_824.0
	COLOR_FOLDER            = tcell.ColorLightGreen
	COLOR_FILE              = tcell.ColorYellow
	COLOR_EXECUTABLE        = tcell.ColorLightYellow
	COLOR_SELECTED          = tcell.ColorRed
	LABEL_PARENT_FOLDER     = "<UP>"
)

var Version string

var LogFile *os.File

type SConfigGeneral struct {
	Theme            string
	GitUser          string
	GitKey           string
	GitEmail         string
	Workspace        string
	ShowHidden       bool
	ConfirmExit      bool
	CleanUpOnExit    bool
	FormatTime       string
	FormatDate       string
	ColorAccent      string
	InteractiveShell bool
	OutErrPrefix     bool
}

type SConfigPrivate struct {
	UISleepUpdate        int
	UIGITUpdate          int
	UIStatusTimeout      int
	ClearNullFilesOnExit bool
}

var ConfigGeneral SConfigGeneral
