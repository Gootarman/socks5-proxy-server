"""Token-protected TCP gateway for forwarding traffic to an upstream MTProxy endpoint."""

from __future__ import annotations

import argparse
import asyncio
from typing import Optional

from db_setup import get_connection, init_db


async def relay_stream(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
    """Relay bytes from one stream to another until EOF."""
    try:
        while not reader.at_eof():
            data = await reader.read(65536)
            if not data:
                break
            writer.write(data)
            await writer.drain()
    except (ConnectionResetError, BrokenPipeError):
        pass
    finally:
        try:
            writer.close()
            await writer.wait_closed()
        except Exception:
            pass


def is_valid_token(token: str) -> bool:
    """Check if token exists and active in DB."""
    with get_connection() as connection:
        row = connection.execute(
            "SELECT id FROM users WHERE token = ? AND is_active = 1", (token,)
        ).fetchone()
    return row is not None


async def read_token_line(reader: asyncio.StreamReader) -> Optional[str]:
    """Read token from first line with a simple timeout."""
    try:
        raw = await asyncio.wait_for(reader.readline(), timeout=10)
    except asyncio.TimeoutError:
        return None

    token = raw.decode("utf-8", errors="ignore").strip()
    return token or None


async def handle_client(
    client_reader: asyncio.StreamReader,
    client_writer: asyncio.StreamWriter,
    upstream_host: str,
    upstream_port: int,
) -> None:
    """Validate token and proxy traffic for authorized clients."""
    peer = client_writer.get_extra_info("peername")

    token = await read_token_line(client_reader)
    if not token or not is_valid_token(token):
        client_writer.write(b"AUTH_FAILED\n")
        await client_writer.drain()
        client_writer.close()
        await client_writer.wait_closed()
        print(f"Rejected connection from {peer}")
        return

    try:
        upstream_reader, upstream_writer = await asyncio.open_connection(
            upstream_host, upstream_port
        )
    except OSError as exc:
        client_writer.write(b"UPSTREAM_UNAVAILABLE\n")
        await client_writer.drain()
        client_writer.close()
        await client_writer.wait_closed()
        print(f"Upstream connect failed for {peer}: {exc}")
        return

    print(f"Accepted connection from {peer}")
    await asyncio.gather(
        relay_stream(client_reader, upstream_writer),
        relay_stream(upstream_reader, client_writer),
    )


async def start_gateway(listen_host: str, listen_port: int, upstream_host: str, upstream_port: int) -> None:
    """Start the async TCP gateway server."""
    init_db()
    server = await asyncio.start_server(
        lambda r, w: handle_client(r, w, upstream_host, upstream_port),
        listen_host,
        listen_port,
    )

    print(
        f"Gateway listening on {listen_host}:{listen_port} -> {upstream_host}:{upstream_port}"
    )
    async with server:
        await server.serve_forever()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Token-protected TCP gateway")
    parser.add_argument("--listen-host", default="0.0.0.0")
    parser.add_argument("--listen-port", type=int, default=9090)
    parser.add_argument("--upstream-host", default="127.0.0.1")
    parser.add_argument("--upstream-port", type=int, default=443)
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    asyncio.run(
        start_gateway(
            args.listen_host,
            args.listen_port,
            args.upstream_host,
            args.upstream_port,
        )
    )
