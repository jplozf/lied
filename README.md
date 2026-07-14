# ♯LIED

## Summary
* **Lied** is a TUI (Text User Interface) enhanced editor.
* **Lied** is written in `Go` and has been tested on **Linux** sytem.
* Built from source, it should run on **Windows** or **MacOS** systems as well.
* **Lied** is capable of displaying text with syntax highlighting; however, beyond source code, it can also open binary files in hexadecimal view, as well as `SQLite3` databases, allowing you to run queries and view their contents.
* For a lighter version of a TUI editor, please consider to use my other text editor [**Bled**](https://github.com/jplozf/bled).

## Credits
*Pay honour to whom honour is due*, packages used in this project are as follows :
* [`rivo/tview`](https://github.com/rivo/tview) : Package tview implements rich widgets for terminal based user interfaces.
* [`gdamore/tcell`](https://github.com/gdamore/tcell) : Tcell is an alternate terminal package, similar in some ways to termbox, but better in others. 
* [`pgavlin/femto`](https://github.com/pgavlin/femto) : An editor component for tview. Derived from the micro editor. 
* [`atotto/clipboard`](https://github.com/atotto/clipboard) : Clipboard management.

## Features
⯈ The main functions are reachable through function keys :
| Function Key | Purpose |
| :--- | :--- |
| `F1`  | This help text |
| `F2`  | Switch from the current panel to the next |
| `F3`  | Access to Git Menu for running Git common commands |
| `F4`  | Access to shell to run system and special commands |
| `F6`  | Switch to the previous open file |
| `F7`  | Switch to the next open file |
| `F8`  | Access to the settings of Lied |
| `F9`  | Access to the workspace menu |
| `F10` | Access to the main menu of Lied |
| `F12` | Exit |

⯈ Alternate common functions are also reachable through `CTRL` and `ALT` keys :
| Keystroke | Purpose |
| :--- | :--- |
| `CTRL + F` | Switch to the Find & Replace panel, or go back to the Editor panel |
| `CTRL + S` | Saves the current document being edited |
| `ALT  + S` | Saves the current document being edited under another name |
| `CTRL + N` | Opens a new blank document |
| `CTRL + O` | Opens an existing document for editing |
| `CTRL + T` | Closes the current document |
| `ALT  + M` | Opens the macros menu |
| `CTRL + Q` | Quits the editor (same as `F12`) |
| `ALT  + Q` | Temporary escapes to the Shell |

⯈ When editing a text, common editing functions are of course supported :
| Keystroke | Purpose |
| :--- | :--- |
| `CTRL + C` | Copy the selection |
| `CTRL + X` | Cut the selection |
| `CTRL + V` | Paste the selection |
| `CTRL + Z` | Cancels the previous entry  |
| `CTRL + Y` | Redo the previous cancelled operation |

All these commands are summarized in the two lines at the bottom of the screen.

Let's take a look at all these features...

### Help `F1`
---
The embedded Help is opened by pressing the `F1` key. 
This help can also be invoked by running the special command `!help` in the Shell input box.

### Moving between Panels `F2`
---
The `F2` key allow you to move the focus from the current panel to the next one.
The current panel is recognizable by its double border while the other panels only have a single border.
Starting from the default current panel which is the Editor panel, you can move to the panel and so on, and returning to the first panel.

These panels are in order :
* The **Editor** panel
* The **Open Files** panel
* The **Find & Replace** panel
* The **Explorer** panel
* The **Outline** panel

These panels will be detailed later in this document.

### Git Menu `F3`
---
Some useful common `git` commands are accessible through the options of a dedicated menu.
This menu is displayed when pressing the `F3` key.

These options are as follows :
| Option | Purpose |
| :--- | :--- |
| `Status`               | Displays the status of the current branch. |
| `Log`                  | Displays the log for the current repository. |
| `Add All (.)`          | Add all files recursively to the Git tracking. |
| `Commit`               | Commits the current changes after prompting for the commit message. |
| `Push`                 | Pushes the current commit to the remote repository. |
| `Commit & Push`        | Commits the current changes and the push it to remote repository. |
| `Fetch`                | This only copies changes from remote repository into your local Git repository. |
| `Pull (Fetch & Merge)` | Copies changes from remote repository to you local repository and merge them with your working copy. |
| `Initialize`           | This will initialize a local repository for a new project. |
| `Initialize & Push`    | This will initialize a local repository for a new project. The remote repository should already exists. |
| `Clone`                | Clone a remote repository into the current local folder. No local repository should exists. |
| `Configure`            | This should normally only be done once, as it saves the Github username and associated password that will be used for Git commands. |

### Shell Input Box `F4`
---
### Settings Menu `F8`
---
### Workspace Menu `F9`
---
### Main menu `F10`
---
### Panels and UI
---
We already saw that the main panels are in order :
* The **Editor** panel :

This is the main panel where you view the **text file** currently being edited. The source code language is automatically detected based on the file extension, and the syntax highlighting is adjusted accordingly. Relevant information regarding the file type—such as encoding, detected language, and line-ending format—is displayed in the top-right corner of this panel.

For a **binary file**, this panel switches to hexadecimal view; the display then features the standard three-column layout, with addresses starting at 0 on the left, hexadecimal values ​​in the center, and the ASCII representation on the right. Currently, the hexadecimal view is read-only , no modifications are allowed.

When opening an **SQLite3** database file, this panel splits into two sections : an input field allows you to enter `SQL` code, which is then executed by pressing the `F5` key or the `Alt + Enter` key combination. The query result is then displayed above, in the second part of the panel.

* The **Open Files** panel :

This panel displays a list of the various files currently open for editing. For each file, you can see its name followed by the full file path. A bullet point `●` appears at the start of the line if the file has been modified but not yet saved. You can switch between files by selecting the corresponding line in the list, just as you would navigate to the previous open file using `F6` or the next one using `F7`.

* The **Find & Replace** panel


* The **Explorer** panel


* The **Outline** panel
  
We can also add some interesting spots like (from up to down) :
| UI Item | Purpose |
| :--- | :--- |
| The Title Bar | It shows the name of this program but also its version. Current time and date are also displayed in corners. |
| The Workspace and File text boxes | Just to be sure to edit the good file. |
| The Keys' Summary Area | Keys functions are summarized as a reminder. |
| The Git Status Bar | Useful when editing a tracked file, it displays the branch name, the current commit hash and the pending status. |
| The Status Bar | The classical status bar displays the size of the current file, the relative position of the cursor in percentage, the position of the cursor in terms of line and column numbers, and other various indicators. These informations may change depending of the currently edited file's type (Source Code, Binary, Database...). |

### Macros `Alt + M`
---
Useful common functions can be added to the Macros menu accessible with the `ALT + M` keystroke.
This menu shows all the macros you previously recorded and the last option allows you to edit and add new macros.

The syntax for the macros file is very simple :
* One line, one macro.
* Each line beginning with a `#` char is ignored.
* The name of the macro which will be displayed on the menu is the first part of the line **BEFORE** the `:` char.
* The command of the macro which will be runned is the second part of the line **AFTER** the `:` char.

Macros can interact with dynamic values by using special placeholders which will be replaced by the appropriate value at run time.

Theses placeholders, prefixed by a `%` char and case sensitive, are as follows :

| Placeholder | Purpose |
| :--- | :--- |
| `%D` | Full directory of current file |
| `%P` | Parent directory of current file |
| `%W` | Full directory of current workspace |
| `%w` | Base name of current workspace |
| `%F` | Full file name with directory and extension of current file |
| `%f` | File name without path and with extension of current file |
| `%e` | File name without path nor extension of current file |
| `%L` | Line number of current file in editor |
| `%T` | Current timestamp |
| `%H` | Home directory of current user |
| `%s` | OS path separator |
| `%U` | Current user name |
| `%GU` | GitHub user from config file |
| `%GK` | GitHub key from config file |
| `%GE` | GitHub email from config file |
| `%ON` | Organization name from config file |
| `%OE` | Organization extension from config file |

As we already said, macros syntax is simple, each line in the macros file represents a macro, with the following format :

`<Macro Name> : <Command to execute>`

For example, a macro to open the current file in the default system viewer could be defined as follows :

`Open in Explorer : xdg-open %D`

This macro will use the `xdg-open` command to open the current file's directory (represented by the `%D` placeholder) in the default system explorer when executed.

If the command to execute starts with `insert:`, the rest of the command will be inserted into the current document at the current cursor position instead of being executed as a system command.

For example, a macro to insert the current date and time into the document could be defined as follows :

`Insert Timestamp : insert:%T`

This macro will insert the current timestamp (represented by the `%T` placeholder) into the document at the current cursor position when executed.

The position of the cursor after the insertion can be controlled using the special `$0` marker in the inserted text. 

For example, the following macro will insert a main function template in Go and place the cursor at the beginning of the function body :

`Insert Go main : insert:func main() {\n\t$0\n}`

You can create as many macros as you want, and they will be available in the **Macros** menu `ALT + M` for easy access.

### Templates
---
* Templates for new files are located into the `~/.lied/templates` directory. 
* By default, this folder is populated by embedded templates.
* You can add or remove templates as you wish. 
* These templates can accept the same placeholders as macros.
* To create a file from a template, open the **Workspace** menu by pressing the `F9` key, and then select **New file**.
  
### Specials Commands
---
These special commands are available when entering command in the Shell input box by pressing the `F4` key.
These special commands all have a maximum of 4 letters and are prefixed by an exclamation point `!`.

| Command | Purpose |
| :--- | :--- |
| `!quit` | Quit the editor (same as `F12` or `CTRL + Q`) |
| `!exit` | *idem* |
| `!bye`  | *idem* |
| `!log`  | Opens the log file of the Lied editor |
| `!out`  | Opens the output file for the Shell input box |
| `!foll` | Switch the current file in following mode |
| `!tail` | *idem* |
| `!shel` | Enable the interactive shell if not enabled by default in settings |
| `!next` | Switch to the next open file (same as `F7`) |
| `!prev` | Switch to the previous open file (same as `F6`) |
| `!clos` | Closes the current document (same as `CTRL + T`) |
| `!save` | Saves the current document being edited (same as `CTRL + S`) |
| `!conf` | Opens the Lied configuration file for editing |
| `!macr` | Opens the macros file for editing |
| `!help` | Displays the Help text (same as `F1`) |
| `!info` | Displays system informations about the computer |
| `!b`    | Go to the bottom of the current file |
| `!bott` | *idem* |
| `!t`    | Go to the top of the current file |
| `!top`  | *idem* |
| `!h`    | Insert a timestamp string at the current cursor location |
| `!time` | *idem* |
| `!uuid` | Insert a random `UUID` at the current cursor location |
| `!lore` | Insert a random `Lorem Ipsum` text at the current cursor location |
| `!42`   | Go to that line number (**Breaking News** : *it works also with other line numbers !*) |

### Internal Files
---
Some files are generated and managed internally to keep the current settings and history.
These files are located into the `.lied` directory, which is itself into the user folder : `~/.lied` for **Linux** and **MacOS**. They look like something like this :

```bash
.
├── find
├── history
├── lied_202604.zip
├── lied_202605.zip
├── lied_202606.zip
├── lied.ini
├── lied.log
├── lied.log.bak
├── macros
├── mru
├── output
├── sql
└── templates
    ├── Assembly Source Code 32 bits.asm
    ├── Assembly Source Code 64 bits.asm
    ├── Bash Script
    ├── C C++ Header.h
    ├── C GTK Source Code.c
    ├── C Ncurses Source Code.c
    ├── C Source Code.c
    ├── C++ Source Code.cpp
    ├── C++ wxWidgets Source Code.cpp
    ├── _Empty File
    ├── Go Source Code.go
    ├── HTML Page.html
    ├── Java Applet.java
    ├── Java Source Code.java
    ├── Java Swing Application.java
    ├── Makefile
    ├── Markdown File.md
    ├── Python Script.py
    ├── Python TkInter.py
    ├── QT MainWindow.ui
    ├── Rust Source Code.rs
    ├── Text File.txt
    └── XML File.xml
```
Where :
| File | Purpose |
| :--- | :--- |
| `find` | Find & replace history |
| `history` | Commands history |
| `lied.ini` | Settings *(updated by the UI or directly edited by `!conf` command)* |
| `lied.log` | Main log file |
| `lied.log.bak` | Backup log file *(used for archiving)* |
| `lied_YYYYMM.zip` | Archived log file by month and year |
| `macros` | Macros file *(edited from menu **Macros**)* |
| `mru` | Most Recent Used files |
| `output` | Commands output |
| `sql` | SQL history |
| `templates` | Folder for templates *(filled by default embedded templates, feel free to add your own templates or delete unwanted items)* |

## License
This project is licensed under the terms of the [MIT license](https://www.mit.edu/~amini/LICENSE.md).
