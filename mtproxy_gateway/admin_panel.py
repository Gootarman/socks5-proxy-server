"""Flask admin panel for managing MTProxy gateway users."""

from __future__ import annotations

from flask import Flask, redirect, render_template, request, url_for

from db_setup import get_connection, init_db
from generate_token import create_token

app = Flask(__name__)


@app.before_request
def ensure_db() -> None:
    init_db()


@app.get("/")
def index():
    with get_connection() as connection:
        users = connection.execute(
            "SELECT id, username, token, is_active, created_at FROM users ORDER BY id DESC"
        ).fetchall()
    return render_template("index.html", users=users)


@app.route("/add", methods=["GET", "POST"])
def add_user():
    if request.method == "POST":
        username = request.form.get("username", "").strip()
        token = request.form.get("token", "").strip() or create_token(24)

        if username:
            with get_connection() as connection:
                connection.execute(
                    "INSERT INTO users (username, token, is_active) VALUES (?, ?, 1)",
                    (username, token),
                )
                connection.commit()
            return redirect(url_for("index"))

    return render_template("add_user.html")


@app.get("/user/<int:user_id>")
def user_detail(user_id: int):
    with get_connection() as connection:
        user = connection.execute(
            "SELECT id, username, token, is_active, created_at FROM users WHERE id = ?",
            (user_id,),
        ).fetchone()

    if not user:
        return redirect(url_for("index"))

    return render_template("user_detail.html", user=user)


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
    app.run(host="0.0.0.0", port=8000, debug=True)
