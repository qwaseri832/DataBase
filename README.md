In-Memory Key-Value Storage


Возможности:
 SET / GET / DEL   | Базовые операции с ключами |
 
 Партиционирование | FNV-1a хеш + 8 партиций для многопоточности |
 
 WAL               | Асинхронная пакетная запись (100 записей / 10 мс) |
 
 Репликация        | Master-Slave с синхронизацией сегментов |
 
 Graceful shutdown | Корректное завершение по SIGINT/SIGTERM |
 
 Конфигурация      | YAML файл со всеми настройками |




Запуск сервера:
go run cmd/server/main.go

Запуск CLI:
go run cmd/cli/main.go




Архитектура:
- TCP Server — принимает соединения, лимит клиентов, таймауты
- Parser — разбирает команды SET/GET/DEL, проверяет аргументы
- Storage — оркестрирует Engine, WAL и репликацию
- Engine — партиционированная хеш-таблица (FNV-1a, 8 партиций)
- WAL — Write-Ahead Log с батчингом и ротацией сегментов
- Replication — Master-Slave синхронизация через TCP

Клиент → TCP Server → Parser → Storage → Engine/WAL/Replication → Ответ




Тестирование:
Запуск всех тестов:
go test -v ./...

Покрытие кода
go test -cover ./...
                                    
