"""TCP multi-port gateway for dockerized nineseconds/mtg upstream."""

from __future__ import annotations

import argparse
import asyncio
import time
from contextlib import suppress

from db_setup import get_connection, init_db


class UserGatewayManager:
    def __init__(
        self,
        upstream_host: str,
        upstream_port: int,
        listen_host: str,
        poll_interval: float,
        client_max_age: float,
        enforce_interval: float,
    ):
        self.upstream_host = upstream_host
        self.upstream_port = upstream_port
        self.listen_host = listen_host
        self.poll_interval = poll_interval
        self.client_max_age = client_max_age
        self.enforce_interval = enforce_interval
        self.servers: dict[int, asyncio.AbstractServer] = {}
        self.clients_by_port: dict[int, dict[asyncio.StreamWriter, float]] = {}
        self.last_restart_token = 0

    async def relay_stream(self, reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
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
            with suppress(Exception):
                writer.close()
                await writer.wait_closed()

    async def _drop_clients(self, writers: list[asyncio.StreamWriter]) -> None:
        for writer in writers:
            with suppress(Exception):
                writer.close()
        for writer in writers:
            with suppress(Exception):
                await writer.wait_closed()

    async def handle_client(self, client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        sockname = client_writer.get_extra_info("sockname")
        listen_port = int(sockname[1]) if sockname else None
        if listen_port is not None:
            self.clients_by_port.setdefault(listen_port, {})[client_writer] = time.monotonic()

        try:
            upstream_reader, upstream_writer = await asyncio.open_connection(
                self.upstream_host, self.upstream_port
            )
        except OSError:
            client_writer.close()
            await client_writer.wait_closed()
            if listen_port is not None:
                self.clients_by_port.get(listen_port, {}).pop(client_writer, None)
            return

        try:
            await asyncio.gather(
                self.relay_stream(client_reader, upstream_writer),
                self.relay_stream(upstream_reader, client_writer),
            )
        finally:
            if listen_port is not None:
                self.clients_by_port.get(listen_port, {}).pop(client_writer, None)

    def active_ports(self) -> set[int]:
        with get_connection() as connection:
            rows = connection.execute(
                "SELECT listen_port FROM users WHERE is_active = 1"
            ).fetchall()
        return {int(row["listen_port"]) for row in rows if row["listen_port"] is not None}

    def get_restart_token(self) -> int:
        with get_connection() as connection:
            row = connection.execute(
                "SELECT restart_token FROM gateway_control WHERE id = 1"
            ).fetchone()
        return int(row["restart_token"]) if row else 0

    async def hard_refresh(self) -> None:
        current_ports = sorted(self.servers.keys())
        for port in current_ports:
            await self.close_port(port)
        await self.sync_ports()
        print("Gateway restart signal applied: listeners fully refreshed")

    async def open_port(self, port: int) -> None:
        if port in self.servers:
            return
        server = await asyncio.start_server(self.handle_client, self.listen_host, port)
        self.servers[port] = server
        self.clients_by_port.setdefault(port, {})
        print(f"Port enabled: {self.listen_host}:{port} -> {self.upstream_host}:{self.upstream_port}")

    async def close_port(self, port: int) -> None:
        server = self.servers.pop(port, None)
        if not server:
            return
        server.close()
        await server.wait_closed()

        clients = list(self.clients_by_port.pop(port, {}).keys())
        await self._drop_clients(clients)

        print(f"Port disabled: {self.listen_host}:{port}; dropped {len(clients)} active connection(s)")

    async def enforce_client_lifetime(self) -> None:
        if self.client_max_age <= 0:
            return

        now = time.monotonic()
        dropped_total = 0
        for port, clients in list(self.clients_by_port.items()):
            expired = [w for w, started in list(clients.items()) if now - started >= self.client_max_age]
            if not expired:
                continue
            for w in expired:
                clients.pop(w, None)
            await self._drop_clients(expired)
            dropped_total += len(expired)
            print(f"Session TTL reached on port {port}: dropped {len(expired)} connection(s)")

        if dropped_total:
            print(f"Session TTL sweep complete: dropped {dropped_total} connection(s)")

    async def sync_ports(self) -> None:
        desired = self.active_ports()
        current = set(self.servers.keys())
        for port in sorted(desired - current):
            await self.open_port(port)
        for port in sorted(current - desired):
            await self.close_port(port)

    async def run(self) -> None:
        init_db()
        self.last_restart_token = self.get_restart_token()
        next_enforce = time.monotonic() + self.enforce_interval
        while True:
            await self.sync_ports()
            token = self.get_restart_token()
            if token != self.last_restart_token:
                self.last_restart_token = token
                await self.hard_refresh()
            now = time.monotonic()
            if now >= next_enforce:
                await self.enforce_client_lifetime()
                next_enforce = now + self.enforce_interval
            await asyncio.sleep(self.poll_interval)


async def main_async(args: argparse.Namespace) -> None:
    manager = UserGatewayManager(
        upstream_host=args.upstream_host,
        upstream_port=args.upstream_port,
        listen_host=args.listen_host,
        poll_interval=args.poll_interval,
        client_max_age=args.client_max_age,
        enforce_interval=args.enforce_interval,
    )
    await manager.run()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="MTG user-aware TCP gateway")
    parser.add_argument("--listen-host", default="0.0.0.0")
    parser.add_argument("--upstream-host", default="127.0.0.1")
    parser.add_argument("--upstream-port", type=int, default=3128)
    parser.add_argument("--poll-interval", type=float, default=2.0)
    parser.add_argument(
        "--client-max-age",
        type=float,
        default=0.0,
        help="Force disconnect client sessions older than this many seconds (0 disables)",
    )
    parser.add_argument(
        "--enforce-interval",
        type=float,
        default=5.0,
        help="How often to run client session TTL enforcement",
    )
    return parser.parse_args()


if __name__ == "__main__":
    asyncio.run(main_async(parse_args()))
