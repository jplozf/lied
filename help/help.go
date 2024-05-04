// ****************************************************************************
//
//	 _____ _____ _____ _____
//	|   __|     |   __|  |  |
//	|  |  |  |  |__   |     |
//	|_____|_____|_____|__|__|
//
// ****************************************************************************
// G O S H   -   Copyright © JPL 2023
// ****************************************************************************
package help

import "lied/ui"

// ****************************************************************************
// SelfInit()
// ****************************************************************************
func SelfInit(a any) {
	SetHelp()
}

// ****************************************************************************
// SetHelp()
// ****************************************************************************
func SetHelp() {
	ui.TxtHelp.SetDynamicColors(true).SetText(`[yellow]		 _____ _____ _____ _____
		|   __|     |   __|  |  |
		|  |  |  |  |__   |     |
		|_____|_____|_____|__|__|
        [red]Copyright jpl@ozf.fr 2024

[white]Gosh is a TUI (Text User Interface) for common management functions on a Linux system.
Gosh is written in Go. The main layout interface is inspired by [green]AS400[white] text console.
		
	╔════╦═══════════╦═══════╗
	║ [yellow]F1[white] ║ [red]This Help[white] ║ [yellow]!help[white] ║
	╚════╩═══════════╩═══════╝
	
	╔═══════════════════════════════════════════════════╗
 	║           [red]Fast access to common features[white]          ║
 	╠═════╦══════════════════════════════╦══════════════╣
 	║ [yellow]F1[white]  ║ This help                    ║ [yellow]!help[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F2[white]  ║ Shell                        ║ [yellow]!shel[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F3[white]  ║ Files Manager                ║ [yellow]!file[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F4[white]  ║ Process and Services Manager ║ [yellow]!proc[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F5[white]  ║ (refresh)                    ║              ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F6[white]  ║ Text Editor                  ║ [yellow]!edit[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F7[white]  ║ Network Manager              ║ [yellow]!net[white]         ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F8[white]  ║ (special functions)          ║              ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F9[white]  ║ SQLite3 Manager              ║ [yellow]!sql[white]         ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F10[white] ║ Users Manager                ║ [yellow]!user[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F11[white] ║ Dashboard                    ║ [yellow]!dash[white]        ║
 	╠═════╬══════════════════════════════╬══════════════╣
 	║ [yellow]F12[white] ║ Exit                         ║ [yellow]!quit[white], [yellow]!exit[white] ║
 	╚═════╩══════════════════════════════╩══════════════╝


	╔════╦═══════╦═══════╗
	║ [yellow]F2[white] ║ [red]Shell[white] ║ [yellow]!shel[white] ║
	╚════╩═══════╩═══════╝


	╔════╦═══════════════╦═══════╗
	║ [yellow]F3[white] ║ [red]Files Manager[white] ║ [yellow]!file[white] ║
	╚════╩═══════════════╩═══════╝

	[yellow]TAB  [white]  : Move between panels
	[yellow]Del  [white]  : Delete the file or folder highlighted or the selection
	[yellow]Ins  [white]  : Add the current file or folder to the selection
	[yellow]Ctrl+A[white] : Select or unselect all the files and folders in the current folder
	[yellow]Ctrl+C[white] : Select or unselect all the files and folders in the current folder
	
	╔════╦══════════════════════════════╦═══════╗
	║ [yellow]F4[white] ║ [red]Process and Services Manager[white] ║ [yellow]!proc[white] ║
	╚════╩══════════════════════════════╩═══════╝

	╔════╦════════╦═══════╗
	║ [yellow]F6[white] ║ [red]Editor[white] ║ [yellow]!edit[white] ║
	╚════╩════════╩═══════╝
	
	╔════╦═════════════════╦══════╗
	║ [yellow]F7[white] ║ [red]Network Manager[white] ║ [yellow]!net[white] ║
	╚════╩═════════════════╩══════╝

 	╔════╦═════════════════╦══════╗
 	║ [yellow]F9[white] ║ [red]SQLite3 Manager[white] ║ [yellow]!sql[white] ║
 	╚════╩═════════════════╩══════╝

Here are the .commands available :
 	╔════════════════╦═══════════════════════════════════════════════════╗
 	║ [yellow].OPEN database[white] ║ Open the database by its file name                ║
 	╠════════════════╬═══════════════════════════════════════════════════╣
 	║ [yellow].TABLE[white]         ║ List all tables available in the current database ║
 	╠════════════════╬═══════════════════════════════════════════════════╣
 	║ [yellow].DATABASE[white]      ║ List names and files of attached databases        ║
 	╠════════════════╬═══════════════════════════════════════════════════╣
 	║ [yellow].SCHEMA table[white]  ║ Show the CREATE statements for the matching table ║
 	╠════════════════╬═══════════════════════════════════════════════════╣
 	║ [yellow].COLUMNS table[white] ║ Show the columns types for the matching table     ║
 	╚════════════════╩═══════════════════════════════════════════════════╝

The common SQL statements are summarized as following :
	╔════════════════════════════════════════╦════════════════════════════════════════════════════════════════════════════╗
	║                                        ║ ANALYZE;                                                                   ║
	║                                        ║ or                                                                         ║
	║ SQLite [yellow]ANALYZE[white] Statement               ║ ANALYZE database_name;                                                     ║
	║                                        ║ or                                                                         ║
	║                                        ║ ANALYZE database_name.table_name;                                          ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2….columnN                                           ║
	║ SQLite [yellow]AND/OR[white] Clause                   ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  CONDITION-1 {AND|OR} CONDITION-2;                                   ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]ALTER TABLE[white] Statement           ║ ALTER TABLE table_name ADD COLUMN column_def…;                             ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]ALTER TABLE[white] Statement (Rename)  ║ ALTER TABLE table_name RENAME TO new_table_name;                           ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]ATTACH DATABASE[white] Statement       ║ ATTACH DATABASE ‘DatabaseName’ As ‘Alias-Name’;                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ BEGIN;                                                                     ║
	║ SQLite [yellow]BEGIN TRANSACTION[white] Statement     ║ or                                                                         ║
	║                                        ║ BEGIN EXCLUSIVE TRANSACTION;                                               ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2….columnN                                           ║
	║ SQLite [yellow]BETWEEN[white] Clause                  ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name BETWEEN val-1 AND val-2;                                ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]CREATE INDEX[white] Statement          ║ CREATE INDEX index_name                                                    ║
	║                                        ║ ON table_name ( column_name COLLATE NOCASE );                              ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]CREATE UNIQUE INDEX[white] Statement   ║ CREATE UNIQUE INDEX index_name                                             ║
	║                                        ║ ON table_name ( column1, column2,…columnN);                                ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ CREATE TABLE table_name(                                                   ║
	║                                        ║    column1 datatype,                                                       ║
	║                                        ║    column2 datatype,                                                       ║
	║ SQLite [yellow]CREATE TABLE[white] Statement          ║    column3 datatype,                                                       ║
	║                                        ║    …                                                                       ║
	║                                        ║    columnN data type,                                                      ║
	║                                        ║    PRIMARY KEY( one or more columns ));                                    ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ CREATE TRIGGER database_name.trigger_name                                  ║
	║                                        ║ BEFORE INSERT ON table_name FOR EACH ROW                                   ║
	║                                        ║ BEGIN                                                                      ║
	║ SQLite [yellow]CREATE TRIGGER[white] Statement        ║    stmt1;                                                                  ║
	║                                        ║    stmt2;                                                                  ║
	║                                        ║    …                                                                       ║
	║                                        ║ END;                                                                       ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]CREATE VIEW[white] Statement           ║ CREATE VIEW database_name.view_name  AS                                    ║
	║                                        ║ SELECT statement…;                                                         ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ CREATE VIRTUAL TABLE database_name.table_name USING weblog( access.log );  ║
	║ SQLite [yellow]CREATE VIRTUAL TABLE[white] Statement  ║ or                                                                         ║
	║                                        ║ CREATE VIRTUAL TABLE database_name.table_name USING fts3( );               ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]COMMIT TRANSACTION[white] Statement    ║ COMMIT;                                                                    ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT COUNT(column_name)                                                  ║
	║ SQLite [yellow]COUNT[white] Clause                    ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  CONDITION;                                                          ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DELETE FROM table_name                                                     ║
	║ SQLite [yellow]DELETE[white] Statement                ║ WHERE  {CONDITION};                                                        ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DETACH DATABASE ‘Alias-Name’;                                              ║
	║ SQLite [yellow]DETACH DATABASE[white] Statement       ║                                                                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT DISTINCT column1, column2… columnN                                  ║
	║ SQLite [yellow]DISTINCT[white] Clause                 ║ FROM   table_name;                                                         ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DROP INDEX database_name.index_name;                                       ║
	║ SQLite [yellow]DROP INDEX[white] Statement            ║                                                                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DROP TABLE database_name.table_name;                                       ║
	║ SQLite [yellow]DROP TABLE[white] Statement            ║                                                                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DROP INDEX database_name.view_name;                                        ║
	║ SQLite [yellow]DROP VIEW[white] Statement             ║                                                                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ DROP INDEX database_name.trigger_name;                                     ║
	║ SQLite [yellow]DROP TRIGGER[white] Statement          ║                                                                            ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]EXISTS[white] Clause                   ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name EXISTS (SELECT * FROM   table_name );                   ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ EXPLAIN INSERT statement…;                                                 ║
	║ SQLite [yellow]EXPLAIN[white] Statement               ║ or                                                                         ║
	║                                        ║ EXPLAIN QUERY PLAN SELECT statement…;                                      ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]GLOB[white] Clause                     ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name GLOB { PATTERN };                                       ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT SUM(column_name)                                                    ║
	║ SQLite [yellow]GROUP BY[white] Clause                 ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  CONDITION GROUP BY column_name;                                     ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT SUM(column_name)                                                    ║
	║                                        ║ FROM   table_name                                                          ║
	║ SQLite [yellow]HAVING[white] Clause                   ║ WHERE  CONDITION                                                           ║
	║                                        ║ GROUP BY column_name                                                       ║
	║                                        ║ HAVING (arithmetic function condition);                                    ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]INSERT INTO[white] Statement           ║ INSERT INTO table_name( column1, column2… columnN)                         ║
	║                                        ║ VALUES ( value1, value2… valueN);                                          ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]IN[white] Clause                       ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name IN (val-1, val-2,… val-N);                              ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]LIKE[white] Clause                     ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name LIKE { PATTERN };                                       ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]NOT IN[white] Clause                   ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  column_name NOT IN (val-1, val-2,… val-N);                          ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]ORDER BY[white] Clause                 ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  CONDITION                                                           ║
	║                                        ║ ORDER BY column_name {ASC|DESC};                                           ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]PRAGMA[white] Statement                ║ PRAGMA pragma_name;                                                        ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]RELEASE[white] SAVEPOINT Statement     ║ RELEASE savepoint_name;                                                    ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ REINDEX collation_name;                                                    ║
	║ SQLite [yellow]REINDEX[white] Statement               ║ REINDEX database_name.index_name;                                          ║
	║                                        ║ REINDEX database_name.table_name;                                          ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ ROLLBACK;                                                                  ║
	║ SQLite [yellow]ROLLBACK[white] Statement              ║ or                                                                         ║
	║                                        ║ ROLLBACK TO SAVEPOINT savepoint_name;                                      ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]SAVEPOINT[white] Statement             ║ SAVEPOINT savepoint_name;                                                  ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]SELECT[white] Statement                ║ SELECT column1, column2… columnN                                           ║
	║                                        ║ FROM   table_name;                                                         ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ UPDATE table_name                                                          ║
	║ SQLite [yellow]UPDATE[white] Statement                ║ SET column1 = value1, column2 = value2… columnN=valueN                     ║
	║                                        ║ [ WHERE  CONDITION ];                                                      ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║ SQLite [yellow]VACUUM[white] Statement                ║ VACUUM;                                                                    ║
	╠════════════════════════════════════════╬════════════════════════════════════════════════════════════════════════════╣
	║                                        ║ SELECT column1, column2… columnN                                           ║
	║ SQLite [yellow]WHERE[white] Clause                    ║ FROM   table_name                                                          ║
	║                                        ║ WHERE  CONDITION;                                                          ║
	╚════════════════════════════════════════╩════════════════════════════════════════════════════════════════════════════╝

	╔═════╦═══════════════╦═══════╗
	║ [yellow]F10[white] ║ [red]Users Manager[white] ║ [yellow]!user[white] ║
	╚═════╩═══════════════╩═══════╝

	╔═════╦═══════════╦═══════╗
	║ [yellow]F11[white] ║ [red]Dashboard[white] ║ [yellow]!dash[white] ║
	╚═════╩═══════════╩═══════╝
`)
}
