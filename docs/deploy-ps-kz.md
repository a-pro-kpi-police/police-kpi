# Деплой на PS.kz VPS

Проект лучше размещать на VPS/Cloud Server с Docker Engine: frontend собирается
в статический билд и отдаётся через Nginx-контейнер, backend работает на Go, а
PostgreSQL хранит данные в Docker volume.

## 1. Подготовить VPS

1. В PS.kz закажите VPS или Cloud Server с публичным IP.
2. Если в первоначальной настройке доступен вариант `Docker Engine`, выберите
   его. Иначе выберите Ubuntu 24.04 LTS и установите Docker вручную.
3. Откройте доступы `SSH`, `HTTP` и `HTTPS` в правилах безопасности.
4. Направьте домен `A`-записью на IP VPS.

После создания сервера подключитесь по SSH:

```bash
ssh ubuntu@<SERVER_IP>
sudo apt update
sudo apt upgrade -y
```

Установите инструменты для деплоя:

```bash
sudo apt install -y ca-certificates curl git nginx certbot python3-certbot-nginx
```

Если Docker не был установлен через первоначальную настройку PS.kz, установите
его по официальной инструкции Docker для Ubuntu:

```bash
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker
```

## 2. Загрузить проект

Вариант через Git:

```bash
sudo mkdir -p /opt/kpi-police
sudo chown "$USER":"$USER" /opt/kpi-police
git clone <REPO_URL> /opt/kpi-police
cd /opt/kpi-police
```

Если репозитория нет на сервере, можно загрузить архив проекта в `/opt/kpi-police`
через `scp` или SFTP.

## 3. Настроить секреты

```bash
cd /opt/kpi-police
cp .env.ps.kz.example .env
openssl rand -hex 32
openssl rand -hex 32
openssl rand -base64 24
openssl rand -base64 24
openssl rand -base64 24
```

В `.env` замените:

- `POSTGRES_PASSWORD` — первый сгенерированный hex-секрет.
- `KPI_SECRET` — второй сгенерированный hex-секрет.
- `KPI_ADMIN_PASSWORD` — пароль системного администратора `admin`.
- `KPI_HEAD_PASSWORD` — пароль начальника `head` при первом импорте.
- `KPI_IMPORT_DEFAULT_PASSWORD` — временный пароль импортированных сотрудников.
- `KPI_SEED_MODE=catalog_only` — для production: создавать только справочник
  KPI и аккаунт `admin`, без демо-сотрудников и месячного CSV-импорта.

Важно: начальные пароли применяются при первом создании базы. Если контейнеры
уже запускались, смена этих переменных не изменит существующие пароли в БД.

## 4. Запустить приложение

```bash
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f backend
```

Проверка с VPS:

```bash
curl -I http://127.0.0.1:8080
```

## 5. Подключить домен и HTTPS

Скопируйте пример Nginx-конфига и замените домен:

```bash
sudo cp deploy/ps-kz-nginx.conf.example /etc/nginx/sites-available/kpi-police
sudo nano /etc/nginx/sites-available/kpi-police
sudo ln -s /etc/nginx/sites-available/kpi-police /etc/nginx/sites-enabled/kpi-police
sudo nginx -t
sudo systemctl reload nginx
```

Выпустите Let's Encrypt сертификат:

```bash
sudo certbot --nginx -d example.kz -d www.example.kz
sudo certbot renew --dry-run
```

## 6. Обновление версии

```bash
cd /opt/kpi-police
git pull
docker compose -f docker-compose.prod.yml up -d --build
docker compose -f docker-compose.prod.yml logs -f --tail=100
```

## Удаление моковых seed-данных с production

Если production уже запускался с `KPI_SEED_MODE=monthly_import` или старым
demo seed, сначала в `.env` выставьте:

```bash
KPI_SEED_MODE=catalog_only
```

Затем перезапустите backend, чтобы новые моковые сотрудники больше не
создавались:

```bash
docker compose -f docker-compose.prod.yml up -d --build backend
```

Перед очисткой сделайте резервную копию:

```bash
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U kpi kpi > kpi_backup_before_mock_cleanup.sql
```

Сначала посмотрите, кого скрипт считает моковыми:

```bash
docker compose -f docker-compose.prod.yml exec -T postgres psql -U kpi -d kpi < deploy/preview-prod-mock-data.sql
```

Если список верный, удалите моковых сотрудников вместе с их KPI-записями,
целями, уведомлениями и сообщениями:

```bash
docker compose -f docker-compose.prod.yml exec -T postgres psql -U kpi -d kpi < deploy/cleanup-prod-mock-data.sql
```

## Резервные копии

Перед обновлениями и регулярно сохраняйте PostgreSQL и вложения:

```bash
docker compose -f docker-compose.prod.yml exec postgres pg_dump -U kpi kpi > kpi_backup.sql
docker run --rm -v kpi-police_backend_uploads:/data/uploads -v "$PWD":/backup alpine tar czf /backup/uploads_backup.tgz -C /data uploads
```
