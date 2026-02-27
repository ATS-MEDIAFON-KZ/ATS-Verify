# ATS-Verify Deployment Guide

Данная инструкция предназначена для DevOps-инженеров и системных администраторов для безопасного и правильного развертывания проекта ATS-Verify на боевом (Production) сервере из репозитория GitHub.

## Архитектура
Проект полностью контейнеризирован и состоит из следующих микросервисов, управляемых через Docker Compose:
* **ATS-Verify App** (Golang + React SPA) — Основной бэкенд и раздача скомпилированного фронтенда.
* **PostgreSQL 16** (`ats-verify-db`) — Основная база данных.
* **PgAdmin** (`ats-verify-pgadmin`) — Опциональная панель управления БД.

---

## Требования к серверу (Prerequisites)
1. **ОС**: Любой дистрибутив Linux (Ubuntu 22.04+ или Debian 12+ рекомендуются).
2. ОЗУ: минимум 4 ГБ (оптимально 8 ГБ, так как выполняется аналитика больших CSV-файлов).
3. Установленный **Git**.
4. Установленный **Docker Compose V2** (команда `docker compose` должна работать).

---

## Инструкция по развертыванию (Installation)

### 1. Клонирование репозитория
Осуществите вход на сервер (по SSH) и клонируйте `main` ветку из GitHub-репозитория в целевую директорию:

```bash
git clone https://github.com/aldaniyazov/ATS-Verify.git
cd ATS-Verify
```

### 2. Конфигурация переменных среды (.env)
Архитектура требует строгой конфигурации переменных окружения перед запуском. В корне проекта создайте файл `.env`:

```bash
nano .env
```

Вставьте и заполните следующие секреты. **КРИТИЧНО:** Обязательно замените пароли по умолчанию на надежные криптографические ключи!

```env
# === PostgreSQL ===
POSTGRES_DB=ats_verify
POSTGRES_USER=ats_admin
POSTGRES_PASSWORD=<ВАШ_НАДЕЖНЫЙ_ПАРОЛЬ_БД>
POSTGRES_PORT=5432

# === PgAdmin ===
PGADMIN_EMAIL=admin@ats-verify.app
PGADMIN_PASSWORD=<ВАШ_ПАРОЛЬ_ДЛЯ_PGADMIN>
PGADMIN_PORT=5050

# === Application ===
APP_PORT=80
JWT_SECRET=<УНИКАЛЬНЫЙ_ПРОИЗВОЛЬНЫЙ_ХЕШ_ДЛЯ_ТОКЕНОВ>
JWT_EXPIRATION_HOURS=24

# === External APIs ===
KAZPOST_API_KEY=<ПРОМ_КЛЮЧ_КАЗПОЧТЫ>
CDEK_CLIENT_ID=<CDEK_ID>
CDEK_CLIENT_SECRET=<CDEK_SECRET>
```

Сохраните файл (Ctrl+O, Enter, Ctrl+X).

### 3. Бильд и Запуск контейнеров
Запустите оркестрацию Docker Compose в фоновом режиме. Флаг `--build` обязателен при первом запуске, чтобы он скомпилировал TypeScript-фронтенд и Go-бэкенд.

```bash
docker compose up -d --build
```

### 4. Верификация запуска
Убедитесь, что все три контейнера перешли в статус `Up (healthy)`.

```bash
docker compose ps
```

Должно появиться:
* `ats-verify-db` (healthy)
* `ats-verify-app` (up)
* `ats-verify-pgadmin` (up)

Вы также можете проверить логи основного приложения на предмет успешного подключения к БД:
```bash
docker compose logs -f app
```

Ожидаемые логи:
```text
connected to PostgreSQL
ATS-Verify server starting on :8080
```

---

## Настройка базы данных
Таблицы БД (схема, триггеры, расширения `uuid-ossp`) разворачиваются автоматически при первом поднятии контейнера благодаря файлу `database/schema.sql`, который проброшен в `/docker-entrypoint-initdb.d/`.

В коде есть авто-сидер (seeder), который создаст дефолтных пользователей при совершенно пустой базе.
* Дефолтный логин администратора: `admin`
* Пароль по умолчанию: `admin`

**Внимание DevOps:** Сразу после развертывания авторизуйтесь в интерфейсе под админом и **смените пароль администратора** в целях безопасности!

---

## Обратный прокси (Reverse Proxy / SSL)
Само приложение поднимается на порту, указанном в `APP_PORT` (например, 8080 или 80). Рекомендуется закрыть его за Nginx или Traefik для терминации SSL-сертификатов (HTTPS) и настройки доменного имени.

Пример конфига Nginx:
```nginx
server {
    listen 80;
    server_name verify.yourdomain.kz;

    # Увеличиваем лимит размера загружаемого файла (для CSV отчетов)
    client_max_body_size 50M;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Обновление приложения (CI/CD)
Для доставки будущих релизов на сервер выполняйте следующую цепочку скриптов:

```bash
cd ATS-Verify
git pull origin main
docker compose up -d --build
```
Downtime составит всего пару секунд, так как Docker сначала скомпилирует новый образ, а затем моментально заменит контейнер.
