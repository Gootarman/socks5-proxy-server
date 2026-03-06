# MTG Admin Gateway (for `nineseconds/mtg`)

Этот модуль — админ-прослойка для `nineseconds/mtg`, упакованная в `docker-compose`.

## Что это делает

- поднимает контейнер `mtg`;
- поднимает `gateway`, который открывает отдельный TCP-порт на каждого активного пользователя;
- поднимает web-admin (`Flask`), где можно:
  - создать пользователя;
  - получить `tg://proxy` ссылку;
  - отключить / удалить пользователя.

> Важно: Telegram-клиент не умеет отправлять произвольный «токен первой строкой», поэтому рабочая модель — **персональный порт на пользователя**.

## Структура

```text
mtproxy_gateway/
├─ docker-compose.yml
├─ Dockerfile
├─ .env.example
├─ admin_panel.py
├─ gateway.py
├─ db_setup.py
├─ generate_token.py
├─ templates/
└─ requirements.txt
```

## Быстрый старт через Docker Compose

1. Перейдите в папку:

```bash
cd mtproxy_gateway
```

2. Подготовьте env:

```bash
cp .env.example .env
# отредактируйте PUBLIC_HOST и MTG_SECRET
```

3. Запустите стек:

```bash
docker compose up -d --build
```

4. Откройте admin UI:

- `http://<ваш_хост>:${ADMIN_PORT}` (по умолчанию `8000`)

## Переменные `.env`

- `PUBLIC_HOST` — внешний IP/домен сервера (используется в `tg://proxy` ссылках).
- `MTG_SECRET` — секрет для контейнера `mtg`.
- `MTG_UPSTREAM_PORT` — внутренний порт `mtg` в docker-сети (обычно `3128`).
- `PORT_RANGE_START`, `PORT_RANGE_END` — диапазон портов, который gateway публикует для пользователей.
- `ADMIN_PORT` — порт web-admin.
- `GATEWAY_POLL_INTERVAL` — как часто gateway перечитывает БД пользователей.

## Как работает доступ

- При создании пользователя ему назначается `listen_port` (авто из диапазона или вручную).
- Gateway открывает этот порт и проксирует в `mtg`.
- В UI показывается готовая ссылка:
  - `tg://proxy?server=<PUBLIC_HOST>&port=<listen_port>&secret=<MTG_SECRET>`
- Если пользователя отключить, gateway закроет его порт.

## Управление

```bash
docker compose logs -f admin gateway mtg
docker compose down
docker compose down -v
```

## Локальный запуск без Docker (опционально)

```bash
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python db_setup.py
python admin_panel.py
```

## Безопасность

- Ограничьте доступ к admin UI (VPN, reverse-proxy auth, firewall).
- Откройте во внешнюю сеть только диапазон пользовательских портов и `ADMIN_PORT` (если нужно).
- Храните `.env` приватно.
