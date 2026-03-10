"""Database initialization and helpers for MTG admin gateway."""

from __future__ import annotations

import os
import sqlite3
from pathlib import Path

DB_PATH = Path(os.getenv("DB_PATH", Path(__file__).resolve().parent / "users.db"))


USERS_SCHEMA = """
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    token TEXT NOT NULL UNIQUE,
    listen_port INTEGER NOT NULL UNIQUE,
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
"""

RESERVED_PORTS_SCHEMA = """
CREATE TABLE IF NOT EXISTS reserved_ports (
    port INTEGER PRIMARY KEY,
    reserved_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reserved_by_user_id INTEGER
);
"""

PORT_OVERRIDES_SCHEMA = """
CREATE TABLE IF NOT EXISTS port_overrides (
    port INTEGER PRIMARY KEY,
    is_enabled INTEGER NOT NULL DEFAULT 1,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
"""


GATEWAY_CONTROL_SCHEMA = """
CREATE TABLE IF NOT EXISTS gateway_control (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    restart_token INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
"""


def get_connection(db_path: Path = DB_PATH) -> sqlite3.Connection:
    connection = sqlite3.connect(db_path)
    connection.row_factory = sqlite3.Row
    return connection


def _table_exists(connection: sqlite3.Connection, table_name: str) -> bool:
    row = connection.execute(
        "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?",
        (table_name,),
    ).fetchone()
    return row is not None


def _next_free_port(used_ports: set[int], start: int = 11000) -> int:
    port = start
    while port in used_ports:
        port += 1
    used_ports.add(port)
    return port


def _rebuild_users_table(connection: sqlite3.Connection) -> None:
    connection.execute(
        """
        CREATE TABLE users_new (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            token TEXT NOT NULL UNIQUE,
            listen_port INTEGER NOT NULL UNIQUE,
            is_active INTEGER NOT NULL DEFAULT 1,
            created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
        );
        """
    )

    rows = connection.execute(
        "SELECT id, username, token, listen_port, is_active, created_at FROM users ORDER BY id DESC"
    ).fetchall()

    used_ports: set[int] = set()
    for row in rows:
        port = row["listen_port"]
        if port is None:
            port = _next_free_port(used_ports)
        else:
            try:
                port = int(port)
            except (TypeError, ValueError):
                port = _next_free_port(used_ports)

        if port in used_ports:
            continue

        used_ports.add(port)
        connection.execute(
            """
            INSERT OR IGNORE INTO users_new (username, token, listen_port, is_active, created_at)
            VALUES (?, ?, ?, ?, ?)
            """,
            (
                row["username"],
                row["token"],
                port,
                1 if row["is_active"] else 0,
                row["created_at"] or "1970-01-01 00:00:00",
            ),
        )

    connection.execute("DROP TABLE users")
    connection.execute("ALTER TABLE users_new RENAME TO users")


def _needs_rebuild(connection: sqlite3.Connection) -> bool:
    columns = {row[1]: row for row in connection.execute("PRAGMA table_info(users)").fetchall()}

    if "listen_port" not in columns:
        return True

    listen_col = columns["listen_port"]
    listen_not_null = bool(listen_col[3])

    unique_on_listen = False
    for idx in connection.execute("PRAGMA index_list(users)").fetchall():
        if not idx[2]:
            continue
        idx_name = idx[1]
        idx_cols = [r[2] for r in connection.execute(f"PRAGMA index_info({idx_name})").fetchall()]
        if idx_cols == ["listen_port"]:
            unique_on_listen = True
            break

    return not (listen_not_null and unique_on_listen)


def _ensure_reserved_ports(connection: sqlite3.Connection) -> None:
    connection.execute(RESERVED_PORTS_SCHEMA)
    rows = connection.execute("SELECT id, listen_port FROM users WHERE listen_port IS NOT NULL").fetchall()
    for row in rows:
        connection.execute(
            "INSERT OR IGNORE INTO reserved_ports (port, reserved_by_user_id) VALUES (?, ?)",
            (int(row["listen_port"]), row["id"]),
        )


def _ensure_port_overrides(connection: sqlite3.Connection) -> None:
    connection.execute(PORT_OVERRIDES_SCHEMA)


def _ensure_gateway_control(connection: sqlite3.Connection) -> None:
    connection.execute(GATEWAY_CONTROL_SCHEMA)
    connection.execute(
        "INSERT OR IGNORE INTO gateway_control (id, restart_token) VALUES (1, 0)"
    )


def migrate_schema(connection: sqlite3.Connection) -> None:
    if not _table_exists(connection, "users"):
        return

    if _needs_rebuild(connection):
        _rebuild_users_table(connection)

    _ensure_reserved_ports(connection)
    _ensure_port_overrides(connection)
    _ensure_gateway_control(connection)
    connection.commit()


def init_db() -> None:
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    with get_connection() as connection:
        connection.execute(USERS_SCHEMA)
        migrate_schema(connection)
        connection.commit()


if __name__ == "__main__":
    init_db()
    print(f"Database initialized: {DB_PATH}")
