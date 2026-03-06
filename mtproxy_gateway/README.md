# MTG Admin Gateway (for `nineseconds/mtg:2`)

Этот модуль — админ-прослойка для `nineseconds/mtg`, упакованная в `docker-compose`.

## Что это делает

- поднимает контейнер `mtg`;
- поднимает `gateway`, который открывает отдельный TCP-порт на каждого активного пользователя;
- поднимает web-admin (`Flask`), где можно:
  - создать пользователя;
  - получить `tg://proxy` ссылку;
  - отключить / удалить пользователя.

> Важно: Telegram-клиент не умеет отправлять произвольный «токен первой строкой», поэтому рабочая модель — **персональный порт на пользователя**.

## Быстрый старт (рабочая схема)

1. Перейдите в папку:

```bash
cd mtproxy_gateway
```

2. Подготовьте env:

```bash
cp .env.example .env
```

3. Укажите домен-прикрытие для FakeTLS (например `ru.wikipedia.org`) и сгенерируйте секрет:

```bash
export MTG_FAKE_TLS_DOMAIN=ru.wikipedia.org
docker run --rm nineseconds/mtg:2 generate-secret --hex ${MTG_FAKE_TLS_DOMAIN}
```

Скопируйте полученный hex-секрет в `.env` как `MTG_SECRET` (обычно он начинается с `ee...`, это нормально для FakeTLS).

4. Запустите стек:

```bash
docker compose up -d --build
```

5. Откройте admin UI:

- `http://<ваш_хост>:<ADMIN_PORT>` (по умолчанию `8000`)

6. Создавайте пользователей в UI — у каждого будет свой `port` и готовая `tg://proxy` ссылка.

## Как именно запускается mtg в compose

Используется корректный синтаксис для `nineseconds/mtg:2`:

```bash
mtg run --bind-to 0.0.0.0:<MTG_UPSTREAM_PORT> --secret <MTG_SECRET>
```

Это уже зашито в `docker-compose.yml`.

## Переменные `.env`

- `PUBLIC_HOST` — внешний IP/домен сервера (используется в `tg://proxy` ссылках).
- `MTG_FAKE_TLS_DOMAIN` — домен для генерации FakeTLS секрета (`ru.wikipedia.org` и т.д.).
- `MTG_SECRET` — секрет для контейнера `mtg`, сгенерированный через `generate-secret --hex`.
- `MTG_UPSTREAM_PORT` — внутренний порт `mtg` в docker-сети (обычно `3128`).
- `PORT_RANGE_START`, `PORT_RANGE_END` — диапазон портов, который gateway публикует для пользователей.
- `ADMIN_PORT` — порт web-admin.
- `GATEWAY_POLL_INTERVAL` — как часто gateway перечитывает БД пользователей.

## Диагностика если «не подключается»

Проверьте по шагам:

```bash
docker compose ps
docker compose logs -f mtg
docker compose logs -f gateway
docker compose logs -f admin
```

Что важно:

- `MTG_SECRET` обязательно должен быть сгенерирован именно для выбранного домена (`generate-secret --hex <domain>`);
- `PUBLIC_HOST` должен указывать на реальный внешний адрес сервера;
- диапазон `PORT_RANGE_START..PORT_RANGE_END` должен быть открыт в firewall/security-group;
- клиент Telegram должен использовать ссылку из UI (server+port+secret).

## Управление

```bash
docker compose down
docker compose down -v
```

## Безопасность

- Ограничьте доступ к admin UI (VPN, reverse-proxy auth, firewall).
- Откройте во внешнюю сеть только диапазон пользовательских портов и `ADMIN_PORT` (если нужно).
- Храните `.env` приватно.
