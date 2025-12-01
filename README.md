# Website Status Checker

Веб-сервер на Go для проверки доступности сайтов. Возвращает статус сайтов (доступен/не доступен) с использованием кэширования.

## 🚀 Быстрый старт

### Через Docker (рекомендуется)
```bash
# Сборка и запуск
docker build -t website-checker .
docker run -p 8080:8080 website-checker

# Или с готовым образом
docker run -p 8080:8080 yourname/website-checker
```

### Напрямую
```bash
# Сборка
go build -o website-checker .

# Запуск
./website-checker
```

## ⚙️ Конфигурация

Создайте `config.yaml`:
```yaml
server:
  address: "localhost"
  port: 8080

log:
  level: 0
  file: "./app.log"

storage:
  links_size: 1000
  cache_size: 800
```

## 📡 Использование

### Проверить сайты
```bash
curl -X POST "http://localhost:8080/links" \
  -H "Content-Type: application/json" \
  -d '["google.com", "github.com"]'
```

## ✨ Особенности

- **Кэширование LRU** - результаты проверок кэшируются
- **Валидация кэша** - автоматическое обновление устаревших данных
- **Гибкая настройка** - конфигурация через YAML-файл
- **Docker поддержка** - готовые образы для развертывания
- **Службы ОС** - запуск как служба Windows/Linux
- **Настраиваемое логирование** - разные уровни детализации

## 🐳 Docker

```bash
# Сборка
docker-compose up -d
```

## 🔧 Как системная служба

### Linux
```bash
sudo cp website-checker.service /etc/systemd/system/
sudo systemctl enable website-checker
sudo systemctl start website-checker
```

### Windows
```cmd
sc create link_verifier binPath= "C:\path\to\link_verifier.exe"
sc start link_verifier
```