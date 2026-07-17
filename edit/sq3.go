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
package edit

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"lied/conf"
	"lied/dialog"
	"lied/menu"
	"lied/ui"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
)

// ****************************************************************************
// sq3 is the SQLite3 interface
// ****************************************************************************

var (
	DlgOpenDB  *dialog.Dialog
	DlgCloseDB *dialog.Dialog
	root       *tview.TreeNode
	MnuSQL     *menu.Menu
	ASql       []string
	ISql       int
)
var headerBackgroundColor = tcell.ColorDarkGreen
var headerTextColor = tcell.ColorYellow

// ****************************************************************************
// Xeq()
// ****************************************************************************
func XeqSQL(c string) error {
	c = strings.TrimSpace(c)
	if len(ASql) > 0 {
		if ASql[len(ASql)-1] != c {
			ASql = append(ASql, c)
			ISql++
		}
	} else {
		ASql = append(ASql, c)
		ISql++
	}

	if len(c) > 0 {
		if c[0] == '.' {
			if strings.HasPrefix(strings.ToUpper(c), ".TABLE") {
				return DoSelect("SELECT name FROM sqlite_master WHERE type ='table' AND name NOT LIKE 'sqlite_%';")
			} else {
				if strings.HasPrefix(strings.ToUpper(c), ".DATABASE") {
					return DoSelect("PRAGMA database_list;")
				} else {
					if strings.HasPrefix(strings.ToUpper(c), ".SCHEMA") {
						tokens := strings.Fields(c)
						if len(tokens) > 1 {
							table := tokens[1]
							return DoSelect(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE name = '%s';", table))
						} else {
							ui.SetStatus(tview.Escape("Too few arguments for .SCHEMA [table]"))
							return errors.New("Too few arguments for .SCHEMA [table]")
						}
					} else {
						if strings.HasPrefix(strings.ToUpper(c), ".COLUMNS") {
							tokens := strings.Fields(c)
							if len(tokens) > 1 {
								table := tokens[1]
								return DoSelect(fmt.Sprintf("PRAGMA table_info(%s);", table))
							} else {
								ui.SetStatus(tview.Escape("Too few arguments for .COLUMNS [table]"))
								return errors.New("Too few arguments for .COLUMNS [table]")
							}
						} else {
							if strings.HasPrefix(strings.ToUpper(c), ".OPEN") {
								tokens := strings.Fields(c)
								if len(tokens) > 1 {
									db := tokens[1]
									return OpenDB(db)
								} else {
									ui.SetStatus(tview.Escape("Too few arguments for .OPEN [database]"))
									return errors.New("Too few arguments for .OPEN [database]")
								}
							} else {
								if strings.HasPrefix(strings.ToUpper(c), ".CLOSE") {
									if CurrentView.Database != nil {
										CloseCurrentFile()
										return nil
									} else {
										ui.SetStatus("No open database")
										return errors.New("No open database")
									}
								} else {
									ui.SetStatus(fmt.Sprintf("Unknown command %s", c))
									return errors.New(fmt.Sprintf("Unknown command %s", c))
								}
							}
						}
					}
				}
			}
		} else {
			ui.SetStatus(fmt.Sprintf("Executing %s", c))
			if strings.HasPrefix(strings.ToUpper(c), "SELECT") {
				return DoSelect(c)
			} else {
				if CurrentView.Database != nil {
					return DoExec(c)
				} else {
					ui.SetStatus("No open database")
					return errors.New("No open database")
				}
			}
		}
	}
	return nil
}

// ****************************************************************************
// DoExec()
// ****************************************************************************
func DoExec(cmd string) error {
	_, err := CurrentView.Database.Exec(cmd)
	if err != nil {
		ui.SetStatus(err.Error())
		return err
	}
	ui.SetStatus(fmt.Sprintf("Executing %s", cmd))
	return nil
}

// ****************************************************************************
// OpenDB()
// ****************************************************************************
func OpenDB(fName string) error {
	db, err := sql.Open("sqlite3", fName)
	if err == nil {
		CurrentView.Database = db
		CurrentView.FName = fName
		// dummy instruction to create the header
		_, err = db.Exec("PRAGMA user_version = 0;")
		ui.SetStatus(fmt.Sprintf("Database %s open successfully", fName))
	}
	return err
}

// ****************************************************************************
// CloseDB()
// ****************************************************************************
func CloseDB(db *sql.DB) error {
	err := db.Close()
	ui.SetStatus("Database closed")
	// ui.TxtSQLName.SetText("")
	ui.TblSQLOutput.Clear()
	// ui.TblSQLTables.Clear()
	// ui.TrvSQLDatabase.GetRoot().ClearChildren()
	root = tview.NewTreeNode("")
	// ui.TrvSQLDatabase.SetRoot(root).SetCurrentNode(root)
	CurrentView.Database = nil
	CurrentView.FName = ""
	return err
}

// ****************************************************************************
// DoCloseDB()
// ****************************************************************************
func DoCloseDB() {
	DlgCloseDB = DlgCloseDB.YesNoCancel(fmt.Sprintf("Close Database %s", CurrentView.FName), // Title
		"This file has been modified. Do you want to save it ?", // Message
		confirmCloseDB,
		0,
		ui.GetCurrentScreen(), ui.TxtPrompt) // Focus return
	ui.PgsApp.AddPage("dlgCloseDB", DlgCloseDB.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgCloseDB")
}

// ****************************************************************************
// confirmCloseDB()
// ****************************************************************************
func confirmCloseDB(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_YES {
		CloseDB(CurrentView.Database)
	}
}

// ****************************************************************************
// DoOpenDB()
// ****************************************************************************
func DoOpenDB(path string) {
	DlgOpenDB = DlgOpenDB.Input("Open Database", // Title
		"Please, enter the name for the database to open :", // Message
		path,
		confirmOpenDB,
		0,
		ui.GetCurrentScreen(), ui.TxtPrompt) // Focus return
	ui.PgsApp.AddPage("dlgOpenDB", DlgOpenDB.Popup(), true, false)
	ui.PgsApp.ShowPage("dlgOpenDB")
}

// ****************************************************************************
// confirmOpenDB()
// ****************************************************************************
func confirmOpenDB(rc dialog.DlgButton, idx int) {
	if rc == dialog.BUTTON_OK {
		newDB := DlgOpenDB.Value
		err := OpenDB(newDB)
		if err != nil {
			ui.SetStatus(err.Error())
		} else {
			ui.SetStatus(fmt.Sprintf("Database %s successfully open", newDB))
		}
	}
}

// ****************************************************************************
// showTreeDB()
// ****************************************************************************
func showTreeDB() {
	root = tview.NewTreeNode(filepath.Base(CurrentView.FName))
	ui.TrvExplorer.SetRoot(root).SetCurrentNode(root)
	nodeTables := tview.NewTreeNode("Tables")
	tables := getTables()
	for _, t := range tables {
		nodeTables.AddChild(tview.NewTreeNode(t))
	}
	root.AddChild(nodeTables)
	nodeViews := tview.NewTreeNode("Views")
	views := getViews()
	for _, t := range views {
		nodeViews.AddChild(tview.NewTreeNode(t))
	}
	root.AddChild(nodeViews)
	nodeIndexes := tview.NewTreeNode("Indexes")
	indexes := getIndexes()
	for _, t := range indexes {
		nodeIndexes.AddChild(tview.NewTreeNode(t))
	}
	root.AddChild(nodeIndexes)
	nodeTriggers := tview.NewTreeNode("Triggers")
	triggers := getTriggers()
	for _, t := range triggers {
		nodeTriggers.AddChild(tview.NewTreeNode(t))
	}
	root.AddChild(nodeTriggers)
}

// ****************************************************************************
// getTables()
// ****************************************************************************
func getTables() []string {
	var tables []string
	rows, err := CurrentView.Database.Query("SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';")
	if err == nil {
		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			if err == nil {
				tables = append(tables, name)
			}
		}
	}
	return tables
}

// ****************************************************************************
// getViews()
// ****************************************************************************
func getViews() []string {
	var views []string
	rows, err := CurrentView.Database.Query("SELECT name FROM sqlite_schema WHERE type = 'view';")
	if err == nil {
		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			if err == nil {
				views = append(views, name)
			}
		}
	}
	return views
}

// ****************************************************************************
// getIndexes()
// ****************************************************************************
func getIndexes() []string {
	var indexes []string
	rows, err := CurrentView.Database.Query("SELECT name FROM sqlite_master WHERE type = 'index';")
	if err == nil {
		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			if err == nil {
				indexes = append(indexes, name)
			}
		}
	}
	return indexes
}

// ****************************************************************************
// getTriggers()
// ****************************************************************************
func getTriggers() []string {
	var triggers []string
	rows, err := CurrentView.Database.Query("select name from sqlite_master where type = 'trigger';")
	if err == nil {
		for rows.Next() {
			var name string
			err = rows.Scan(&name)
			if err == nil {
				triggers = append(triggers, name)
			}
		}
	}
	return triggers
}

// ****************************************************************************
// DoSelect()
// ****************************************************************************
func DoSelect(q string) error {
	if CurrentView.Database != nil {
		ui.TblSQLOutput.Clear()
		var myMap = make(map[string]interface{})
		rows, err := CurrentView.Database.Query(q)
		if err != nil {
			ui.SetStatus(err.Error())
			return err
		} else {
			defer rows.Close()
			colNames, err := rows.Columns()
			if err != nil {
				ui.SetStatus(err.Error())
				return err
			} else {
				cols := make([]interface{}, len(colNames))
				colPtrs := make([]interface{}, len(colNames))
				for i := 0; i < len(colNames); i++ {
					colPtrs[i] = &cols[i]
				}
				// Header of fields names
				for k, colName := range colNames {
					ui.TblSQLOutput.SetCell(0, k, tview.NewTableCell(tview.Escape("["+colName+"]")).SetAlign(tview.AlignLeft).SetTextColor(headerTextColor).SetBackgroundColor(tcell.GetColor(conf.DEFAULT_COLOR_ACCENT)))
				}
				i := 1
				for rows.Next() {
					err = rows.Scan(colPtrs...)
					if err != nil {
						ui.SetStatus(err.Error())
						return err
					} else {
						for k, col := range cols {
							myMap[colNames[k]] = col
						}
						j := 0
						for k := range cols {
							field := colNames[k]
							value := myMap[field]
							if reflect.TypeOf(value) != nil {
								typeVal := reflect.TypeOf(value).String()
								if typeVal == "string" {
									ui.TblSQLOutput.SetCell(i, j, tview.NewTableCell(fmt.Sprintf("%s", value)))
								}
								if strings.HasPrefix(typeVal, "int") {
									ui.TblSQLOutput.SetCell(i, j, tview.NewTableCell(fmt.Sprintf("%d", value)).SetAlign(tview.AlignRight))
								}
								if strings.HasPrefix(typeVal, "float") {
									ui.TblSQLOutput.SetCell(i, j, tview.NewTableCell(fmt.Sprintf("%f", value)).SetAlign(tview.AlignRight))
								}
								if strings.HasPrefix(typeVal, "bool") {
									ui.TblSQLOutput.SetCell(i, j, tview.NewTableCell(fmt.Sprintf("%t", value)).SetAlign(tview.AlignCenter))
								}
							} else {
								ui.TblSQLOutput.SetCell(i, j, tview.NewTableCell("(NULL)").SetAlign(tview.AlignCenter))
							}
							j++
						}
					}
					i++
				}
				ui.TblSQLOutput.SetFixed(1, 0)
				ui.TblSQLOutput.Select(1, 0)
				ui.App.SetFocus(ui.TxtPromptSQL)
				return nil
			}
		}
	} else {
		ui.SetStatus("No open database")
		return errors.New("No open database")

	}
}

// ****************************************************************************
// DoExportRow(p any)
// ****************************************************************************
func DoExportRow(p any) {
	r, _ := ui.TblSQLOutput.GetSelection()
	if r > 0 {
		f, err := os.CreateTemp(conf.APP_FOLDER, conf.NEW_FILE_TEMPLATE)
		if err == nil {
			defer f.Close()
			w := bufio.NewWriter(f)
			out := ""
			for c := 0; c < ui.TblSQLOutput.GetColumnCount(); c++ {
				out = out + fmt.Sprintf("\"%s\",", ui.TblSQLOutput.GetCell(r, c).Text)
			}
			_, err = fmt.Fprintf(w, "%s", out[:len(out)-1])
			if err != nil {
				ui.SetStatus(err.Error())
			} else {
				w.Flush()
				SwitchToEditor(f.Name())
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
}

// ****************************************************************************
// DoExportAll(p any)
// ****************************************************************************
func DoExportAll(p any) {
	f, err := os.CreateTemp(conf.APP_FOLDER, conf.NEW_FILE_TEMPLATE)
	if err == nil {
		defer f.Close()
		w := bufio.NewWriter(f)
		for r := 0; r < ui.TblSQLOutput.GetRowCount(); r++ {
			out := ""
			for c := 0; c < ui.TblSQLOutput.GetColumnCount(); c++ {
				fldName := ui.TblSQLOutput.GetCell(r, c).Text
				if r == 0 {
					// Special case for escaping '[]' in field's name in first row
					fldName = fldName[:len(fldName)-2] + fldName[len(fldName)-1:]
				}
				out = out + fmt.Sprintf("\"%s\",", fldName)
			}
			_, err = fmt.Fprintf(w, "%s\n", out[:len(out)-1])
			if err != nil {
				ui.SetStatus(err.Error())
			}
		}
		w.Flush()
		SwitchToEditor(f.Name())
	} else {
		ui.SetStatus(err.Error())
	}
}

// ****************************************************************************
// DoExportAllJSON(p any)
// ****************************************************************************
func DoExportAllJSON(p any) {
	if ui.TblSQLOutput.GetRowCount() <= 1 {
		ui.SetStatus("No SQL result to export")
		return
	}

	f, err := os.CreateTemp(conf.APP_FOLDER, conf.NEW_FILE_TEMPLATE)
	if err != nil {
		ui.SetStatus(err.Error())
		return
	}
	defer f.Close()

	cols := ui.TblSQLOutput.GetColumnCount()
	rows := make([]map[string]string, 0, ui.TblSQLOutput.GetRowCount()-1)
	headers := make([]string, cols)

	for c := 0; c < cols; c++ {
		head := ui.TblSQLOutput.GetCell(0, c).Text
		head = strings.TrimPrefix(head, "[")
		head = strings.TrimSuffix(head, "]")
		headers[c] = head
	}

	for r := 1; r < ui.TblSQLOutput.GetRowCount(); r++ {
		row := make(map[string]string, cols)
		for c := 0; c < cols; c++ {
			row[headers[c]] = ui.TblSQLOutput.GetCell(r, c).Text
		}
		rows = append(rows, row)
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rows); err != nil {
		ui.SetStatus(err.Error())
		return
	}

	SwitchToEditor(f.Name())
}

// ****************************************************************************
// DoCopyCell(p any)
// ****************************************************************************
func DoCopyCell(p any) {
	r, c := ui.TblSQLOutput.GetSelection()
	if r <= 0 {
		ui.SetStatus("No SQL data cell selected")
		return
	}

	value := ui.TblSQLOutput.GetCell(r, c).Text
	if err := clipboard.WriteAll(value); err != nil {
		ui.SetStatus(err.Error())
		return
	}
	ui.SetStatus("Cell copied to system clipboard")
}

// ****************************************************************************
// DoExportCell(p any)
// ****************************************************************************
func DoExportCell(p any) {
	r, c := ui.TblSQLOutput.GetSelection()
	if r > 0 {
		f, err := os.CreateTemp(conf.APP_FOLDER, conf.NEW_FILE_TEMPLATE)
		if err == nil {
			defer f.Close()
			w := bufio.NewWriter(f)
			_, err = fmt.Fprintf(w, "%s", ui.TblSQLOutput.GetCell(r, c).Text)
			if err != nil {
				ui.SetStatus(err.Error())
			} else {
				w.Flush()
				SwitchToEditor(f.Name())
			}
		} else {
			ui.SetStatus(err.Error())
		}
	}
}

// ****************************************************************************
// SwitchToSQLite3()
// ****************************************************************************
func SwitchToSQLite3() {
	// ui.AddNewScreen(ui.ModeSQLite3, nil, nil)
	ui.PgsApp.SwitchToPage("edit")
	ui.PgsEditorContent.SwitchToPage("sqlViewer")
	ui.App.SetFocus(ui.TxtPromptSQL)
	ui.SetStatus("Switching to [SQLite3]")
}

// ****************************************************************************
// SelfInit()
// ****************************************************************************
/*
func SelfInit(a any) {
		if ui.CurrentMode == ui.ModeFiles {
			idx, _ := ui.TblFiles.GetSelection()
			fName := filepath.Join(conf.Cwd, strings.TrimSpace(ui.TblFiles.GetCell(idx, 2).Text))
			xtype, _ := mimetype.DetectFile(fName)
			if strings.HasSuffix(xtype.String(), "sqlite3") {
				// Is there an open database ?
				if CurrentDB == nil {
					// no, then open the targeted database
					err := OpenDB(fName)
					if err == nil {
						ui.AddNewScreen(ui.ModeSQLite3, nil, nil)
						ui.App.SetFocus(ui.TxtPrompt)
						ui.SetStatus(fmt.Sprintf("Switching to [SQLite3]"))
					} else {
						ui.CurrentMode = ui.ModeSQLite3
						ui.SetTitle("SQLite3")
						ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save")
						ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
						ui.App.SetFocus(ui.TxtPrompt)
						ui.SetStatus(err.Error())
					}
				} else {
					// attach the targeted database to the current database
					DoExec(fmt.Sprintf("attach database '%s' as %s", fName, utils.FilenameWithoutExtension(filepath.Base(fName))))
					ui.CurrentMode = ui.ModeSQLite3
					ui.SetTitle("SQLite3")
					ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save")
					ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
					ui.App.SetFocus(ui.TxtPrompt)
				}
			} else {
				ui.CurrentMode = ui.ModeSQLite3
				ui.SetTitle("SQLite3")
				ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save")
				ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
				ui.App.SetFocus(ui.TxtPrompt)
			}
		} else {
			ui.CurrentMode = ui.ModeSQLite3
			ui.SetTitle("SQLite3")
			ui.LblKeys.SetText(conf.FKEY_LABELS + "\nCtrl+O=Open Ctrl+S=Save")
			ui.PgsApp.SwitchToPage(ui.GetCurrentScreen())
			ui.App.SetFocus(ui.TxtPrompt)
		}
}
*/

// ****************************************************************************
// CreateTempDatabase()
// ****************************************************************************
func CreateTempDatabase(dir string) {
	dbName := filepath.Join(dir, conf.NEW_DATABASE_TEMPLATE+strconv.Itoa(rand.Int()))
	db, err := sql.Open("sqlite3", dbName)
	if err == nil {
		ui.SetStatus(err.Error())
	} else {
		defer db.Close()

	}
}

// ****************************************************************************
// ExportToJSONStreaming()
// ****************************************************************************
func ExportToJSONStreaming(rows *sql.Rows, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	columns, _ := rows.Columns()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	file.WriteString("[\n") // Start array

	first := true
	for rows.Next() {
		if !first {
			file.WriteString(",\n") // Separator
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return err
		}

		entry := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// NULL handling: Go's json encoder converts nil to 'null' automatically
			entry[col] = val
		}

		encoder.Encode(entry)
		first = false
	}

	file.WriteString("\n]") // End array
	return nil
}

// ****************************************************************************
// ExportToCSV()
// ****************************************************************************
func ExportToCSV(rows *sql.Rows, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	columns, _ := rows.Columns()
	writer.Write(columns)

	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	for rows.Next() {
		rows.Scan(valuePtrs...)
		record := make([]string, len(columns))

		for i, val := range values {
			if val == nil {
				record[i] = "" // Handle NULL as empty string
			} else {
				// Handle byte slices (common in SQLite for strings/blobs)
				if b, ok := val.([]byte); ok {
					record[i] = string(b)
				} else {
					record[i] = fmt.Sprintf("%v", val)
				}
			}
		}
		writer.Write(record)
	}
	return nil
}
