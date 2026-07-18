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
	"bytes"
	"fmt"
	"lied/conf"
	"lied/edit"
	"lied/ui"
	"lied/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-cmd/cmd"
	"github.com/google/uuid"
)

// ****************************************************************************
// Xeq()
// ****************************************************************************
func Xeq(c string) {
	sCmd := strings.Fields(c)
	if len(ACmd) > 0 {
		if ACmd[len(ACmd)-1] != c {
			ACmd = append(ACmd, c)
			ICmd++
		}
	} else {
		ACmd = append(ACmd, c)
		ICmd++
	}

	if len(sCmd) > 0 {
		ui.SetStatus(fmt.Sprintf("Running [%s]", c))
		if sCmd[0][0] == '!' {
			xCmd := sCmd[0] + "     "
			// Is it a line number ?
			if l, err := strconv.Atoi(strings.TrimSpace(xCmd[1:])); err == nil {
				// Yes, go to that line number
				edit.GoLine(l)
				return
			}
			// No, continue...
			xCmd = xCmd[:5]
			xCmd = strings.TrimSpace(xCmd)
			if p, ok := ui.GetPluginByInternalCommand(xCmd); ok {
				if p.CommandOpensPluginView() {
					edit.OpenPluginView(p)
				} else {
					if err := p.ExecuteInternalCommand(); err != nil {
						ui.SetStatus(ui.FormatInternalCommandError(p, err))
					}
				}
				return
			}
			switch xCmd {
			case "!quit", "!exit", "!bye":
				ui.PgsApp.SwitchToPage("dlgQuit")
			case "!log":
				edit.OpenView(filepath.Join(appDir, conf.FILE_LOG), false)
			case "!out":
				edit.OpenView(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), false)
				edit.SwitchFollow("dummy")
			case "!foll", "!tail":
				edit.SwitchFollow("dummy")
			case "!next":
				edit.SwitchNextFile()
			case "!prev":
				edit.SwitchPreviousFile()
			case "!clos":
				edit.CloseCurrentFile()
			case "!save":
				edit.SaveFile()
			case "!conf":
				edit.OpenView(filepath.Join(appDir, conf.FILE_INI), false)
			case "!macr":
				edit.OpenView(filepath.Join(appDir, conf.FILE_MACROS), false)
			case "!help":
				ShowManual()
			case "!info":
				ShowSysInfo()
			case "!shel":
				doInteractiveShell()
			case "!b", "!bott":
				edit.GoBottom()
			case "!t", "!top":
				edit.GoTop()
			case "!h", "!time":
				t := time.Now()
				edit.InsertString(t.Format("20060102-150405"))
			case "!uuid":
				id := uuid.New()
				edit.InsertString(id.String())
			case "!lore":
				edit.InsertString(utils.GenerateLoremIpsum(1, 3, 5, 8, 15))
			default:
				ui.SetStatus(fmt.Sprintf("Invalid command %s", sCmd[0]))
			}
		} else {
			// 1. Setup the command
			cmdOptions := cmd.Options{
				Buffered:  false, // We want streaming
				Streaming: true,
			}
			xCmd := cmd.NewCmdOptions(cmdOptions, sCmd[0], sCmd[1:]...)
			activeCmd = xCmd // Assign to the shared variable
			xCmd.Dir = shellWorkingDirectory()

			// 2. Open the log file once (use O_APPEND)
			fOut, err := os.OpenFile(filepath.Join(appDir, conf.FILE_SHELL_OUTPUT), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				ui.SetStatus("Error opening shell output file: " + err.Error())
				return
			}

			// 3. Start the command in the background
			statusChan := xCmd.Start()
			// Check if the command failed to even start (e.g., command not found)
			initialStatus := xCmd.Status()
			if initialStatus.Error != nil {
				ui.SetStatus("Error: " + initialStatus.Error.Error())
				// Log it to the file as well
				fmt.Fprintln(fOut, "START ERROR: "+initialStatus.Error.Error())
				return
			}

			// 4. Handle Output & Lifecycle in a single Goroutine

			go func() {
				defer fOut.Close()

				// Write header to file
				fmt.Fprintf(fOut, "%s ⯈ %s\n", time.Now().Format("20060102-150405"), c)
				outPrefix := ""
				errPrefix := ""
				commandDone := false
				exitCode := 0
				if conf.ConfigGeneral.OutErrPrefix {
					outPrefix = "OUT : "
					errPrefix = "ERR : "
				}
				for {
					select {
					case line, open := <-xCmd.Stdout:
						if !open {
							xCmd.Stdout = nil
							if commandDone && xCmd.Stderr == nil {
								fmt.Fprintln(fOut, fmt.Sprintf("%s ⯈ Done [%s] Exit Code: %d\n", time.Now().Format("20060102-150405"), c, exitCode))
								ui.App.QueueUpdateDraw(func() {
									ui.SetStatus(fmt.Sprintf("Done [%s] Exit Code: %d", c, exitCode))
								})
								return
							}
						} else {
							fmt.Fprintln(fOut, outPrefix+line)
						}
					case line, open := <-xCmd.Stderr:
						if !open {
							xCmd.Stderr = nil
							if commandDone && xCmd.Stdout == nil {
								fmt.Fprintln(fOut, fmt.Sprintf("%s ⯈ Done [%s] Exit Code: %d\n", time.Now().Format("20060102-150405"), c, exitCode))
								ui.App.QueueUpdateDraw(func() {
									ui.SetStatus(fmt.Sprintf("Done [%s] Exit Code: %d", c, exitCode))
								})
								return
							}
						} else {
							fmt.Fprintln(fOut, errPrefix+line)
						}
					case status, open := <-statusChan:
						if !open {
							statusChan = nil
							continue
						}
						commandDone = true
						exitCode = status.Exit
						statusChan = nil
						if xCmd.Stdout == nil && xCmd.Stderr == nil {
							fmt.Fprintln(fOut, fmt.Sprintf("%s ⯈ Done [%s] Exit Code: %d\n", time.Now().Format("20060102-150405"), c, exitCode))
							ui.App.QueueUpdateDraw(func() {
								ui.SetStatus(fmt.Sprintf("Done [%s] Exit Code: %d", c, exitCode))
							})
							return
						}
					}

					// If both streams are closed but status hasn't arrived,
					// we still need to wait for statusChan to avoid leaking
					if commandDone && xCmd.Stdout == nil && xCmd.Stderr == nil {
						return
					}
				}
			}()
		}
	} else {
		ui.SetStatus("Nothing to run")
	}
}

// ****************************************************************************
// XeqOut()
// ****************************************************************************
func XeqOut(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
			}
		} else {
			out = outb.String()
			out = out + errb.String()
			out = out + "\nExit code 0"
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}

// ****************************************************************************
// XeqOutErr()
// ****************************************************************************
func XeqOutErr(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
				out = out + outb.String()
				out = out + errb.String()
			}
		} else {
			out = outb.String()
			out = out + errb.String()
			out = out + "\nExit code 0"
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}

// ****************************************************************************
// XeqRaw()
// ****************************************************************************
func XeqRaw(c string) string {
	// sCmd := strings.Fields(c)
	// https://stackoverflow.com/questions/47489745/splitting-a-string-at-space-except-inside-quotation-marks
	quoted := false
	sCmd := strings.FieldsFunc(c, func(r rune) bool {
		if r == '"' {
			quoted = !quoted
		}
		return !quoted && r == ' '
	})

	out := ""
	if len(sCmd) > 0 {
		cmd := exec.Command(sCmd[0], sCmd[1:]...)
		cmd.Dir = conf.ConfigGeneral.Workspace
		ui.SetStatus(fmt.Sprintf("Executing [%s] in %s", c, cmd.Dir))
		var outb, errb bytes.Buffer
		cmd.Stdout = &outb
		cmd.Stderr = &errb
		if err := cmd.Run(); err != nil {
			out = "Error : " + err.Error()
			if exitError, ok := err.(*exec.ExitError); ok {
				out = out + fmt.Sprintf("\nExit code %d", exitError.ExitCode())
			}
		} else {
			out = outb.String()
			out = out + errb.String()
		}
	} else {
		out = "Nothing to run\n\nExit code 0"
	}

	out = strings.TrimSpace(out)
	ui.SetStatus(out)
	ui.SetStatus(fmt.Sprintf("Done [%s]", c))
	return out
}
