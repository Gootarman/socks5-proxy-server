# MTProxy Gateway — ручная работа с БД (SQLite)

Этот документ описывает, как вручную смотреть и править базу `users.db` для `mtproxy_gateway`.

> Перед ручными изменениями рекомендуется сделать бэкап и временно остановить `admin` и `gateway`.

---

## 1) Где лежит БД

В docker-compose БД монтируется как volume и в контейнерах доступна по пути:

- `/data/users.db`

Локально (без docker) путь обычно:

- `mtproxy_gateway/users.db`

---

## 2) Быстрый бэкап

### В Docker

```bash
cd mtproxy_gateway
docker compose exec admin sh -lc 'cp /data/users.db /data/users.db.bak'
```

### Локально

```bash
cp mtproxy_gateway/users.db mtproxy_gateway/users.db.bak
```

---

## 3) Подключение к SQLite

### Через контейнер

```bash
docker compose exec admin sh -lc 'sqlite3 /data/users.db'
```

### Локально

```bash
sqlite3 mtproxy_gateway/users.db
```

Полезно сразу включить удобный вывод:

```sql
.headers on
.mode column
```

---

## 4) Структура таблиц

Показать таблицы:

```sql
.tables
```

Показать схему:

```sql
.schema users
.schema reserved_ports
.schema gateway_control
```

Назначение:

- `users` — текущие пользователи и активные порты;
- `reserved_ports` — уже когда-либо выданные порты (повторно не используются);
- `gateway_control` — служебный restart-сигнал для gateway.

---

## 5) Просмотр данных

### Пользователи

```sql
SELECT id, username, listen_port, is_active, created_at
FROM users
ORDER BY id DESC;
```

### Зарезервированные порты

```sql
SELECT port, reserved_at, reserved_by_user_id
FROM reserved_ports
ORDER BY port;
```

### Служебный restart token

```sql
SELECT * FROM gateway_control;
```

---

## 6) Ручное добавление пользователя

> Важно: порт должен быть уникальным и не должен существовать в `reserved_ports`.

Пример (с ручным токеном):

```sql
BEGIN;

INSERT INTO users (username, token, listen_port, is_active)
VALUES ('manual_user', 'manual_token_123', 11050, 1);

INSERT OR IGNORE INTO reserved_ports (port, reserved_by_user_id)
VALUES (11050, (SELECT id FROM users WHERE username = 'manual_user'));

COMMIT;
```

После добавления можно принудительно обновить gateway:

```sql
UPDATE gateway_control
SET restart_token = restart_token + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

---

## 7) Отключение пользователя (без удаления)

```sql
UPDATE users
SET is_active = 0
WHERE username = 'manual_user';

UPDATE gateway_control
SET restart_token = restart_token + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

---

## 8) Удаление пользователя

```sql
DELETE FROM users
WHERE username = 'manual_user';

UPDATE gateway_control
SET restart_token = restart_token + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = 1;
```

Обратите внимание:

- запись в `reserved_ports` обычно **не удаляется**, это защита от повторного использования порта.

---

## 9) Проверка консистентности

Проверить дубликаты портов в `users` (не должно быть строк):

```sql
SELECT listen_port, COUNT(*) c
FROM users
GROUP BY listen_port
HAVING c > 1;
```

Проверить пользователей без резервирования порта (не должно быть строк):

```sql
SELECT u.id, u.username, u.listen_port
FROM users u
LEFT JOIN reserved_ports r ON r.port = u.listen_port
WHERE r.port IS NULL;
```

---

## 10) Частые проблемы

### Пользователь удалён, а доступ ещё есть

1. Убедитесь, что пользователя нет в `users`.
2. Увеличьте `restart_token` вручную (см. выше).
3. Проверьте логи:

```bash
docker compose logs -f gateway
```

### Не получается добавить пользователя на порт

Скорее всего порт уже есть в `reserved_ports` (и это ожидаемое поведение).

```sql
SELECT * FROM reserved_ports WHERE port = 11050;
```

---

## 11) Полезные one-liner команды

Список пользователей из контейнера:

```bash
docker compose exec admin sh -lc "sqlite3 /data/users.db 'SELECT id,username,listen_port,is_active FROM users ORDER BY id DESC;'"
```

Деактивировать пользователя и перезапустить gateway-сигнал:

```bash
docker compose exec admin sh -lc "sqlite3 /data/users.db \"UPDATE users SET is_active=0 WHERE username='manual_user'; UPDATE gateway_control SET restart_token=restart_token+1, updated_at=CURRENT_TIMESTAMP WHERE id=1;\""
```
