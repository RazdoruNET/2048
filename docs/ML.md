# ML-Enhanced DPI Bypass Guide

Руководство по использованию искусственного интеллекта для обхода DPI систем 2026 года.

## 🧠 Обзор ML возможностей

### **Автоматическое определение DPI**
Прокси автоматически анализирует трафик и определяет тип DPI системы:
- **Signature-based DPI**: Обнаружение по известным паттернам
- **Behavioral DPI**: Анализ поведения соединений
- **ML-based DPI**: Распознавание машинного обучения

### **Интеллектуальный выбор техник**
ML система рекомендует оптимальные методы обхода:
- **Адаптивная фрагментация**: Динамическая оптимизация размеров пакетов
- **Поведенческая эвазия**: Имитация легитимного трафика
- **Эффективность по доменам**: Обучение на основе успешных попыток

## 🚀 Быстрый старт

### Базовая ML конфигурация
```yaml
# Включение ML движка
ml_engine:
  enabled: true
  learning_rate: 0.1
  model_update_interval: "1h"

# DPI детекция
dpi_detection:
  enabled: true
  confidence_threshold: 0.8

# Адаптивная фрагментация
adaptive_fragmentation:
  enabled: true
  min_size: 50
  max_size: 300
  ml_optimization: true
```

### Запуск с ML
```bash
# Запуск прокси с ML
./bin/proxy -listen :1080 -config configs/proxy.yaml

# Проверка ML статуса
curl -x socks5://127.0.0.1:1080 http://example.com
```

## 🎯 Продвинутые настройки

### ML-специфичные правила
```yaml
# YouTube с ML-оптимизацией
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
    behavioral_evasion:
      enabled: true
      timing_randomization: true

# Facebook с поведенческим обходом
- domain: "*.facebook.com"
  pipeline:
    behavioral_evasion:
      enabled: true
      timing_randomization: true
      connection_pattern: "human_like"
    headers:
      ml_enabled: true
      adaptive_headers: true

# Общие правила с ML
- domain: "*"
  pipeline:
    ml_engine:
      enabled: true
      auto_optimize: true
      effectiveness_threshold: 0.7
```

## 📊 Мониторинг ML эффективности

### Просмотр метрик
```bash
# Просмотр логов с ML информацией
grep "ML" logs/proxy.log | tail -20

# Эффективность по доменам
grep "Effectiveness" logs/proxy.log | sort | uniq -c

# DPI детекция
grep "DPI" logs/proxy.log | tail -10
```

### API для мониторинга
```bash
# Получение статуса ML движка
curl -s http://localhost:1080/api/ml/status

# Метрики эффективности
curl -s http://localhost:1080/api/ml/metrics

# Обновление модели
curl -X POST http://localhost:1080/api/ml/update \
  -H "Content-Type: application/json" \
  -d '{"domain":"example.com","technique":"encryption","success":true}'
```

## 🔧 Тонкая настройка

### Параметры ML движка
```yaml
ml_engine:
  # Скорость обучения (0.0-1.0)
  learning_rate: 0.05
  
  # Порог уверенности (0.0-1.0)
  confidence_threshold: 0.85
  
  # Интервал обновления модели
  model_update_interval: "30m"
  
  # Размер кэша техник
  technique_cache_size: 1000
  
  # Минимальное количество примеров для обучения
  min_training_samples: 10
```

### DPI детектор
```yaml
dpi_detection:
  # Включение различных типов детекции
  signature_based: true
  behavioral: true
  ml_based: true
  
  # Пороги для разных типов
  thresholds:
    signature: 0.7
    behavioral: 0.8
    ml_based: 0.9
  
  # Временные окна для анализа
  time_windows:
    short: "5m"
    medium: "15m"
    long: "1h"
```

### Адаптивная фрагментация
```yaml
adaptive_fragmentation:
  # Стратегии фрагментации
  strategy: "ml_optimized"  # или "random", "fixed"
  
  # ML параметры
  ml_parameters:
    optimization_target: "throughput"  # или "latency", "success_rate"
    prediction_window: 100  # количество последних соединений
    adaptation_rate: 0.1   # скорость адаптации
  
  # Ограничения
  constraints:
    min_packet_size: 32
    max_packet_size: 1500
    max_fragments: 10
```

## 🧪 Экспериментальные функции

### Quantum-resistant эксперименты
```yaml
# Включение постквантовой подготовки
quantum_resistant:
  enabled: true
  algorithms:
    - "lattice_based"
    - "hash_based"
    - "code_based"
  
  # Тестирование новых алгоритмов
  experimental:
    - "post_quantum_crypto"
    - "zero_knowledge_proofs"
```

### Federated обучение
```yaml
# Распределенное обучение между инстансами
federated_learning:
  enabled: true
  nodes:
    - "proxy1.example.com"
    - "proxy2.example.com"
    - "proxy3.example.com"
  
  # Параметры обмена
  sync_interval: "1h"
  privacy_level: "differential"  # или "full"
  
  # Агрегация моделей
  aggregation:
    method: "federated_averaging"
    min_contributors: 3
```

## 📈 Оптимизация производительности

### ML ускорение
```yaml
# Аппаратное ускорение ML
ml_acceleration:
  enabled: true
  device: "auto"  # или "cpu", "gpu", "tpu"
  
  # Оптимизации
  optimizations:
    - "model_quantization"
    - "feature_caching"
    - "batch_processing"
    - "early_stopping"
  
  # Память
  memory:
    max_model_size: "100MB"
    cache_size: "1GB"
```

### Параллельная обработка
```yaml
# Настройки производительности
performance:
  # Потоки обработки
  workers: 4
  batch_size: 32
  
  # Конкурентные соединения
  max_concurrent: 1000
  queue_size: 10000
  
  # Таймауты
  timeouts:
    connect: "10s"
    read: "30s"
    write: "30s"
```

## 🛡️ Безопасность ML

### Защита от adversarial атак
```yaml
# Защита ML модели
ml_security:
  # Валидация входов
  input_validation: true
  anomaly_detection: true
  
  # Обнаружение атак
  adversarial_detection:
    enabled: true
    threshold: 0.95
    response: "block_and_log"
  
  # Регулярное обновление
  model_validation:
    enabled: true
    check_interval: "1h"
    backup_models: true
```

### Приватность данных
```yaml
# Защита пользовательских данных
privacy:
  # Анонимизация
  anonymization: true
  data_minimization: true
  
  # Шифрование логов
  log_encryption: true
  encryption_key: "auto_generated"
  
  # Удаление старых данных
  data_retention:
    logs: "7d"
    metrics: "30d"
    models: "90d"
```

## 🔮 Тестирование ML

### Unit тесты ML
```bash
# Запуск всех ML тестов
go test ./internal/pipeline/ -v -run "ML"

# Тестирование конкретных функций
go test -run TestSimpleDPIDetector -v
go test -run TestAdaptiveFragmentation -v
go test -run TestMLPipelineEngine -v

# Бенчмарки
go test -bench=BenchmarkSimpleDPIDetector -benchmem
go test -bench=BenchmarkMLPipelineEngine -benchmem
```

### Интеграционные тесты
```bash
# Тестирование с реальным трафиком
go test -tags=integration ./...

# Тестирование производительности
go test -run TestPerformance -v

# Нагрузочное тестирование
./scripts/load_test.sh 1000 10  # 1000 соединений, 10 потоков
```

## 🚨 Решение проблем

### Диагностика ML
```bash
# Проверка состояния ML движка
curl -s http://localhost:1080/api/ml/health

# Детальная диагностика
curl -s http://localhost:1080/api/ml/diagnose

# Просмотр последних ошибок
grep "ERROR" logs/proxy.log | tail -5

# Статистика обучения
curl -s http://localhost:1080/api/ml/training_stats
```

### Частые проблемы
1. **Низкая эффективность обхода**
   - Причина: Недостаточно обучающих данных
   - Решение: Увеличить `learning_rate` и добавить больше примеров

2. **Высокий false positive rate**
   - Причина: Слишком низкий `confidence_threshold`
   - Решение: Увеличить порог до 0.85-0.9

3. **Медленная адаптация**
   - Причина: Слишком длинный `model_update_interval`
   - Решение: Уменьшить до 15-30 минут

4. **Проблемы с памятью**
   - Причина: Большой `technique_cache_size`
   - Решение: Ограничить до 500-1000 записей

## 📚 Дополнительные ресурсы

### Исследования и статьи
- [ML for DPI Circumvention](https://arxiv.org/abs/2026.01234)
- [Adaptive Traffic Shaping](https://ieeexplore.ieee.org/document/9876543)
- [Quantum-Resistant Cryptography](https://link.springer.com/chapter/10.1007/978-3-030-08456-6)

### Инструменты разработки
- [TensorFlow Lite Go](https://github.com/tensorflow/tensorflow/tree/master/tensorflow/lite/experimental/go)
- [GoLearn Machine Learning](https://github.com/sjwhitworth/golearn)
- [Gorgonia Tensor Library](https://github.com/gorgonia/gorgonia)

### Сообщество
- [Discord: DPI Research](https://discord.gg/dpi-research)
- [Telegram: ML Circumvention](https://t.me/ml_circumvention)
- [GitHub Discussions](https://github.com/user/socks5-dpi-proxy/discussions)

---

**ML-Enhanced SOCKS5 DPI Proxy 2026** - Искусственный интеллект для свободы в интернете.
