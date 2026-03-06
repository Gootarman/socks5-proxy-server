"""Database initialization and helpers for MTG admin gateway."""

from __future__ import annotations

import os
import sqlite3
from pathlib import Path

DB_PATH = Path(os.getenv("DB_PATH", Path(__file__).resolve().parent / "users.db"))


SCHEMA = """
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    token TEXT NOT NULL UNIQUE,
    listen_port INTEGER NOT NULL UNIQUE,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
"""


def get_connection(db_path: Path = DB_PATH) -> sqlite3.Connection:
    connection = sqlite3.connect(db_path)
    connection.row_factory = sqlite3.Row
    return connection


def migrate_schema(connection: sqlite3.Connection) -> None:
    """Best-effort migration for older DB versions."""
    columns = {
        row[1] for row in connection.execute("PRAGMA table_info(users)").fetchall()
    }
    if "listen_port" not in columns:
        connection.execute("ALTER TABLE users ADD COLUMN listen_port INTEGER")
        rows = connection.execute("SELECT id FROM users ORDER BY id").fetchall()
        for idx, row in enumerate(rows, start=11000):
            connection.execute(
                "UPDATE users SET listen_port = ? WHERE id = ?", (idx, row["id"])
            )
    connection.commit()


def init_db() -> None:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    with get_connection() as connection:
        connection.execute(SCHEMA)
        migrate_schema(connection)
        connection.commit()


if __name__ == "__main__":
    init_db()
    print(f"Database initialized: {DB_PATH}")
