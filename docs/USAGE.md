# Использование SOCKS5 DPI Proxy с ML-оптимизацией

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

## ML-оптимизация и интеллектуальная маршрутизация

### Как работает ML-оптимизация

Прокси использует машинное обучение для автоматической оптимизации DPI обхода:

1. **Обнаружение типа DPI**: Автоматическое определение системы фильтрации (сигнатурная, поведенческая, ML-based)
2. **Предсказание эффективности**: ML-модели прогнозируют успешность техник обхода
3. **Адаптивная фрагментация**: Динамическая оптимизация размеров пакетов
4. **Обучение с подкреплением**: Система улучшается на основе результатов
5. **Интеллектуальный выбор техник**: Автоматический выбор оптимальных методов обхода

### Конфигурация правил

#### Файл: `configs/proxy.yaml`

```yaml
listen: ":1080"
log_level: "info"

# Глобальные настройки ML
ml_enabled: true
learning_enabled: true
model_update_interval: 3600  # Обновлять модель каждый час

rules:
  # Правило для доменов которые всегда доступны напрямую
  - domain: "*.github.com"
    pipeline: {}

  # ML-оптимизированное правило для соцсетей
  - domain: "*.facebook.com"
    pipeline:
      ml_optimized: true
      fragmentation:
        min_size: 64
        max_size: 128
        random: true
        adaptive: true  # ML-адаптивная фрагментация
      headers:
        random_headers: true
        modify_host: true
        add_junk_headers: true
        custom_headers:
          "X-Forwarded-For": "192.168.1.100"
          "X-Real-IP": "10.0.0.1"
        user_agents:
          - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
          - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
      encryption:
        type: "chacha20poly1305"  # Продвинутое шифрование
        key: "facebook_bypass_key_2026"
      protocol_mask:
        type: "https"
        https_noise: true
        timing_random: true  # Временная рандомизация

  # ML-правило для видеосервисов с адаптивной фрагментацией
  - domain: "*.youtube.com"
    pipeline:
      ml_optimized: true
      fragmentation:
        size: 100
        adaptive: true
        learning_rate: 0.1  # Скорость обучения
      protocol_mask:
        type: "tls"
        timing_random: true
        noise_level: 0.3

  # Правило с предсказанием эффективности
  - domain: "*.twitter.com"
    pipeline:
      ml_optimized: true
      effectiveness_threshold: 0.8  # Порог эффективности
      fallback_on_failure: true
      fragmentation:
        min_size: 50
        max_size: 200
        adaptive: true
      headers:
        random_headers: true
        modify_host: true
      encryption:
        type: "aes-gcm"
        key: "twitter_secure_key"

  # Правило по умолчанию с базовой ML-оптимизацией
  - domain: "*"
    pipeline:
      headers:
        random_headers: true
        ml_optimized_headers: true  # ML-оптимизация заголовков
```

### Типы модификаторов

#### 1. Fragmentation (Фрагментация)
Разбивает пакеты на мелкие части для обхода DPI.

```yaml
fragmentation:
  min_size: 50      # Минимальный размер фрагмента
  max_size: 200     # Максимальный размер фрагмента
  random: true       # Случайный размер фрагментов
  adaptive: true     # ML-адаптивная фрагментация
  learning_rate: 0.1 # Скорость обучения адаптации
  ml_optimized: true # Использовать ML для оптимизации
```

#### 2. Headers (Модификация заголовков)
Изменяет HTTP заголовки для маскировки запросов.

```yaml
headers:
  random_headers: true        # Добавлять случайные заголовки
  modify_host: true          # Модифицировать Host заголовок
  add_junk_headers: true     # Добавлять "мусорные" заголовки
  ml_optimized_headers: true # ML-оптимизация заголовков
  custom_headers:           # Пользовательские заголовки
    "X-Forwarded-For": "192.168.1.100"
    "X-Real-IP": "10.0.0.1"
  user_agents:              # Список User-Agent для ротации
    - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
    - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
```

#### 3. Encryption (Шифрование)
Шифрует трафик для обхода инспекции пакетов.

```yaml
encryption:
  type: "chacha20poly1305"  # Тип шифрования: xor, rc4, chacha20poly1305, aes-gcm
  key: "secure_ml_key_2026"   # Ключ шифрования
  ml_key_rotation: true      # ML-оптимизированная ротация ключей
```

#### 4. Protocol Mask (Маскировка протокола)
Маскирует трафик под другие протоколы.

```yaml
protocol_mask:
  type: "https"        # Тип маскировки: ssh, https, tls
  ssh_padding: true     # Добавлять SSH паддинг
  https_noise: true     # Добавлять HTTPS шум
  timing_random: true   # Временная рандомизация
  noise_level: 0.3      # Уровень шума (0.0-1.0)
  ml_optimized_noise: true # ML-оптимизация уровня шума
```

#### 5. ML Optimization (ML-оптимизация)
Настройки машинного обучения для правила.

```yaml
ml_optimized: true              # Включить ML-оптимизацию
learning_enabled: true          # Разрешить обучение
effectiveness_threshold: 0.8    # Порог эффективности
fallback_on_failure: true       # Fallback при неудаче
model_update_interval: 3600     # Интервал обновления модели (сек)
learning_rate: 0.1              # Скорость обучения
```

## Мониторинг и отладка

### Логи ML-решений

Прокси логирует ML-решения и DPI обнаружение:

```
2024/01/01 12:00:00 DPI detected: type=behavioral, confidence=0.85 for facebook.com:443
2024/01/01 12:00:01 ML selected optimal technique: adaptive_fragmentation (effectiveness=0.92)
2024/01/01 12:00:02 Learning updated: fragmentation effectiveness increased to 0.88
2024/01/01 12:00:03 Using ML-optimized pipeline for youtube.com:443
2024/01/01 12:00:04 Model updated with new training samples: 5 successful, 2 failed
```

### Проверка ML-метрик

```bash
# Статистика DPI обнаружения
grep "DPI detected" /var/log/proxy.log | wc -l

# Эффективность техник обхода
grep "effectiveness=" /var/log/proxy.log | awk '{print $NF}' | sort -n

# ML-обучение статистика
grep "Learning updated" /var/log/proxy.log | wc -l

# Оптимальные техники по доменам
grep "optimal technique" /var/log/proxy.log
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

### 2. Динамическое ML-переключение

Прокси автоматически переключается между режимами на основе ML:

- **Обнаружение DPI**: Автоматическое определение типа DPI системы
- **Адаптивные пайплайны**: ML-выбор оптимальных техник
- **Обучение в реальном времени**: Обновление моделей на основе результатов
- **Предсказание эффективности**: Прогнозирование успешности техник
- **Fallback логика**: Интеллектуальный возврат при ошибках

### 3. Мониторинг ML в реальном времени

```bash
# Следить за ML-решениями
tail -f /var/log/proxy.log | grep -E "(DPI detected|ML selected|Learning updated|optimal technique)"

# Статистика эффективности техник
watch -n 5 'grep "effectiveness=" /var/log/proxy.log | tail -10'

# Мониторинг обучения модели
watch -n 10 'grep "Model updated" /var/log/proxy.log'

# DPI типы статистика
grep "DPI detected" /var/log/proxy.log | awk '{print $4}' | sort | uniq -c
```

## Решение проблем

### Частые проблемы

1. **ML-компоненты не работают**
   ```bash
   # Проверить включена ли ML-оптимизация
   grep "ml_enabled" configs/proxy.yaml
   
   # Проверить логи ML
   grep "ML" /var/log/proxy.log
   ```

2. **Низкая эффективность обхода**
   ```bash
   # Проверить эффективность техник
   grep "effectiveness=" /var/log/proxy.log | tail -20
   
   # Сбросить ML-модель
   rm -rf /tmp/ml_model_cache/*
   ```

3. **Высокое потребление памяти**
   ```bash
   # Отключить тяжелые ML-компоненты
   # Установить learning_enabled: false в конфигурации
   ```

### Отладка ML

```bash
# Включить ML debug логирование
./bin/proxy -log_level debug -ml_debug true

# Проверить конкретный домен с ML-анализом
curl --socks5 127.0.0.1:1080 -v http://target-domain.com 2>&1 | grep -i ml

# Анализировать DPI тип
./bin/proxy -analyze_dpi facebook.com:443

# Экспорт ML-модели
./bin/proxy -export_ml_model /tmp/model.json
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

### Ожидаемые показатели с ML-оптимизацией

- **Прямые соединения**: 40-60% трафика (ML-оптимизированное определение)
- **Экономия задержки**: 15-60ms для доступных доменов
- **Пропускная способность**: 1Gbps+ при прямых соединениях
- **Потребление памяти**: < 150MB для 1000 соединений (с ML)
- **Успешность обхода**: До 95% для сложных DPI (с ML)
- **Адаптивное обучение**: Улучшение на 10-25% со временем

### Мониторинг ML-метрик

```bash
# Статистика DPI обнаружения
grep -c "DPI detected" /var/log/proxy.log

# Эффективность техник обхода
grep "effectiveness=" /var/log/proxy.log | awk -F'=' '{sum+=$NF; count++} END {print "Average effectiveness:", sum/count}'

# ML-обучение статистика
grep -c "Learning updated" /var/log/proxy.log

# Производительность с ML
awk '/Using ML-optimized/ {print $NF}' /var/log/proxy.log | sort -n

# Типы DPI статистика
grep "DPI detected" /var/log/proxy.log | awk '{type=$4; gsub(/,/,"",type); count[type]++} END {for (t in count) print t, count[t]}'
```

Прокси готов к использованию в продакшене с ML-интеллектуальной оптимизацией!
