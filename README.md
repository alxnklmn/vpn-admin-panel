# 🔧 VPN Admin Panel

Веб-админка для управления VPN ботом от Jolymmiels с расширенными функциями управления.

![GitHub release (latest by date)](https://img.shields.io/github/v/release/alxnklmn/vpn-admin-panel)
![GitHub](https://img.shields.io/github/license/alxnklmn/vpn-admin-panel)
![GitHub stars](https://img.shields.io/github/stars/alxnklmn/vpn-admin-panel)

## ✨ Функции

### 📢 Массовая рассылка
- Отправка сообщений всем пользователям бота
- Поддержка HTML разметки
- Статистика отправленных/неудачных сообщений

### 📋 Просмотр логов
- Мониторинг логов контейнера в реальном времени
- Настраиваемое количество строк (50-500)
- Автообновление логов

### ✏️ Редактирование переводов
- Веб-интерфейс для редактирования текстов бота
- Поддержка русского (ru.json) и английского (en.json) языков
- Автоматическое сохранение с перезапуском

## 🚀 Установка и запуск

### Предварительные требования
- Docker и Docker Compose
- Работающая сеть `remnawave-network`
- Основной VPN бот Remnawave

### Запуск

```bash
# Клонирование репозитория
git clone https://github.com/alxnklmn/vpn-admin-panel.git
cd vpn-admin-panel

# Сборка и запуск контейнера
docker compose up -d --build
```

### Получение данных для входа

```bash
# Просмотр логов для получения логина и пароля
docker logs vpn-admin-server
```

### Доступ к админке

Откройте браузер и перейдите по адресу:
```
http://localhost:8081
```

## 🔧 Конфигурация

### Переменные окружения (.env)

```bash
# Токен Telegram бота
TELEGRAM_TOKEN=your_bot_token_here

# Строка подключения к PostgreSQL
DATABASE_URL=postgres://postgres:postgres@remnawave-telegram-shop-db:5432/postgres?sslmode=disable
```

### Структура проекта

```
vpn-admin-panel/
├── main.go                 # Основной файл сервера
├── docker-compose.yml      # Конфигурация Docker
├── Dockerfile             # Образ для сборки
├── static/                # Статические файлы
│   ├── css/admin.css      # Стили интерфейса
│   └── js/admin.js        # JavaScript функциональность
├── templates/             # HTML шаблоны
│   └── admin.html         # Главная страница админки
└── .env.example          # Пример переменных окружения
```

## 🔗 Интеграция

### Сеть Docker
Админ-панель подключается к существующей сети `remnawave-network` и использует ту же базу данных, что и основной бот.

### Монтирование переводов
Директория `translations` автоматически монтируется из основного проекта для редактирования файлов переводов.

## 🔒 Безопасность

- Генерация случайных логина и пароля при каждом запуске
- HTTP-only cookies для сессий
- Аутентификация для всех AJAX запросов
- Безопасный доступ к Docker API

## 📊 API Endpoints

| Endpoint | Метод | Описание |
|----------|--------|----------|
| `/admin/broadcast` | POST | Массовая рассылка |
| `/admin/logs` | GET | Получение логов |
| `/admin/translations` | GET | Получение переводов |
| `/admin/translations/update` | POST | Обновление переводов |
| `/admin/restart-bot` | POST | Перезапуск основного бота |



## 📝 Changelog

### v0.3 (2025-09-07)
- ✨ Добавлено редактирование переводов через веб-интерфейс
- 🔄 Автоматический перезапуск основного бота
- 📁 Монтирование директории translations
- 🔒 Улучшенная аутентификация

### v0.1 (2025-08-05)
- 📢 Массовая рассылка
- 📋 Просмотр логов
- 🔐 Базовая аутентификация

## 🤝 Связанные проекты

- [Remnawave Telegram Shop](https://github.com/Jolymmiles/remnawave-telegram-shop) - Основной VPN бот

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. См. файл [LICENSE](LICENSE) для подробностей.

## 🛠️ Поддержка

Если у вас возникли вопросы или проблемы, создайте [issue](https://github.com/alxnklmn/vpn-admin-panel/issues) в репозитории.

---
Made with ❤️ for the VPN community
