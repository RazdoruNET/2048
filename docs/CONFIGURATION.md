# Configuration Guide

Руководство по конфигурации SOCKS5 DPI Proxy с ML-оптимизацией.

## 🚀 Быстрая настройка

### Минимальная конфигурация
```yaml
listen: ":1080"
log_level: "info"

rules:
  # Базовое правило с ML
  - domain: "*"
    pipeline:
      ml_engine:
        enabled: true
        auto_optimize: true
      encryption:
        type: "chacha20poly1305"
        key: "default_key_2026"
```

## 🧠 ML-оптимизированные правила

### YouTube с адаптивной фрагментацией
```yaml
- domain: "*.youtube.com"
  pipeline:
    adaptive_fragmentation:
      enabled: true
      min_size: 64
      max_size: 256
      ml_optimization: true
    encryption:
      type: "chacha20poly1305"
      key: "youtube_ml_key_2026"
    headers:
      ml_enabled: true
      adaptive_headers: true
```

### Социальные сети с поведенческой эвазией
```yaml
- domain: "*.facebook.com"
  pipeline:
    behavioral_evasion:
      enabled: true
      timing_randomization: true
      connection_pattern: "human_like"
    headers:
      ml_enabled: true
      adaptive_headers: true
    protocol_mask:
      type: "https"
      https_noise: true
```

### Стриминговые сервисы
```yaml
- domain: "*.twitch.tv"
  pipeline:
    adaptive_fragmentation:
      enabled: true
      strategy: "throughput_optimized"
    encryption:
      type: "aes-gcm"
      key: "streaming_key_2026"
    headers:
      ml_enabled: true
      content_type_obfuscation: true
```

## ⚙️ Продвинутые настройки

### ML движок
```yaml
ml_engine:
  enabled: true
  learning_rate: 0.05
  confidence_threshold: 0.85
  model_update_interval: "30m"
  
  # DPI детекция
  dpi_detection:
    enabled: true
    signature_based: true
    behavioral: true
    ml_based: true
    thresholds:
      signature: 0.7
      behavioral: 0.8
      ml_based: 0.9
  
  # Адаптивная фрагментация
  adaptive_fragmentation:
    enabled: true
    strategy: "ml_optimized"
    ml_parameters:
      optimization_target: "success_rate"
      prediction_window: 50
      adaptation_rate: 0.1
    constraints:
      min_packet_size: 32
      max_packet_size: 1024
      max_fragments: 8
```

### Quantum-resistant подготовка
```yaml
# Экспериментальные постквантовые алгоритмы
quantum_resistant:
  enabled: false  # Включать с осторожностью
  algorithms:
    - "lattice_based"
    - "hash_based"
    - "code_based"
  
  # Параметры
  key_sizes: [256, 512, 1024]  # бит
  security_level: "post_quantum"
```

## 🔒 Безопасность

### Криптографические настройки
```yaml
# Key rotation
security:
  key_rotation:
    enabled: true
    interval: "1h"
    algorithm: "chacha20poly1305"
    
  # Защита от атак
  anti_tampering:
    enabled: true
    integrity_check: "hmac-sha256"
    
  # Rate limiting
  rate_limiting:
    enabled: true
    requests_per_second: 100
    burst_size: 1000
    
  # IP фильтрация
  ip_filtering:
    enabled: true
    whitelist:
      - "192.168.1.0/24"
      - "10.0.0.0/8"
    blacklist:
      - "malicious.ip.range"
```

### Приватность
```yaml
privacy:
  # Анонимизация
  anonymization:
    enabled: true
    strip_user_agent: false
    custom_headers:
      "X-Forwarded-For": "proxy"
      "X-Real-IP": "random"
  
  # Шифрование логов
  log_encryption:
    enabled: false  # Включить для production
    algorithm: "aes-256-gcm"
    key_rotation: true
```

## 📊 Мониторинг и логирование

### Структура логов
```yaml
logging:
  # Уровни логирования
  levels:
    - "ERROR"   # Критические ошибки
    - "WARN"    # Предупреждения
    - "INFO"    # Общая информация
    - "DEBUG"   # Отладочная информация
    - "TRACE"   # Детальная трассировка
  
  # Формат вывода
  format: "json"  # или "text"
  
  # Вращение логов
  rotation:
    max_size: "100MB"
    max_files: 10
    compress: true
  
  # Специфические логи
  ml_metrics:
    enabled: true
    interval: "1m"
    retention: "7d"
  
  security_events:
    enabled: true
    alert_threshold: 5  # событий в минуту
```

### Метрики производительности
```yaml
metrics:
  # Системные метрики
  system:
    enabled: true
    interval: "30s"
    include:
      - "cpu_usage"
      - "memory_usage"
      - "goroutines"
      - "gc_stats"
  
  # Сетевые метрики
  network:
    enabled: true
    interval: "10s"
    include:
      - "connections_per_second"
      - "bytes_per_second"
      - "latency_percentiles"
      - "error_rate"
  
  # ML метрики
  ml:
    enabled: true
    interval: "1m"
    include:
      - "dpi_detection_accuracy"
      - "technique_effectiveness"
      - "model_confidence"
      - "learning_progress"
```

## 🌍 Географическая оптимизация

### Региональные правила
```yaml
# Конфигурация для разных регионов
regions:
  # Европа
  europe:
    default_techniques:
      - "encryption"
      - "adaptive_fragmentation"
    latency_optimization: true
    mtu: 1400
    
  # Азия
  asia:
    default_techniques:
      - "fragmentation"
      - "protocol_mask"
    packet_size_optimization: true
    mtu: 1200
    
  # Северная Америка
  north_america:
    default_techniques:
      - "encryption"
      - "headers"
      - "behavioral_evasion"
    connection_stability: true
    mtu: 1500
```

## 🔧 Отладка

### Отладочная конфигурация
```yaml
# Включение отладки
debug:
  enabled: false  # Включить для разработки
  
  # Детальное логирование ML
  ml_debug:
    enabled: false
    log_predictions: true
    log_features: true
    log_confidence: true
  
  # Трассировка пайплайнов
  pipeline_debug:
    enabled: false
    log_modifiers: true
    log_processing_time: true
  
  # Сетевая отладка
  network_debug:
    enabled: false
    log_packets: false
    log_connections: true
```

### Тестирование конфигурации
```yaml
# Тестовые сценарии
testing:
  # Тестовые домены
  test_domains:
    - "youtube.com"
    - "facebook.com"
    - "twitter.com"
    - "example.com"
  
  # Эмуляция DPI
  dpi_simulation:
    enabled: false
    types:
      - "signature_based"
      - "behavioral"
      - "ml_based"
  
  # Нагрузочное тестирование
  load_testing:
    enabled: false
    concurrent_connections: 100
    duration: "5m"
```

## 🚀 Production настройки

### Высокопроизводительность
```yaml
# Оптимизация для production
performance:
  # Go runtime
  gomaxprocs: 8  # Количество CPU ядер
  gc_percent: 20   # Целевое использование GC
  
  # Сетевые пулы
  connection_pool:
    max_connections: 10000
    idle_timeout: "30s"
    read_buffer_size: 8192
    write_buffer_size: 8192
  
  # ML оптимизация
  ml:
    cache_size: "1000"
    batch_size: 32
    parallel_processing: true
```

### Отказоустойчивость
```yaml
# High availability
high_availability:
  # Health checks
  health_check:
    enabled: true
    interval: "30s"
    timeout: "5s"
    retries: 3
    
  # Graceful shutdown
  graceful_shutdown:
    enabled: true
    timeout: "30s"
    drain_timeout: "5m"
    
  # Backup конфигурации
  backup:
    enabled: true
    interval: "1h"
    retention: "7d"
```

## 📝 Примеры использования

### Базовый прокси
```bash
# Запуск с базовой конфигурацией
./bin/proxy -listen :1080 -config configs/basic.yaml

# С логированием
./bin/proxy -listen :1080 -config configs/basic.yaml -log-level DEBUG

# С ML
./bin/proxy -listen :1080 -config configs/ml-enabled.yaml
```

### Docker развертывание
```bash
# Development
docker-compose -f docker-compose.dev.yml up -d

# Production
docker-compose -f docker-compose.prod.yml up -d

# С переменными окружения
ML_ENABLED=true DPI_DETECTION=true docker-compose up -d
```

### Клиентская настройка
```bash
# Firefox
export ALL_PROXY=socks5://127.0.0.1:1080
firefox

# Chrome
export ALL_PROXY=socks5://127.0.0.1:1080
google-chrome

# curl
curl -x socks5://127.0.0.1:1080 https://example.com

# Git
git config --global http.proxy socks5://127.0.0.1:1080
```

---

**Configuration Guide** - Полное руководство по настройке SOCKS5 DPI Proxy с ML-оптимизацией 2026 года.
