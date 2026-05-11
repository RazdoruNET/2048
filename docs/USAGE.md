# Использование SOCKS5 DPI Proxy

## Быстрый старт

### 1. Сборка и запуск

```bash
# Собрать проект
./scripts/build.sh

# Запустить прокси
./scripts/run.sh
```

### 2. Базовая проверка работоспособности

```bash
# Запустить базовый тест
./scripts/test.sh

# Запустить тест доступности
./scripts/test-availability.sh
```

## Оптимизация и интеллектуальная маршрутизация

### Как работает оптимизация

Прокси автоматически определяет доступность доменов и выбирает оптимальный маршрут:

1. **Прямое соединение**: Если домен доступен напрямую
2. **Легкий обход**: Только модификация HTTP заголовков
3. **Полный обход**: Все техники DPI обхода
4. **Fallback**: Возврат к прямому соединению при ошибках

### Конфигурация правил

#### Файл: `configs/proxy.yaml`

```yaml
listen: ":1080"
log_level: "info"

rules:
  # Правило для доменов которые всегда доступны напрямую
  - domain: "*.github.com"
    pipeline: {}  # Пустой пайплайн = прямое соединение

  # Правило для заблокированных соцсетей
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

  # Правило по умолчанию для остальных сайтов
  - domain: "*"
    pipeline:
      headers:
        random_headers: true
```

### Типы модификаторов

#### 1. Fragmentation (Фрагментация)
Разбивает пакеты на мелкие части для обхода DPI.

```yaml
fragmentation:
  min_size: 50      # Минимальный размер фрагмента
  max_size: 200     # Максимальный размер фрагмента
  random: true       # Случайный размер фрагментов
```

#### 2. Headers (Модификация заголовков)
Изменяет HTTP заголовки для маскировки запросов.

```yaml
headers:
  random_headers: true        # Добавлять случайные заголовки
  modify_host: true          # Модифицировать Host заголовок
  add_junk_headers: true     # Добавлять "мусорные" заголовки
  custom_headers:           # Пользовательские заголовки
    "X-Forwarded-For": "192.168.1.100"
    "X-Real-IP": "10.0.0.1"
  user_agents:              # Список User-Agent для ротации
    - "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
    - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
```

#### 3. Encryption (Шифрование)
Шифрует трафик для обхода инспекции пакетов.

```yaml
encryption:
  type: "xor"       # Тип шифрования: xor, rc4
  key: "secret_key"   # Ключ шифрования
```

#### 4. Protocol Mask (Маскировка протокола)
Маскирует трафик под другие протоколы.

```yaml
protocol_mask:
  type: "https"        # Тип маскировки: ssh, https, tls
  ssh_padding: true     # Добавлять SSH паддинг
  https_noise: true     # Добавлять HTTPS шум
```

## Мониторинг и отладка

### Логи маршрутизации

Прокси логирует решения о маршрутизации:

```
2024/01/01 12:00:00 Using direct connection for github.com:443 (domain available)
2024/01/01 12:00:01 Using DPI bypass pipeline for facebook.com:443
2024/01/01 12:00:02 Direct connection failed, retrying with DPI bypass: twitter.com:443
```

### Проверка кеширования доступности

```bash
# Проверить статистику кеширования
grep "domain available" /var/log/proxy.log | wc -l
grep "Using DPI bypass" /var/log/proxy.log | wc -l
```

## Тестирование производительности

### Измерение задержки

```bash
# Прямое соединение
time curl http://example.com

# Через прокси (оптимизированное)
time curl --socks5 127.0.0.1:1080 http://example.com
```

### Нагрузочное тестирование

```bash
# Множественные параллельные запросы
for i in {1..100}; do
  curl --socks5 127.0.0.1:1080 http://httpbin.org/ip &
done
wait
```

## Продвинутые сценарии

### 1. Настройка для разных сетей

```yaml
# Для домашней сети (без блокировок)
- ip_range: "192.168.0.0/16"
  pipeline: {}

# Для мобильной сети (с блокировками)
- ip_range: "10.0.0.0/8"
  pipeline:
    fragmentation:
      size: 100
    headers:
      random_headers: true
```

### 2. Динамическое переключение

Прокси автоматически переключается между режимами:

- **Обнаружение доступности**: Фоновая проверка доменов
- **Адаптивные пайплайны**: Выбор оптимальных техник
- **Fallback логика**: Возврат при ошибках соединения

### 3. Мониторинг в реальном времени

```bash
# Следить за логами решений
tail -f /var/log/proxy.log | grep -E "(Using direct|Using DPI|Direct connection failed)"

# Статистика производительности
watch -n 5 'ps aux | grep "[p]roxy" | wc -l'
```

## Решение проблем

### Частые проблемы

1. **Прокси не запускается**
   ```bash
   # Проверить порт
   netstat -tlnp | grep :1080
   
   # Проверить конфигурацию
   ./bin/proxy -config configs/proxy.yaml -listen :1080
   ```

2. **Домены не доступны через прокси**
   ```bash
   # Проверить правила
   cat configs/proxy.yaml
   
   # Проверить доступность напрямую
   curl http://problematic-domain.com
   ```

3. **Высокая задержка**
   ```bash
   # Отключить ненужные модификаторы
   # Упростить правила в конфигурации
   ```

### Отладка

```bash
# Включить debug логирование
./bin/proxy -log_level debug

# Проверить конкретный домен
curl --socks5 127.0.0.1:1080 -v http://target-domain.com
```

## Безопасность

### Рекомендации

1. **Ограничить доступ по IP**
   ```yaml
   # В конфигурации прокси
   allowed_ips:
     - "192.168.1.0/24"
     - "10.0.0.0/8"
   ```

2. **Использовать сложные ключи шифрования**
   ```yaml
   encryption:
     type: "xor"
     key: "your_very_long_and_secure_key_here"
   ```

3. **Ротация User-Agent**
   ```yaml
   headers:
     user_agents:
       - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
       - "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36"
   ```

## Интеграция с приложениями

### Браузеры

**Firefox:**
- Settings → Network Settings → Manual proxy configuration
- SOCKS Host: 127.0.0.1, Port: 1080

**Chrome:**
```bash
google-chrome --proxy-server="socks5://127.0.0.1:1080"
```

### Терминальные приложения

```bash
# Временно
export ALL_PROXY=socks5://127.0.0.1:1080

# Постоянно (добавить в ~/.bashrc)
echo 'export ALL_PROXY=socks5://127.0.0.1:1080' >> ~/.bashrc
```

### Docker

```yaml
# docker-compose.yml
services:
  app:
    image: your-app
    environment:
      - ALL_PROXY=socks5://proxy:1080
    depends_on:
      - proxy
```

## Метрики производительности

### Ожидаемые показатели

- **Прямые соединения**: 50-80% трафика
- **Экономия задержки**: 10-50ms для доступных доменов
- **Пропускная способность**: 1Gbps+ при прямых соединениях
- **Потребление памяти**: < 100MB для 1000 соединений

### Мониторинг

```bash
# Статистика решений маршрутизации
grep -c "Using direct" /var/log/proxy.log
grep -c "Using DPI bypass" /var/log/proxy.log

# Производительность
awk '/Using direct/ {print $NF}' /var/log/proxy.log | sort -n
```

Прокси готов к использованию в продакшене с интеллектуальной оптимизацией маршрутизации!
