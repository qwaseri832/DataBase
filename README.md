markdown
# 🕷️ Spider - Distributed Key-Value Store

[![Go Version](https://img.shields.io/badge/Go-1.26-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

**Spider** — это легковесное распределённое key-value хранилище с поддержкой WAL (Write-Ahead Log), репликацией Master-Slave и TCP-интерфейсом.

## ✨ Особенности

- 🚀 **Высокая производительность** — in-memory движок с партиционированием
- 💾 **Долговечность** — WAL гарантирует сохранность данных
- 🔄 **Репликация** — Master-Slave синхронизация через WAL-сегменты
- 📦 **Пакетная запись** — батчинг для оптимальной работы с диском
- 🔌 **Простой TCP-протокол** — легко интегрируется с любыми языками
- ⚡ **Потокобезопасность** — все операции безопасны для конкурентного доступа

## 🏗️ Архитектура
┌─────────┐ ┌─────────────┐ ┌─────────────────────────┐
│ Клиент │────▶│ TCP сервер │────▶│ Database │
└─────────┘ └─────────────┘ │ ┌─────────┬─────────┐ │
│ │ Парсер │ Storage │ │
│ └─────────┴─────────┘ │
│ │ │
│ ┌────┴────┐ │
│ ↓ ↓ │
│ Engine WAL │
│(in-memory) (диск) │
└─────────────────────────┘
│
┌─────────┴─────────┐
↓ ↓
┌──────────┐ ┌──────────┐
│ Master │◀─────▶│ Slave │
└──────────┘ └──────────┘

text

## 📋 Команды

| Команда | Формат | Ответ |
|---------|--------|-------|
| SET | `SET <key> <value>` | `[ok]` |
| GET | `GET <key>` | `[ok] <value>` или `[not found]` |
| DEL | `DEL <key>` | `[ok]` |

## 🚀 Быстрый старт

### Установка

```bash
git clone https://github.com/yourusername/spider.git
cd spider
go build -o spider-server ./cmd/server
go build -o spider-client ./cmd/client
Запуск сервера (Master)
bash
./spider-server --config config.yml
Пример конфигурации config.yml:

yaml
engine:
  partitions: 8

wal:
  batch_size: 100
  batch_timeout: 10ms
  segment_max_size: 10MB
  directory: ./data/wal

replication:
  role: master
  master_addr: "127.0.0.1:4400"
  sync_interval: 1s

server:
  addr: "127.0.0.1:4200"
  max_clients: 100
  read_buffer: 4KB
  idle_timeout: 5m

logging:
  level: info
  file: spider.log
Запуск Slave
yaml
replication:
  role: slave
  master_addr: "127.0.0.1:4400"
Клиент
bash
./spider-client -addr 127.0.0.1:4200
[spider] > SET name Spider
[ok]
[spider] > GET name
[ok] Spider
[spider] > DEL name
[ok]
📁 Структура проекта
text
spider/
├── cmd/
│   ├── server/          # Точка входа сервера
│   └── client/          # Точка входа клиента
├── internal/
│   ├── bootstrap/       # Сборка и запуск компонентов
│   ├── config/          # Парсинг конфигурации
│   ├── database/
│   │   ├── compute/     # Парсер команд (SET/GET/DEL)
│   │   ├── filesystem/  # Работа с файлами сегментов
│   │   └── storage/
│   │       ├── engine/  # In-memory хеш-таблица с партициями
│   │       ├── wal/     # Write-Ahead Log
│   │       └── replication/ # Master-Slave репликация
│   ├── network/         # TCP сервер и клиент
│   ├── syncx/           # Утилиты: Future, Promise, Semaphore
│   └── tools/           # Вспомогательные функции
├── config.yml           # Пример конфигурации
└── go.mod
🔧 Конфигурация
Параметр	Описание	По умолчанию
engine.partitions	Количество партиций для шардирования	1
wal.batch_size	Размер батча для WAL	100
wal.batch_timeout	Таймаут накопления батча	10ms
wal.segment_max_size	Максимальный размер сегмента	10MB
replication.role	master или slave	-
replication.master_addr	Адрес мастера (для слейва)	-
server.max_clients	Максимум одновременных клиентов	без ограничений
server.idle_timeout	Таймаут бездействия соединения	5m
🧪 Тестирование
bash
go test ./...
📊 Производительность
Партиционирование — ключи распределяются по партициям через FNV-1a хеш

Конкурентность — каждая партиция имеет свой RWMutex

Пакетная запись — WAL накапливает операции и пишет батчами

🛠️ Планы развития
Персистентное хранилище (LSM-дерево)

Аутентификация и TLS

Автоматический failover

Replication consensus (Raft)

Prometheus метрики

Docker образ

🤝 Вклад в проект
Приветствуются issues и pull requests!

Форкните репозиторий

Создайте ветку для фичи (git checkout -b feature/amazing)

Закоммитьте изменения (git commit -m 'Add amazing feature')

Запушьте (git push origin feature/amazing)

Откройте Pull Request

📝 Лицензия
MIT © 2024

⭐ Если проект полезен, поставьте звезду на GitHub!

text
