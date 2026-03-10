"""Flask admin panel for MTG gateway user administration."""

from __future__ import annotations

import os
import socket
import sqlite3
from datetime import datetime, timezone

from flask import Flask, jsonify, redirect, render_template, request, url_for

from db_setup import get_connection, init_db
from generate_token import add_user

app = Flask(__name__)

PUBLIC_HOST = os.getenv("PUBLIC_HOST", "127.0.0.1")
MTG_SECRET = os.getenv("MTG_SECRET", "")
PORT_RANGE_START = int(os.getenv("PORT_RANGE_START", "11000"))
PORT_RANGE_END = int(os.getenv("PORT_RANGE_END", "11999"))
MONITOR_UPSTREAM_HOST = os.getenv("MONITOR_UPSTREAM_HOST", "mtg")
MONITOR_UPSTREAM_PORT = int(os.getenv("MONITOR_UPSTREAM_PORT", "443"))


def user_proxy_link(port: int) -> str:
    if not MTG_SECRET:
        return ""
    return f"tg://proxy?server={PUBLIC_HOST}&port={port}&secret={MTG_SECRET}"


def check_upstream(host: str, port: int) -> bool:
    try:
        with socket.create_connection((host, port), timeout=1.5):
            return True
    except OSError:
        return False


def request_gateway_restart() -> None:
    with get_connection() as connection:
        connection.execute(
            """
            UPDATE gateway_control
            SET restart_token = restart_token + 1,
                updated_at = CURRENT_TIMESTAMP
            WHERE id = 1
            """
        )
        connection.commit()


def build_port_overview() -> list[dict]:
    with get_connection() as connection:
        users_rows = connection.execute(
            "SELECT listen_port, username, is_active FROM users"
        ).fetchall()
        users_map = {int(r["listen_port"]): dict(r) for r in users_rows if r["listen_port"] is not None}

        reserved_set = {
            int(r[0])
            for r in connection.execute("SELECT port FROM reserved_ports").fetchall()
        }
        blocked_set = {
            int(r[0])
            for r in connection.execute("SELECT port FROM port_overrides WHERE is_enabled = 0").fetchall()
        }

    overview: list[dict] = []
    for port in range(PORT_RANGE_START, PORT_RANGE_END + 1):
        user = users_map.get(port)
        blocked = port in blocked_set
        reserved = port in reserved_set

        if blocked:
            status = "blocked"
            label = "Отключен админом"
        elif user and user["is_active"]:
            status = "active"
            label = f"Активный пользователь: {user['username']}"
        elif user and not user["is_active"]:
            status = "inactive_user"
            label = f"Пользователь отключен: {user['username']}"
        elif reserved:
            status = "reserved"
            label = "Зарезервирован (повторно не выдается)"
        else:
            status = "free"
            label = "Доступен"

        overview.append(
            {
                "port": port,
                "status": status,
                "label": label,
                "is_blocked": blocked,
                "username": user["username"] if user else None,
            }
        )

    return overview


def monitor_data() -> dict:
    with get_connection() as connection:
        total_users = connection.execute("SELECT COUNT(*) FROM users").fetchone()[0]
        active_users = connection.execute(
            "SELECT COUNT(*) FROM users WHERE is_active = 1"
        ).fetchone()[0]
        disabled_users = total_users - active_users
        reserved_ports = connection.execute("SELECT COUNT(*) FROM reserved_ports").fetchone()[0]
        active_ports_rows = connection.execute(
            """
            SELECT u.listen_port
            FROM users u
            LEFT JOIN port_overrides po ON po.port = u.listen_port
            WHERE u.is_active = 1 AND COALESCE(po.is_enabled, 1) = 1
            ORDER BY u.listen_port
            """
        ).fetchall()
        active_ports = [int(r[0]) for r in active_ports_rows]

        blocked_ports = connection.execute(
            "SELECT COUNT(*) FROM port_overrides WHERE is_enabled = 0"
        ).fetchone()[0]

    pool_size = max(PORT_RANGE_END - PORT_RANGE_START + 1, 0)
    free_ports = max(pool_size - reserved_ports - blocked_ports, 0)

    return {
        "total_users": total_users,
        "active_users": active_users,
        "disabled_users": disabled_users,
        "reserved_ports": reserved_ports,
        "blocked_ports": blocked_ports,
        "free_ports": free_ports,
        "active_ports": active_ports,
        "upstream_ok": check_upstream(MONITOR_UPSTREAM_HOST, MONITOR_UPSTREAM_PORT),
        "upstream_target": f"{MONITOR_UPSTREAM_HOST}:{MONITOR_UPSTREAM_PORT}",
        "updated_at": datetime.now(timezone.utc).isoformat(),
    }


def parse_port(raw_port: str) -> int | None:
    value = raw_port.strip()
    if not value:
        return None
    port = int(value)
    if port < 1 or port > 65535:
        raise ValueError("Порт должен быть в диапазоне 1..65535")
    return port


@app.before_request
def ensure_db() -> None:
    init_db()


@app.get("/")
def index():
    edit_user_id = request.args.get("edit", default=None, type=int)
    form_error = request.args.get("error", default="", type=str)

    with get_connection() as connection:
        users = connection.execute(
            "SELECT id, username, token, listen_port, is_active, created_at FROM users ORDER BY id DESC"
        ).fetchall()

    edit_user = None
    if edit_user_id:
        for user in users:
            if int(user["id"]) == edit_user_id:
                edit_user = dict(user)
                break

    users_with_links = [
        {**dict(user), "proxy_link": user_proxy_link(user["listen_port"])} for user in users
    ]
    return render_template(
        "index.html",
        users=users_with_links,
        edit_user=edit_user,
        form_error=form_error,
        public_host=PUBLIC_HOST,
        monitor=monitor_data(),
        port_overview=build_port_overview(),
    )


@app.get("/monitor.json")
def monitor_json():
    payload = monitor_data()
    payload["port_overview"] = build_port_overview()
    return jsonify(payload)


@app.post("/gateway/restart")
def restart_gateway():
    request_gateway_restart()
    return redirect(url_for("index"))


@app.post("/port/<int:port>/toggle")
def toggle_port(port: int):
    if port < PORT_RANGE_START or port > PORT_RANGE_END:
        return redirect(url_for("index"))

    with get_connection() as connection:
        row = connection.execute(
            "SELECT is_enabled FROM port_overrides WHERE port = ?", (port,)
        ).fetchone()
        if row is None:
            connection.execute(
                "INSERT INTO port_overrides (port, is_enabled, updated_at) VALUES (?, 0, CURRENT_TIMESTAMP)",
                (port,),
            )
        elif int(row["is_enabled"]) == 0:
            connection.execute("DELETE FROM port_overrides WHERE port = ?", (port,))
        else:
            connection.execute(
                "UPDATE port_overrides SET is_enabled = 0, updated_at = CURRENT_TIMESTAMP WHERE port = ?",
                (port,),
            )
        connection.commit()

    request_gateway_restart()
    return redirect(url_for("index"))


@app.post("/user/save")
def save_user():
    user_id_raw = request.form.get("user_id", "").strip()
    username = request.form.get("username", "").strip()
    is_active = 1 if request.form.get("is_active") == "on" else 0

    if not username:
        return redirect(url_for("index", error="Укажите username"))

    try:
        requested_port = parse_port(request.form.get("listen_port", ""))
    except ValueError as exc:
        return redirect(url_for("index", error=str(exc), edit=user_id_raw or None))

    if not user_id_raw:
        try:
            add_user(username, requested_port, PORT_RANGE_START, PORT_RANGE_END)
            request_gateway_restart()
            return redirect(url_for("index"))
        except (sqlite3.IntegrityError, RuntimeError, ValueError) as exc:
            return redirect(url_for("index", error=str(exc)))

    user_id = int(user_id_raw)
    with get_connection() as connection:
        current = connection.execute(
            "SELECT id, listen_port FROM users WHERE id = ?", (user_id,)
        ).fetchone()
        if not current:
            return redirect(url_for("index", error="Пользователь не найден"))

        listen_port = int(current["listen_port"])
        next_port = listen_port if requested_port is None else requested_port

        if next_port != listen_port:
            in_use = connection.execute(
                "SELECT 1 FROM reserved_ports WHERE port = ?", (next_port,)
            ).fetchone()
            if in_use:
                return redirect(
                    url_for(
                        "index",
                        edit=user_id,
                        error=f"Порт {next_port} уже зарезервирован и не может быть выдан повторно",
                    )
                )

            connection.execute(
                "INSERT OR IGNORE INTO reserved_ports (port, reserved_by_user_id) VALUES (?, ?)",
                (next_port, user_id),
            )

        try:
            connection.execute(
                """
                UPDATE users
                SET username = ?, listen_port = ?, is_active = ?
                WHERE id = ?
                """,
                (username, next_port, is_active, user_id),
            )
            connection.commit()
        except sqlite3.IntegrityError as exc:
            return redirect(url_for("index", edit=user_id, error=str(exc)))

    request_gateway_restart()
    return redirect(url_for("index"))


@app.get("/user/<int:user_id>")
def user_detail(user_id: int):
    return redirect(url_for("index", edit=user_id))


@app.post("/user/<int:user_id>/toggle")
def toggle_user(user_id: int):
    with get_connection() as connection:
        connection.execute(
            "UPDATE users SET is_active = CASE WHEN is_active = 1 THEN 0 ELSE 1 END WHERE id = ?",
            (user_id,),
        )
        connection.commit()
    request_gateway_restart()
    return redirect(url_for("index", edit=user_id))


@app.post("/user/<int:user_id>/delete")
def delete_user(user_id: int):
    with get_connection() as connection:
        connection.execute("DELETE FROM users WHERE id = ?", (user_id,))
        connection.commit()
    request_gateway_restart()
    return redirect(url_for("index"))


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=int(os.getenv("ADMIN_PORT", "8000")), debug=False)
