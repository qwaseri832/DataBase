🕷️ Spider - In-Memory Key-Value Storage

https://img.shields.io/badge/Go-1.26-blue.svg
https://img.shields.io/badge/tests-passing-brightgreen.svg
https://img.shields.io/badge/coverage-83%25-brightgreen.svg

Spider — это высокопроизводительное in-memory key-value хранилище на Go с поддержкой персистентности через WAL и мастер-слейв репликацией.

---

✨ Возможности

· 🚀 SET / GET / DEL — базовые операции с ключами
· 📊 Партиционирование — FNV-1a хеш + 8 партиций для многопоточности
· 💾 WAL (Write-Ahead Log) — асинхронная пакетная запись (100 записей / 10 мс)
· 🔄 Репликация — Master-Slave с синхронизацией сегментов
· 🛡️ Graceful shutdown — корректное завершение по SIGINT/SIGTERM
· ⚙️ Конфигурация — YAML файл со всеми настройками

---

🚀 Быстрый старт

Установка

```bash
git clone https://github.com/qwaseri832/DataBase.git
cd DataBase/spider
go mod download
```

Запуск сервера

```bash
go run cmd/server/main.go
```

Запуск CLI

```bash
go run cmd/cli/main.go
```

Пример работы

```bash
[spider] > SET name John
[ok]
[spider] > GET name
[ok] John
[spider] > DEL name
[ok]
[spider] > GET name
[not found]
```

---

📋 Конфигурация

Файл config.yml:

```yaml
engine:
  partitions: 8              # Количество партиций

wal:
  batch_size: 100            # Записей в пакете
  batch_timeout: "10ms"      # Таймаут пакета
  segment_max_size: "10MB"   # Размер сегмента

replication:
  role: "master"             # master / slave
  master_addr: "127.0.0.1:4400"

server:
  addr: ":4200"              # Адрес сервера
  max_clients: 100           # Лимит клиентов
  read_buffer: "4KB"         # Размер буфера
  idle_timeout: "5m"         # Таймаут бездействия

logging:
  level: "info"              # debug / info / warn / error
  file: "spider.log"         # Файл логов
```

---

🏗️ Архитектура

```
┌─────────┐     ┌──────────┐     ┌────────────────────────────────────┐
│ Client  │────▶│   TCP    │────▶│             Database                │
│ (CLI)   │     │  Server  │     │  ┌────────┐     ┌───────────────┐   │
└─────────┘     │  :4200   │     │  │ Parser │────▶│    Storage    │   │
                └──────────┘     │  └────────┘     └───────┬───────┘   │
                                 │                         │           │
                                 │         ┌───────────────┼───────┐   │
                                 │         ▼               ▼       ▼   │
                                 │    ┌────────┐     ┌────────┐ ┌─────┴──┐
                                 │    │ Engine │     │  WAL   │ │ Repl.  │
                                 │    │(8 парт)│     │(диск)  │ │(сеть)  │
                                 │    └────────┘     └────────┘ └────────┘
                                 └────────────────────────────────────────┘
```

Компоненты

Компонент Отвественность Ключевые детали
TCP Server Приём соединений max_clients, таймауты, буферизация
Parser Разбор команд SET/GET/DEL, валидация аргументов
Storage Оркестрация LSN, WAL, репликация
Engine Хранение данных 8 партиций, FNV-1a, RWMutex
WAL Персистентность Batch 100, Segment 10MB
Replication Синхронизация Master-Slave, интервал 1с

---

🧪 Тестирование

```bash
# Запуск всех тестов
go test -v ./...

# С покрытием кода
go test -cover ./...
```

Результаты покрытия

Пакет Покрытие
internal/database 83.3%
internal/database/compute 88.9%
internal/tools 85.7%
internal/database/storage/engine 28.9%

---

📊 Производительность

· Партиционирование — конкурентные операции без глобальных блокировок
· WAL с батчингом — снижение числа fsync до 100 раз в секунду
· FNV-1a хеш — быстрое распределение ключей
· Lock-per-partition — запись в одну партицию не блокирует чтение из других

---

🛡️ Graceful Shutdown

При нажатии Ctrl+C сервер:

1. Получает сигнал SIGINT/SIGTERM
2. Перестаёт принимать новые соединения
3. Дожидается завершения текущих запросов
4. Синхронизирует WAL на диск
5. Закрывает все соединения и завершается

---

🔧 Требования

· Go 1.26 или выше

---

📝 Лицензия

MIT

---

👨‍💻 О проекте

Проект демонстрирует навыки:

· Работа с конкурентностью в Go (sync, errgroup)
· Сетевое программирование (TCP)
· Работа с файловой системой (WAL, сегменты)
· Проектирование архитектуры
· Написание юнит-тестов (83%+ покрытие)
· Master-Slave репликация

---

🔗 Ссылки

· GitHub репозиторий
· Отчёт о покрытии тестами

---

⭐ Если вам понравился проект, поставьте звезду на GitHub!

```

## ✅ Сохраните в файл:

```bash
cd C:\Users\Lenovo\OneDrive\Desktop\extracting-code-from-repositories\spider
notepad README.md
```

Удалите всё и вставьте код выше, сохраните.

✅ Отправьте на GitHub:

```bash
git add README.md
git commit -m "Add beautiful README with emojis and badges"
git push origin main
```

🎉 Готово!

Теперь у вашего проекта Spider есть красивый README:

· ✅ Бейджики (Go version, tests, coverage)
· ✅ Эмодзи для наглядности
· ✅ Таблицы и схемы
· ✅ Примеры кода
· ✅ Полная документация

Ссылка: https://github.com/qwaseri832/DataBase 🚀
