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
package help

var Help = `♯ [yellow]L I E D   -   Copyright © JPL 2024

[white]Lied is a TUI (Text User Interface) Editor.
Lied is written in Go and has been tested on Linux sytem.
Built from source, it should run on Windows or MacOS systems as well.

Pay honour to whom honour is due, packages used in this project are as follows :
[red]rivo/tview    :[white] Package tview implements rich widgets for terminal based user interfaces.
[red]gdamore/tcell :[white] Tcell is an alternate terminal package, similar in some ways to termbox, but better in others. 
[red]pgavlin/femto :[white] An editor component for tview. Derived from the micro editor. 

⯈ The main functions are reachable through function keys :

[red]F1  :[white] This help screen
[red]F2  :[white] Switch from the current panel to the next
[red]F3  :[white] Access to Git Menu for running Git common commands
[red]F4  :[white] Access to shell to run system and special commands
[red]F6  :[white] Switch to the previous open file
[red]F7  :[white] Switch to the next open file
[red]F8  :[white] Access to the settings of Lied
[red]F9  :[white] Access to the workspace menu
[red]F10 :[white] Access to the main menu of Lied
[red]F12 :[white] Exit

⯈ Alternate common functions are also reachable through CTRL and ALT keys :

[red]CTRL + F :[white] Switch to the Find & Replace panel, or go back to the Editor panel
[red]CTRL + S :[white] Saves the current document being edited
[red]ALT  + S :[white] Saves the current document being edited under another name
[red]CTRL + N :[white] Opens a new blank document
[red]CTRL + O :[white] Opens an existing document for editing
[red]CTRL + T :[white] Closes the current document
[red]ALT  + M :[white] Opens the macros menu
[red]ALT  + Q :[white] Temporary escape to the Shell

⯈ When editing a text, common editing functions are of course supported :

[red]CTRL + C :[white] Copy the selection
[red]CTRL + X :[white] Cut the selection
[red]CTRL + V :[white] Paste the selection
[red]CTRL + Z :[white] Cancels the previous entry 
[red]CTRL + Y :[white] Redo the previous cancelled operation

All these commands are summarized in the two lines at the bottom of the screen.

Let's take a look at all these features...

⯈ [red]H E L P[white]
The Help panel is displayed by pressing the [red]F1[white] key. 
This panel can also be invoked by running the special command [red]!help[white] in the Shell input box.
You can browse this panel and all other panels with the arrow keys.
To exit this panel, activate the [red]OK[white] button or press the Escape key.

⯈ [red]M O V I N G   B E T W E E N   P A N E L S[white]
The [red]F2[white] key allow you to move the focus from the current panel to the next one.
The current panel is recognizable by its double border while the other panels only have a single border.
Starting from the default current panel which is the Editor panel, you can move to the panel and so on, and returning to the first panel.

These panels are in order :
● The [red]Editor[white] panel
● The [red]Open Files[white] panel
● The [red]Find & Replace[white] panel
● The [red]Explorer[white] panel
● The [red]Outline[white] panel
These panels will be detailed later in this document.

⯈ [red]G I T   M E N U[white]
Some useful common git commands are accessible through the options of a dedicated menu.
This menu is displayed when pressing the [red]F3[white] key.

These options are as follows :
● [red]Status               :[white] Displays the status of the current branch.
● [red]Log                  :[white] Displays the log for the current repository.
● [red]Add All (.)          :[white] Add all files recursively to the git tracking.
● [red]Commit               :[white] Commits the current changes after prompting for the commit message.
● [red]Push                 :[white] Pushes the current commit to the remote repository.
● [red]Commit & Push        :[white] Commits the current changes and the push it to remote repository.
● [red]Fetch                :[white] This only copies changes from remote repository into your local Git repository.
● [red]Pull (Fetch & Merge) :[white] Copies changes from remote repository to you local repository and merge them with your working copy.
● [red]Initialize           :[white] This will initialize a local repository for a new project.
● [red]Initialize & Push    :[white] This will initialize a local repository for a new project. The remote repository should already exists.
● [red]Clone                :[white] Clone a remote repository into the current local folder. No local repository should exists.
● [red]Configure            :[white] This should normally only be done once, as it saves the Github username and associated password that will be used for git commands.

⯈ [red]S H E L L   I N P U T   B O X[white]

⯈ [red]S E T T I N G S   M E N U[white]

⯈ [red]W O R K S P A C E   M E N U[white]

⯈ [red]M A I N   M E N U[white]

⯈ [red]M A C R O S[white]
Useful common functions can be added to the Macros menu accessible with the ALT + M keystroke.
This menu shows all the macros you previously recorded and the last option allows you to edit and add new macros.

The syntax for the macros file is very simple :
● One line, one macro.
● Each line beginning with a [red]#[white] char is ignored.
● The name of the macro which will be displayed on the menu is the first part of the line BEFORE the [red]:[white] char.
● The command of the macro which will be runned is the second part of the line AFTER the [red]:[white] char.

Macros can interact with dynamic values by using special placeholders which will be replaced by the appropriate value at run time.
Theses placeholders, prefixed by a [red]%[white] char and case sensitive, are as follows :
● [red]%D :[white] Full directory of current file
● [red]%P :[white] Parent directory of current file
● [red]%W :[white] Full directory of current workspace
● [red]%F :[white] Full file name with directory and extension of current file
● [red]%f :[white] File name without path and with extension of current file
● [red]%e :[white] File name without path nor extension of current file
● [red]%L :[white] Line number of current file in editor
● [red]%T :[white] Current timestamp
● [red]%H :[white] Home directory of current user
● [red]%s :[white] OS path separator

⯈ [red]P A N E L S[white]
The main panels displayed are :
● The [red]Editor[white] panel
● The [red]Open Files[white] panel
● The [red]Find & Replace[white] panel
● The [red]Explorer[white] panel
● The [red]Outline[white] panel

⯈ [red]S P E C I A L   C O M M A N D S[white]
These special commands are available when entering command in the Shell input box (F4).
These special commands all have a maximum of 4 letters and are prefixed by an exclamation point (!).
[red]!quit :[white] Quit the editor (same as F12)
[red]!exit :[white] ----------- idem ------------
[red]!bye  :[white] ----------- idem ------------
[red]!log  :[white] Opens the log file of the Lied editor
[red]!out  :[white] Opens the output file for the Shell input box
[red]!foll :[white] Switch the current file in following mode
[red]!tail :[white] --------------- idem --------------------
[red]!shel :[white] Enable the interactive shell if not enabled by default in settings
[red]!next :[white] Switch to the next open file (same as F7)
[red]!prev :[white] Switch to the previous open file (same as F6)
[red]!clos :[white] Closes the current document (same as CTRL + T)
[red]!save :[white] Saves the current document being edited (same as CTRL + S)
[red]!conf :[white] Opens the Lied configuration file for editing
[red]!macr :[white] Opens the macros file for editing
[red]!help :[white] Displays the Help panel (same as F1)
[red]!info :[white] Displays system informations about the computer
[red]!b    :[white] Go to the bottom of the current file
[red]!bott :[white] ------------ idem ------------------
[red]!t    :[white] Go to the top of the current file
[red]!top  :[white] ------------ idem ---------------
[red]!h    :[white] Insert a timestamp string at the current cursor location
[red]!time :[white] ------------------------ idem --------------------------
[red]!uuid :[white] Insert a random UUID at the current cursor location
[red]!lore :[white] Insert a random Lorem Ipsum text at the current cursor location
[red]!42   :[white] Go to that line number

⯈ [red]I N T E R N A L   F I L E S[white]
Some files are generated and managed internally to keep the current settings and history.
These files are located into the [red].lied[white] directory, which is itself into the user folder.
These files are the following :
● [red]find            :[white] Find & replace history
● [red]history         :[white] Commands history
● [red]lied.ini        :[white] Settings
● [red]lied.log        :[white] Log of the application
● [red]lied.log.bak    :[white] Backup of the log file when archiving
● [red]lied_YYYYMM.zip :[white] Archived log files by month and year
● [red]macros          :[white] Macros that are runned with the ALT+M key
● [red]mru             :[white] Most Recent Used files
● [red]output          :[white] Commands output
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
