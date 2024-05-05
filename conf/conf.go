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
	FKEY_LABELS             = "F1=Help F2=Prompt F3=Close F5=Refresh F6=Previous F7=Next F8=Context Menu F10=Main Menu F12=Exit"
)

var Cwd string
var LogFile *os.File
