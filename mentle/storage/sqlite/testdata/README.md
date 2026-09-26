# External SQLite v1 fixture

`external_v1.sqlite` was created with Python 3.11's standard-library `sqlite3` (SQLite 3.53.1), not Mentle's `Open` or modernc. It is a real SQLite database in rollback-journal (`DELETE`) mode, with `PRAGMA user_version = 1` and `legacy_notes(id INTEGER PRIMARY KEY, content TEXT NOT NULL)` containing `(7, 'written by external sqlite3')`. The test copies it to a temporary path before opening or writing, keeping this checked-in fixture immutable. No Python executable is needed to run the Go test.

To regenerate deliberately, from the `mentle` directory:

```sh
python -c 'import pathlib, sqlite3; p=pathlib.Path("storage/sqlite/testdata/external_v1.sqlite"); p.unlink(missing_ok=True); db=sqlite3.connect(p); db.execute("PRAGMA user_version = 1"); db.execute("CREATE TABLE legacy_notes (id INTEGER PRIMARY KEY, content TEXT NOT NULL)"); db.execute("INSERT INTO legacy_notes (id, content) VALUES (?, ?)", (7, "written by external sqlite3")); db.commit(); db.close()'
```
