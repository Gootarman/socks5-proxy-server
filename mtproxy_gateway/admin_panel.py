"""Flask admin panel for MTG gateway user administration."""

from __future__ import annotations

import os
import sqlite3

from flask import Flask, redirect, render_template, request, url_for

from db_setup import get_connection, init_db
from generate_token import add_user

app = Flask(__name__)

PUBLIC_HOST = os.getenv("PUBLIC_HOST", "127.0.0.1")
MTG_SECRET = os.getenv("MTG_SECRET", "")
PORT_RANGE_START = int(os.getenv("PORT_RANGE_START", "11000"))
PORT_RANGE_END = int(os.getenv("PORT_RANGE_END", "11999"))


def user_proxy_link(port: int) -> str:
    if not MTG_SECRET:
        return ""
    return f"tg://proxy?server={PUBLIC_HOST}&port={port}&secret={MTG_SECRET}"


@app.before_request
def ensure_db() -> None:
    init_db()


@app.get("/")
def index():
    with get_connection() as connection:
        users = connection.execute(
            "SELECT id, username, token, listen_port, is_active, created_at FROM users ORDER BY id DESC"
        ).fetchall()

    users_with_links = [
        {**dict(user), "proxy_link": user_proxy_link(user["listen_port"])} for user in users
    ]
    return render_template("index.html", users=users_with_links, public_host=PUBLIC_HOST)


@app.route("/add", methods=["GET", "POST"])
def add_user_page():
    error = ""
    if request.method == "POST":
        username = request.form.get("username", "").strip()
        manual_port = request.form.get("listen_port", "").strip()
        port = int(manual_port) if manual_port else None

        if username:
            try:
                add_user(username, port, PORT_RANGE_START, PORT_RANGE_END)
                return redirect(url_for("index"))
            except (sqlite3.IntegrityError, RuntimeError, ValueError) as exc:
                error = str(exc)

    return render_template(
        "add_user.html",
        error=error,
        range_start=PORT_RANGE_START,
        range_end=PORT_RANGE_END,
    )


@app.get("/user/<int:user_id>")
def user_detail(user_id: int):
    with get_connection() as connection:
        user = connection.execute(
            "SELECT id, username, token, listen_port, is_active, created_at FROM users WHERE id = ?",
            (user_id,),
        ).fetchone()

    if not user:
        return redirect(url_for("index"))

    data = dict(user)
    data["proxy_link"] = user_proxy_link(data["listen_port"])
    return render_template("user_detail.html", user=data, public_host=PUBLIC_HOST)


@app.post("/user/<int:user_id>/toggle")
def toggle_user(user_id: int):
    with get_connection() as connection:
        connection.execute(
            "UPDATE users SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END WHERE id = ?",
            (user_id,),
        )
        connection.commit()
    return redirect(url_for("user_detail", user_id=user_id))


@app.post("/user/<int:user_id>/delete")
def delete_user(user_id: int):
    with get_connection() as connection:
        connection.execute("DELETE FROM users WHERE id = ?", (user_id,))
        connection.commit()
    return redirect(url_for("index"))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.getenv("ADMIN_PORT", "8000")), debug=False)
