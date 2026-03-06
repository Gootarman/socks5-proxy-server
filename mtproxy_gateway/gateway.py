"""TCP multi-port gateway for dockerized nineseconds/mtg upstream."""

from __future__ import annotations

import argparse
import asyncio
from contextlib import suppress

from db_setup import get_connection, init_db


class UserGatewayManager:
    def __init__(self, upstream_host: str, upstream_port: int, listen_host: str, poll_interval: float):
        self.upstream_host = upstream_host
        self.upstream_port = upstream_port
        self.listen_host = listen_host
        self.poll_interval = poll_interval
        self.servers: dict[int, asyncio.AbstractServer] = {}
        self.clients_by_port: dict[int, set[asyncio.StreamWriter]] = {}

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

    async def handle_client(self, client_reader: asyncio.StreamReader, client_writer: asyncio.StreamWriter) -> None:
        sockname = client_writer.get_extra_info("sockname")
        listen_port = int(sockname[1]) if sockname else None
        if listen_port is not None:
            self.clients_by_port.setdefault(listen_port, set()).add(client_writer)

        try:
            upstream_reader, upstream_writer = await asyncio.open_connection(
                self.upstream_host, self.upstream_port
            )
        except OSError:
            client_writer.close()
            await client_writer.wait_closed()
            if listen_port is not None:
                self.clients_by_port.get(listen_port, set()).discard(client_writer)
            return

        try:
            await asyncio.gather(
                self.relay_stream(client_reader, upstream_writer),
                self.relay_stream(upstream_reader, client_writer),
            )
        finally:
            if listen_port is not None:
                self.clients_by_port.get(listen_port, set()).discard(client_writer)

    def active_ports(self) -> set[int]:
        with get_connection() as connection:
            rows = connection.execute(
                "SELECT listen_port FROM users WHERE is_active = 1"
            ).fetchall()
        return {int(row["listen_port"]) for row in rows if row["listen_port"] is not None}

    async def open_port(self, port: int) -> None:
        if port in self.servers:
            return
        server = await asyncio.start_server(self.handle_client, self.listen_host, port)
        self.servers[port] = server
        self.clients_by_port.setdefault(port, set())
        print(f"Port enabled: {self.listen_host}:{port} -> {self.upstream_host}:{self.upstream_port}")

    async def close_port(self, port: int) -> None:
        server = self.servers.pop(port, None)
        if not server:
            return
        server.close()
        await server.wait_closed()

        clients = list(self.clients_by_port.pop(port, set()))
        for writer in clients:
            with suppress(Exception):
                writer.close()
        for writer in clients:
            with suppress(Exception):
                await writer.wait_closed()

        print(f"Port disabled: {self.listen_host}:{port}; dropped {len(clients)} active connection(s)")

    async def sync_ports(self) -> None:
        desired = self.active_ports()
        current = set(self.servers.keys())
        for port in sorted(desired - current):
            await self.open_port(port)
        for port in sorted(current - desired):
            await self.close_port(port)

    async def run(self) -> None:
        init_db()
        while True:
            await self.sync_ports()
            await asyncio.sleep(self.poll_interval)


async def main_async(args: argparse.Namespace) -> None:
    manager = UserGatewayManager(
        upstream_host=args.upstream_host,
        upstream_port=args.upstream_port,
        listen_host=args.listen_host,
        poll_interval=args.poll_interval,
    )
    await manager.run()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="MTG user-aware TCP gateway")
    parser.add_argument("--listen-host", default="0.0.0.0")
    parser.add_argument("--upstream-host", default="127.0.0.1")
    parser.add_argument("--upstream-port", type=int, default=3128)
    parser.add_argument("--poll-interval", type=float, default=2.0)
    return parser.parse_args()


if __name__ == "__main__":
    asyncio.run(main_async(parse_args()))
