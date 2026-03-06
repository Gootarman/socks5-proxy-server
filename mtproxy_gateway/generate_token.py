"""Generate secure tokens and optionally add users to the database."""

from __future__ import annotations

import argparse
import secrets
import sqlite3
from typing import Optional

from db_setup import get_connection, init_db


def create_token(length: int = 32) -> str:
    """Create a URL-safe token."""
    return secrets.token_urlsafe(length)


def add_user(username: str, token: Optional[str] = None) -> str:
    """Add user to DB and return assigned token."""
    init_db()
    user_token = token or create_token(24)

    with get_connection() as connection:
        connection.execute(
            "INSERT INTO users (username, token, is_active) VALUES (?, ?, 1)",
            (username, user_token),
        )
        connection.commit()

    return user_token


def main() -> None:
    parser = argparse.ArgumentParser(description="Generate token for MTProxy gateway user")
    parser.add_argument("username", nargs="?", help="Username to create in DB")
    parser.add_argument("--length", type=int, default=24, help="Token size modifier")
    args = parser.parse_args()

    if args.username:
        token = create_token(args.length)
        try:
            saved_token = add_user(args.username, token)
            print(f"User '{args.username}' created")
            print(f"Token: {saved_token}")
        except sqlite3.IntegrityError:
            print("Failed to create user: token collision. Retry command.")
    else:
        print(create_token(args.length))


if __name__ == "__main__":
    main()
