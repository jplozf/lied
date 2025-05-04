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
	APP_VERSION             = "0.1.0"
	APP_URL                 = "https://github.com/jplozf/lied"
	APP_FOLDER              = ".lied"
	ICON_MODIFIED           = "●"
	NEW_FILE_TEMPLATE       = "lied_"
	FILE_LOG                = "lied.log"
	FILE_CONFIG             = "lied.json"
	FILE_INI                = "lied.ini"
	FILE_MRU                = "mru"
	FKEY_LABELS             = "F1=Help F2=Panel F3=GIT F4=Shell F6=Previous F7=Next F8=Settings F10=Menu F12=Exit"
	CKEY_LABELS             = "Ctrl+F=Find… Ctrl+S=Save Alt+S=Save as… Ctrl+N=New Ctrl+O=Open… Ctrl+T=Close"
)

// var Cwd string
var LogFile *os.File

// var Workspace string

type Config struct {
	Theme       string
	GitUser     string
	GitPassword string
	Workspace   string
	ShowHidden  bool
	ConfirmExit bool
	FormatTime  string
	FormatDate  string
}
