# Spider — key-value хранилище с WAL и репликацией

[![CI](https://github.com/qwaseri832/DataBase/actions/workflows/ci.yml/badge.svg)](https://github.com/qwaseri832/DataBase/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

In-memory key-value хранилище с журналом упреждающей записи (WAL),
репликацией master–slave и TCP-интерфейсом.

Учебный проект: цель — разобрать, как устроены долговечность, батчинг записи
на диск и потоковый сетевой протокол, а не конкурировать с Redis.

## Команды

| Команда | Формат | Ответ |
|---------|--------|-------|
| SET | `SET <key> <value>` | `[ok]` |
| GET | `GET <key>` | `[ok] <value>` или `[not found]` |
| DEL | `DEL <key>` | `[ok]` |

Регистр команды значения не имеет. Ключ и значение не могут содержать пробелов.

## Быстрый старт

```bash
git clone https://github.com/qwaseri832/DataBase.git
cd DataBase
make build
```

Без `make`:

```bash
go build -o bin/spider-server ./cmd/server
go build -o bin/spider-cli ./cmd/cli
```

Сервер:

```bash
./bin/spider-server -config config.yml
```

Без `-config` узел поднимается на `:4200` в памяти, без WAL и репликации.
Путь можно задать и переменной `CONFIG_FILE_NAME`.

Клиент:

```bash
./bin/spider-cli -addr 127.0.0.1:4200
```

```
[spider] > SET user Дмитрий
[ok]
[spider] > GET user
[ok] Дмитрий
[spider] > DEL user
[ok]
[spider] > GET user
[not found]
```

## Устройство

```
cmd/
  server/                точка входа сервера
  cli/                   интерактивный клиент
internal/
  bootstrap/             сборка компонентов и запуск
  config/                разбор YAML
  network/               TCP-сервер, клиент и фрейминг сообщений
  syncx/                 Future/Promise и семафор
  tools/                 разбор размеров, transaction id в контексте
  database/
    compute/             разбор команд
    filesystem/          файлы-сегменты WAL
    storage/
      engine/            хеш-таблица с партиционированием
      wal/               журнал упреждающей записи с батчингом
      replication/       master–slave
```

### Сетевой протокол

Сообщения передаются с 4-байтовым префиксом длины (big-endian):

```
+--------+--------------------+
| uint32 |      payload       |
+--------+--------------------+
```

TCP — поток байтов, а не последовательность сообщений: один `Read` может
вернуть половину запроса или сразу два подряд. Явные границы нужны и
клиентскому протоколу, и репликации, которая передаёт WAL-сегменты в
двоичном виде.

### Долговечность

`SET` и `DEL` возвращают `[ok]` только после того, как запись физически
попала в сегмент WAL. Записи копятся в батч и сбрасываются либо по
достижении `batch_size`, либо по `batch_timeout` — что наступит раньше.
Вызывающий получает `Future`, который разрешается после `fsync`.

При старте узел проигрывает все сегменты и продолжает нумерацию LSN с
последней восстановленной записи.

### Репликация

Реплика периодически спрашивает у мастера сегмент, следующий за её
последним, сохраняет его у себя и применяет к своему движку. Изменяющие
операции на реплике отклоняются с `read-only: mutable operation on slave`.

## Конфигурация

Пример — в [config.yml](config.yml). Неизвестные ключи считаются ошибкой:
опечатка в имени параметра не должна молча превращаться в значение по
умолчанию.

| Секция | Ключ | Значение по умолчанию |
|---|---|---|
| `engine` | `partitions` | 1 |
| `wal` | `batch_size` | 100 |
| `wal` | `batch_timeout` | 10ms |
| `wal` | `segment_max_size` | 10MB |
| `wal` | `directory` | ./data/wal |
| `server` | `addr` | :4200 |
| `server` | `max_clients` | без ограничения |
| `server` | `max_message_size` | 64MB |
| `replication` | `role` | без репликации |
| `logging` | `level` | info |

Секция целиком необязательна: без `wal` узел работает без журнала, без
`replication` — одиночным.

## Разработка

```bash
make test    # go test -race ./...
make lint    # go vet + gofmt -l
make fmt
```

Тесты `-race` здесь обязательны: часть кода — про синхронизацию (батчинг
WAL, Promise, семафор), и без детектора гонок регрессии в них не видно.

## Лицензия

[MIT](LICENSE)
