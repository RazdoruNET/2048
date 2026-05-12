# 🚀 Руководство по развертыванию SOCKS5 DPI Proxy

## 📋 Обзор

Это руководство описывает процесс развертывания SOCKS5 DPI Proxy в различных средах - от разработки до production.

**Версия**: 0.2.0-prod  
**Статус**: Production Ready с ограничениями  
**Требования**: Go 1.25+, Docker (опционально)

---

## 🔧 Требования к системе

### Минимальные требования:
- **CPU**: 2 ядра
- **RAM**: 2GB
- **Диск**: 10GB свободного места
- **ОС**: Linux (Ubuntu 20.04+, CentOS 8+, Debian 10+)
- **Сеть**: Стабильное интернет соединение

### Рекомендуемые требования:
- **CPU**: 4+ ядер
- **RAM**: 8GB+
- **Диск**: 50GB+ SSD
- **ОС**: Linux с ядром 5.4+
- **Сеть**: Высокоскоростное соединение

### Дополнительные зависимости:
- **Docker**: 20.10+ (для контейнеризации)
- **Docker Compose**: 2.0+ (для multi-service развертывания)
- **Node.js**: 18+ (только для разработки dashboard)

---

## 📦 Способы развертывания

### 1. Прямая установка (Binary)

#### Сборка из исходников:
```bash
# Клонирование репозитория
git clone https://github.com/your-repo/socks5-dpi-proxy.git
cd socks5-dpi-proxy

# Сборка бинарного файла
go build -o socks5-dpi-proxy ./cmd/main.go

# Проверка сборки
./socks5-dpi-proxy --version
```

#### Использование готовых бинарников:
```bash
# Скачивание последней версии
wget https://releases.example.com/socks5-dpi-proxy-v0.2.0-linux-amd64.tar.gz

# Распаковка
tar -xzf socks5-dpi-proxy-v0.2.0-linux-amd64.tar.gz

# Копирование бинарника
sudo cp socks5-dpi-proxy /usr/local/bin/
sudo chmod +x /usr/local/bin/socks5-dpi-proxy
```

#### Создание systemd сервиса:
```bash
# Создание файла сервиса
sudo nano /etc/systemd/system/socks5-dpi-proxy.service
```

```ini
[Unit]
Description=SOCKS5 DPI Proxy
After=network.target

[Service]
Type=simple
User=nobody
Group=nogroup
ExecStart=/usr/local/bin/socks5-dpi-proxy -config /etc/socks5-dpi-proxy/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
# Активация сервиса
sudo systemctl daemon-reload
sudo systemctl enable socks5-dpi-proxy
sudo systemctl start socks5-dpi-proxy

# Проверка статуса
sudo systemctl status socks5-dpi-proxy
```

---

### 2. Docker развертывание

#### Сборка Docker образа:
```bash
# Клонирование репозитория
git clone https://github.com/your-repo/socks5-dpi-proxy.git
cd socks5-dpi-proxy

# Сборка образа
docker build -t socks5-dpi-proxy:latest .

# Проверка образа
docker images | grep socks5-dpi-proxy
```

#### Запуск контейнера:
```bash
# Базовый запуск
docker run -d \
  --name socks5-dpi-proxy \
  -p 1080:1080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  socks5-dpi-proxy:latest

# Запуск с ограничениями ресурсов
docker run -d \
  --name socks5-dpi-proxy \
  --cpus="2.0" \
  --memory="4g" \
  -p 1080:1080 \
  -v $(pwd)/configs:/app/configs:ro \
  -v $(pwd)/logs:/app/logs \
  --restart unless-stopped \
  socks5-dpi-proxy:latest
```

#### Docker Compose развертывание:
```bash
# Использование готового docker-compose.yml
docker-compose up -d

# Проверка статуса
docker-compose ps

# Просмотр логов
docker-compose logs -f socks5-proxy

# Остановка
docker-compose down
```

---

## 🚨 Быстрый старт для продакшена

### 1. Запуск прокси

```bash
# Прямой запуск
./socks5-dpi-proxy -listen :1080 -config configs/proxy.yaml

# В фоновом режиме
nohup ./bin/proxy -listen :1080 -config configs/proxy.yaml > logs/proxy.log 2>&1 &

# Через Docker
docker-compose up -d
```

### 2. Проверка работоспособности

```bash
# Базовый тест
./scripts/test.sh

# Полный тест с оптимизацией
./scripts/test-availability.sh
```

## Конфигурация для продакшена

### Основные настройки (`configs/proxy.yaml`)

```yaml
listen: ":1080"
log_level: "info"

rules:
  # Домены доступные напрямую (без обхода)
  - domain: "*.github.com"
    pipeline: {}
  
  - domain: "*.stackoverflow.com"
    pipeline: {}

  # Соцсети (требуют обхода)
  - domain: "*.facebook.com"
    pipeline:
      fragmentation:
        min_size: 64
        max_size: 128
        random: true
      headers:
        random_headers: true
        modify_host: true
        add_junk_headers: true
      encryption:
        type: "xor"
        key: "facebook_bypass"

  - domain: "*.twitter.com"
    pipeline:
      headers:
        random_headers: true
        custom_headers:
          "X-Forwarded-For": "192.168.1.100"
      protocol_mask:
        type: "https"
        https_noise: true

  - domain: "*.youtube.com"
    pipeline:
      fragmentation:
        size: 100
      protocol_mask:
        type: "tls"

  # Правило по умолчанию
  - domain: "*"
    pipeline:
      headers:
        random_headers: true
```

## Настройка клиентов

### Firefox
1. Settings → Network Settings
2. Manual proxy configuration
3. SOCKS Host: 127.0.0.1, Port: 1080

### Chrome/Chromium
```bash
google-chrome --proxy-server="socks5://127.0.0.1:1080"
```

### Системные настройки
```bash
# Временно
export ALL_PROXY=socks5://127.0.0.1:1080

# Постоянно (добавить в ~/.bashrc или ~/.zshrc)
echo 'export ALL_PROXY=socks5://127.0.0.1:1080' >> ~/.bashrc
source ~/.bashrc
```

## Мониторинг

### Проверка статуса
```bash
# Проверить что прокси запущен
ps aux | grep "[b]in/proxy"

# Проверить порт
netstat -tlnp | grep :1080

# Проверить логи
tail -f logs/proxy.log
```

### Метрики производительности
```bash
# Статистика решений маршрутизации
grep -c "Using direct" logs/proxy.log
grep -c "Using DPI bypass" logs/proxy.log

# Текущие соединения
netstat -an | grep :1080 | wc -l
```

## Docker деплой

### Сборка образа
```bash
docker build -t socks5-dpi-proxy .
```

### Запуск контейнера
```bash
# Базовый запуск
docker run -d -p 1080:1080 -v $(pwd)/configs:/root/configs socks5-dpi-proxy

# С docker-compose
docker-compose up -d

# Проверка статуса
docker-compose ps
docker-compose logs proxy
```

### Docker Compose для продакшена
```yaml
version: '3.8'

services:
  socks5-proxy:
    build: .
    container_name: socks5-dpi-proxy
    ports:
      - "1080:1080"
    volumes:
      - ./configs:/root/configs:ro
      - ./logs:/root/logs
    environment:
      - LOG_LEVEL=info
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "nc", "-z", "localhost", "1080"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - proxy-network

networks:
  proxy-network:
    driver: bridge

volumes:
  logs:
    driver: local
```

## Безопасность

### Ограничение доступа
```yaml
# Добавить в configs/proxy.yaml
security:
  allowed_ips:
    - "192.168.1.0/24"
    - "10.0.0.0/8"
  auth_required: true
  auth_users:
    - username: "user1"
      password: "pass1"
```

### SSL/TLS
```bash
# Для HTTPS проксирования
./bin/proxy -listen :1080 -config configs/proxy.yaml -tls-cert server.crt -tls-key server.key
```

## Оптимизация

### Настройка кеширования
```yaml
# В configs/proxy.yaml
cache:
  availability_ttl: 300  # 5 минут
  max_entries: 10000
  cleanup_interval: 60  # 1 минута
```

### Производительность
```yaml
# В configs/proxy.yaml
performance:
  max_connections: 1000
  connection_timeout: 30
  read_buffer_size: 32768
  write_buffer_size: 32768
```

## Траблшутинг

### Общие проблемы
1. **Порт занят**: `bind: address already in use`
   ```bash
   # Решение
   sudo lsof -i :1080
   kill -9 <PID>
   ```

2. **Нет соединения**: `connection refused`
   ```bash
   # Проверить
   telnet 127.0.0.1 1080
   netstat -tlnp | grep :1080
   ```

3. **Медленная работа**: Высокая задержка
   ```bash
   # Решение
   # Упростить правила в configs/proxy.yaml
   # Отключить тяжелые модификаторы
   ```

### Логирование
```bash
# Включить debug режим
./bin/proxy -listen :1080 -config configs/proxy.yaml -log_level debug

# Отдельные логи
./bin/proxy -listen :1080 -config configs/proxy.yaml -log_file /var/log/proxy-debug.log
```

## Резервное копирование

### Автоматический бэкап
```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
cp -r configs/ backups/configs_$DATE/
cp -r logs/ backups/logs_$DATE/

# Добавить в cron
# 0 2 * * * /path/to/backup.sh
```

### Восстановление
```bash
# Восстановление из бэкапа
cp -r backups/configs_20240101_120000/ configs/
cp -r backups/logs_20240101_120000/ logs/
```

## Мониторинг в продакшене

### Prometheus метрики
```yaml
# Добавить в docker-compose.yml
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus
```

### Grafana дашборд
```yaml
# Добавить в docker-compose.yml
  grafana:
    image: grafana/grafana
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
```

## Апгрейды

### Обновление без даунтайма
```bash
#!/bin/bash
# rolling-update.sh

# Сборка новой версии
git pull origin main
./scripts/build.sh

# Плавное переключение
docker-compose up -d --no-deps proxy-new
sleep 30
docker-compose stop proxy-old
docker-compose rm proxy-old
```

### Blue-Green деплой
```bash
# Запуск второй инстанции
docker-compose -f docker-compose.yml -f docker-compose-blue.yml up -d

# Переключение трафика
# Изменить балансировщик или DNS
```

## Заключение

SOCKS5 DPI Proxy готов к продакшенному использованию со следующими характеристиками:

- 🚀 **Высокая производительность**: 1000+ одновременных соединений
- 🎯 **Интеллектуальная маршрутизация**: Автоматическая оптимизация
- 🔧 **Гибкая настройка**: YAML конфигурация правил
- 🐳 **Docker готов**: Контейнеризация для легкого деплоя
- 📊 **Мониторинг**: Логирование и метрики
- 🔒 **Безопасность**: Опциональная аутентификация

Система протестирована и подтверждена работоспособностью!
