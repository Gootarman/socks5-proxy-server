# MTG Admin Gateway (for `nineseconds/mtg`)

Этот модуль — админ-прослойка над уже работающим MTProto proxy в Docker-образе `nineseconds/mtg`.

## Идея

`mtg` не поддерживает пользовательские токены на уровне протокола клиента Telegram. Поэтому вместо «токен в первом пакете» используется рабочая схема:

- один upstream `mtg` контейнер (например, `mtg:3128`);
- для каждого пользователя выделяется **свой внешний порт**;
- gateway проксирует `user_port -> mtg:3128`;
- отключение пользователя = закрытие его порта.

Такой подход совместим с Telegram-клиентами и дает простое администрирование.

## Состав

```text
mtproxy_gateway/
├─ gateway.py           # Multi-port TCP gateway (динамически открывает порты активных пользователей)
├─ db_setup.py          # SQLite + миграции
├─ generate_token.py    # CLI создание пользователя с автопортом
├─ admin_panel.py       # Flask web UI
├─ templates/
│  ├─ index.html
│  ├─ add_user.html
│  └─ user_detail.html
├─ requirements.txt
└─ users.db             # создается автоматически
```

## Установка

```bash
cd mtproxy_gateway
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
python db_setup.py
```

## Настройка окружения

Переменные для `admin_panel.py`:

- `PUBLIC_HOST` — внешний домен/IP, который видит клиент Telegram;
- `MTG_SECRET` — секрет вашего `mtg` (для генерации `tg://proxy` ссылок);
- `PORT_RANGE_START` / `PORT_RANGE_END` — диапазон автопортов пользователей.

Пример:

```bash
export PUBLIC_HOST=proxy.example.com
export MTG_SECRET=dd000000000000000000000000000000
export PORT_RANGE_START=11000
export PORT_RANGE_END=11999
```

## Запуск

### 1) Запуск upstream mtg в Docker

Пример (адаптируйте параметры под ваш конфиг):

```bash
docker run -d --name mtg \
  -p 3128:3128 \
  --restart unless-stopped \
  nineseconds/mtg:latest run --bind 0.0.0.0:3128 --secret dd000000000000000000000000000000
```

### 2) Запуск gateway

```bash
python gateway.py --listen-host 0.0.0.0 --upstream-host 127.0.0.1 --upstream-port 3128
```

Gateway будет периодически читать БД и открывать порты только для активных пользователей.

### 3) Запуск web-admin

```bash
python admin_panel.py
```

Web UI: `http://127.0.0.1:8000`

## Пользователи

### Через web-интерфейс

- создать пользователя (с авто-портом или вручную);
- включить/выключить доступ;
- удалить пользователя;
- получить `tg://proxy` ссылку.

### Через CLI

```bash
python generate_token.py alice
python generate_token.py bob --port 11555
```

## Важно

- Убедитесь, что диапазон пользовательских портов открыт в firewall/security-group.
- Не публикуйте админ-панель без авторизации (лучше держать за VPN/прокси).
- `token` в БД служебный (для учета); для Telegram-клиента ключевые поля — `PUBLIC_HOST`, `listen_port`, `MTG_SECRET`.
