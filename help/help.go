// ****************************************************************************
//
//	 _ _          _
//	| (_) ___  __| |
//	| | |/ _ \/ _` |
//	| | |  __/ (_| |
//	|_|_|\___|\__,_|
//
// ****************************************************************************
// L I E D   -   Copyright © JPL 2024-2026
// ****************************************************************************
package help

var HelpText = `♯ L I E D   -   Copyright © JPL 2024-2026

Lied is a TUI (Text User Interface) Editor.
Lied is written in Go and has been tested on Linux sytem.
Built from source, it should run on Windows or MacOS systems as well.

Pay honour to whom honour is due, packages used in this project are as follows :
rivo/tview    : Package tview implements rich widgets for terminal based user interfaces.
gdamore/tcell : Tcell is an alternate terminal package, similar in some ways to termbox, but better in others. 
pgavlin/femto : An editor component for tview. Derived from the micro editor. 

⯈ The main functions are reachable through function keys :

F1  : This help screen
F2  : Switch from the current panel to the next
F3  : Access to Git Menu for running Git common commands
F4  : Access to shell to run system and special commands
F6  : Switch to the previous open file
F7  : Switch to the next open file
F8  : Access to the settings of Lied
F9  : Access to the workspace menu
F10 : Access to the main menu of Lied
F12 : Exit

⯈ Alternate common functions are also reachable through CTRL and ALT keys :

CTRL + F : Switch to the Find & Replace panel, or go back to the Editor panel
CTRL + S : Saves the current document being edited
ALT  + S : Saves the current document being edited under another name
CTRL + N : Opens a new blank document
CTRL + O : Opens an existing document for editing
CTRL + T : Closes the current document
ALT  + M : Opens the macros menu
CTRL + Q : Quits the editor (same as F12)
ALT  + Q : Temporary escapes to the Shell

⯈ When editing a text, common editing functions are of course supported :

CTRL + C : Copy the selection
CTRL + X : Cut the selection
CTRL + V : Paste the selection
CTRL + Z : Cancels the previous entry 
CTRL + Y : Redo the previous cancelled operation

All these commands are summarized in the two lines at the bottom of the screen.

Let's take a look at all these features...

⯈ H E L P
The Help panel is displayed by pressing the F1 key. 
This panel can also be invoked by running the special command !help in the Shell input box.
You can browse this panel and all other panels with the arrow keys.
To exit this panel, activate the OK button or press the Escape key.

⯈ M O V I N G   B E T W E E N   P A N E L S
The F2 key allow you to move the focus from the current panel to the next one.
The current panel is recognizable by its double border while the other panels only have a single border.
Starting from the default current panel which is the Editor panel, you can move to the panel and so on, and returning to the first panel.

These panels are in order :
● The Editor panel
● The Open Files panel
● The Find & Replace panel
● The Explorer panel
● The Outline panel
These panels will be detailed later in this document.

⯈ G I T   M E N U
Some useful common git commands are accessible through the options of a dedicated menu.
This menu is displayed when pressing the F3 key.

These options are as follows :
● Status               : Displays the status of the current branch.
● Log                  : Displays the log for the current repository.
● Add All (.)          : Add all files recursively to the git tracking.
● Commit               : Commits the current changes after prompting for the commit message.
● Push                 : Pushes the current commit to the remote repository.
● Commit & Push        : Commits the current changes and the push it to remote repository.
● Fetch                : This only copies changes from remote repository into your local Git repository.
● Pull (Fetch & Merge) : Copies changes from remote repository to you local repository and merge them with your working copy.
● Initialize           : This will initialize a local repository for a new project.
● Initialize & Push    : This will initialize a local repository for a new project. The remote repository should already exists.
● Clone                : Clone a remote repository into the current local folder. No local repository should exists.
● Configure            : This should normally only be done once, as it saves the Github username and associated password that will be used for git commands.

⯈ S H E L L   I N P U T   B O X

⯈ S E T T I N G S   M E N U

⯈ W O R K S P A C E   M E N U

⯈ M A I N   M E N U

⯈ M A C R O S
Useful common functions can be added to the Macros menu accessible with the ALT + M keystroke.
This menu shows all the macros you previously recorded and the last option allows you to edit and add new macros.

The syntax for the macros file is very simple :
● One line, one macro.
● Each line beginning with a # char is ignored.
● The name of the macro which will be displayed on the menu is the first part of the line BEFORE the : char.
● The command of the macro which will be runned is the second part of the line AFTER the : char.

Macros can interact with dynamic values by using special placeholders which will be replaced by the appropriate value at run time.
Theses placeholders, prefixed by a % char and case sensitive, are as follows :
● %D : Full directory of current file
● %P : Parent directory of current file
● %W : Full directory of current workspace
● %F : Full file name with directory and extension of current file
● %f : File name without path and with extension of current file
● %e : File name without path nor extension of current file
● %L : Line number of current file in editor
● %T : Current timestamp
● %H : Home directory of current user
● %s : OS path separator

⯈ P A N E L S
The main panels displayed are :
● The Editor panel
● The Open Files panel
● The Find & Replace panel
● The Explorer panel
● The Outline panel

⯈ S P E C I A L   C O M M A N D S
These special commands are available when entering command in the Shell input box (F4).
These special commands all have a maximum of 4 letters and are prefixed by an exclamation point (!).
!quit : Quit the editor (same as F12)
!exit : ----------- idem ------------
!bye  : ----------- idem ------------
!log  : Opens the log file of the Lied editor
!out  : Opens the output file for the Shell input box
!foll : Switch the current file in following mode
!tail : --------------- idem --------------------
!shel : Enable the interactive shell if not enabled by default in settings
!next : Switch to the next open file (same as F7)
!prev : Switch to the previous open file (same as F6)
!clos : Closes the current document (same as CTRL + T)
!save : Saves the current document being edited (same as CTRL + S)
!conf : Opens the Lied configuration file for editing
!macr : Opens the macros file for editing
!help : Displays the Help panel (same as F1)
!info : Displays system informations about the computer
!b    : Go to the bottom of the current file
!bott : ------------ idem ------------------
!t    : Go to the top of the current file
!top  : ------------ idem ---------------
!h    : Insert a timestamp string at the current cursor location
!time : ------------------------ idem --------------------------
!uuid : Insert a random UUID at the current cursor location
!lore : Insert a random Lorem Ipsum text at the current cursor location
!42   : Go to that line number

⯈ I N T E R N A L   F I L E S
Some files are generated and managed internally to keep the current settings and history.
These files are located into the .lied directory, which is itself into the user folder.
These files are the following :
● find            : Find & replace history
● history         : Commands history
● lied.ini        : Settings
● lied.log        : Log of the application
● lied.log.bak    : Backup of the log file when archiving
● lied_YYYYMM.zip : Archived log files by month and year
● macros          : Macros that are runned with the ALT+M key
● mru             : Most Recent Used files
● output          : Commands output
`

// ----------------------------------------------------------------------------
//    Some common useful icons
// ----------------------------------------------------------------------------
//   ♯
//   ●
//   ⯈
//   ⎇
//   ⟟
//   🗨
//   ⚠
//   ©
// ----------------------------------------------------------------------------
