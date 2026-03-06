"""Generate user records for the MTG admin gateway."""

from __future__ import annotations

import argparse
import secrets
import sqlite3

from db_setup import get_connection, init_db


def create_token(length: int = 24) -> str:
    return secrets.token_urlsafe(length)


def find_free_port(start_port: int, end_port: int) -> int:
    with get_connection() as connection:
        used = {
            row[0]
            for row in connection.execute(
                "SELECT listen_port FROM users WHERE listen_port IS NOT NULL"
            ).fetchall()
        }
    for port in range(start_port, end_port + 1):
        if port not in used:
            return port
    raise RuntimeError("No free ports available in the configured range")


def add_user(username: str, port: int | None = None, start_port: int = 11000, end_port: int = 11999) -> tuple[str, int]:
    init_db()
    token = create_token()
    listen_port = port if port is not None else find_free_port(start_port, end_port)

    with get_connection() as connection:
        connection.execute(
            "INSERT INTO users (username, token, listen_port, is_active) VALUES (?, ?, ?, 1)",
            (username, token, listen_port),
        )
        connection.commit()
    return token, listen_port


def main() -> None:
    parser = argparse.ArgumentParser(description="Create MTG gateway user")
    parser.add_argument("username", help="Unique username")
    parser.add_argument("--port", type=int, help="Dedicated listen port")
    parser.add_argument("--start-port", type=int, default=11000)
    parser.add_argument("--end-port", type=int, default=11999)
    args = parser.parse_args()

    try:
        token, listen_port = add_user(args.username, args.port, args.start_port, args.end_port)
        print(f"User '{args.username}' created")
        print(f"Token: {token}")
        print(f"Port: {listen_port}")
    except sqlite3.IntegrityError as exc:
        print(f"Failed to create user: {exc}")


if __name__ == "__main__":
    main()
