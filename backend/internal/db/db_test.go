package db

import "testing"

func TestIsSelfTransactional(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "explicit BEGIN TRANSACTION",
			sql: `-- Migration 025
BEGIN TRANSACTION;
UPDATE folders SET parent_id = NULL WHERE parent_id = 1;
COMMIT;`,
			want: true,
		},
		{
			name: "bare BEGIN",
			sql: `BEGIN;
INSERT INTO foo VALUES (1);
COMMIT;`,
			want: true,
		},
		{
			name: "BEGIN with leading whitespace",
			sql: `  BEGIN TRANSACTION;
INSERT INTO foo VALUES (1);
COMMIT;`,
			want: true,
		},
		{
			name: "normal migration without transaction",
			sql: `ALTER TABLE notes ADD COLUMN color TEXT;
CREATE INDEX idx_notes_color ON notes(color);`,
			want: false,
		},
		{
			name: "trigger containing BEGIN...END is NOT self-transactional",
			sql: `CREATE TRIGGER notes_fts_insert AFTER INSERT ON notes
BEGIN
  INSERT INTO notes_fts(rowid, title, content) VALUES (new.note_rowid, new.title, new.content);
END;`,
			want: false,
		},
		{
			name: "multiple triggers with BEGIN...END",
			sql: `CREATE TRIGGER t1 AFTER INSERT ON foo
BEGIN
  UPDATE bar SET x = 1;
END;
CREATE TRIGGER t2 AFTER DELETE ON foo
BEGIN
  DELETE FROM bar WHERE id = old.id;
END;`,
			want: false,
		},
		{
			name: "empty SQL",
			sql:  "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSelfTransactional(tt.sql)
			if got != tt.want {
				t.Errorf("isSelfTransactional() = %v, want %v", got, tt.want)
			}
		})
	}
}
