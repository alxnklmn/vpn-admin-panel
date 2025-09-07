package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

type Customer struct {
	ID               int64      `json:"id"`
	TelegramID       int64      `json:"telegram_id"`
	ExpireAt         *time.Time `json:"expire_at"`
	CreatedAt        time.Time  `json:"created_at"`
	SubscriptionLink *string    `json:"subscription_link"`
	Language         string     `json:"language"`
}

type BroadcastRequest struct {
	Message string `json:"message"`
}

type BroadcastResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Sent    int    `json:"sent"`
	Failed  int    `json:"failed"`
}

type LogsResponse struct {
	Success bool   `json:"success"`
	Logs    string `json:"logs"`
	Error   string `json:"error,omitempty"`
}

type TranslationResponse struct {
	Success      bool                       `json:"success"`
	Translations map[string]map[string]string `json:"translations,omitempty"`
	Error        string                     `json:"error,omitempty"`
}

type UpdateTranslationRequest struct {
	Language string            `json:"language"`
	Data     map[string]string `json:"data"`
}

type RestartBotResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

type Server struct {
	db           *pgxpool.Pool
	adminUser    string
	adminPass    string
}

func main() {
	// Загружаем переменные окружения
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Генерируем случайные логин и пароль
	adminUser := generateRandomString(8)
	adminPass := generateRandomString(12)
	
	log.Printf("🔐 Admin Panel Credentials:")
	log.Printf("👤 Username: %s", adminUser)
	log.Printf("🔑 Password: %s", adminPass)
	log.Printf("🌐 Access URL: http://localhost:8081")

	// Подключаемся к БД
	dbURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	db, err := pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	server := &Server{
		db:        db,
		adminUser: adminUser,
		adminPass: adminPass,
	}

	// Настраиваем роуты
	mux := http.NewServeMux()
	
	// Статические файлы
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	// API endpoints
	mux.HandleFunc("/admin/broadcast", server.broadcastHandler)
	mux.HandleFunc("/admin/logs", server.logsHandler)
	mux.HandleFunc("/admin/translations", server.translationsHandler)
	mux.HandleFunc("/admin/translations/update", server.updateTranslationHandler)
	mux.HandleFunc("/admin/restart-bot", server.restartBotHandler)
	
	// Логин
	mux.HandleFunc("/login", server.loginHandler)
	
	// Главная страница
	mux.HandleFunc("/", server.indexHandler)

	// Настраиваем сервер
	srv := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan
		
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Printf("Admin server starting on port 8081...")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	// Обрабатываем POST запрос для логина
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		
		if username == s.adminUser && password == s.adminPass {
			// Устанавливаем куки
			http.SetCookie(w, &http.Cookie{
				Name:     "admin_session",
				Value:    "authenticated",
				Path:     "/",
				MaxAge:   3600, // 1 час
				HttpOnly: true,
			})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		
		// Неверные данные - показываем форму входа с ошибкой
		s.showLoginForm(w, "Неверный логин или пароль")
		return
	}
	
	// GET запрос - показываем форму входа
	s.showLoginForm(w, "")
}

func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		return
	}
	
	tmpl, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	
	tmpl.Execute(w, nil)
}

func (s *Server) broadcastHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Получаем всех пользователей
	customers, err := s.getAllCustomers(r.Context())
	if err != nil {
		slog.Error("Failed to get customers for broadcast", "error", err)
		http.Error(w, "Failed to get customers", http.StatusInternalServerError)
		return
	}

	sent := 0
	failed := 0

	// Отправляем сообщение через Telegram Bot API
	botToken := getEnv("TELEGRAM_TOKEN", "")
	if botToken == "" {
		http.Error(w, "Telegram token not configured", http.StatusInternalServerError)
		return
	}

	// Отправляем сообщение каждому пользователю
	for _, customer := range customers {
		// Используем Telegram Bot API для отправки сообщения
		err := s.sendTelegramMessage(customer.TelegramID, req.Message, botToken)
		if err != nil {
			slog.Error("Failed to send broadcast message", 
				"telegram_id", customer.TelegramID, 
				"error", err)
			failed++
		} else {
			sent++
		}

		// Небольшая задержка чтобы не спамить Telegram API
		time.Sleep(50 * time.Millisecond)
	}

	response := BroadcastResponse{
		Success: true,
		Message: fmt.Sprintf("Broadcast completed. Sent: %d, Failed: %d", sent, failed),
		Sent:    sent,
		Failed:  failed,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) logsHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	lines := "100"
	if r.URL.Query().Get("lines") != "" {
		lines = r.URL.Query().Get("lines")
	}

	logs, err := s.getContainerLogs(lines)
	
	response := LogsResponse{
		Success: err == nil,
		Logs:    logs,
	}

	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) getAllCustomers(ctx context.Context) ([]Customer, error) {
	query := `SELECT id, telegram_id, expire_at, created_at, subscription_link, language 
			  FROM customer 
			  ORDER BY created_at DESC`
	
	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers: %w", err)
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var customer Customer
		err := rows.Scan(
			&customer.ID,
			&customer.TelegramID,
			&customer.ExpireAt,
			&customer.CreatedAt,
			&customer.SubscriptionLink,
			&customer.Language,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

func (s *Server) sendTelegramMessage(chatID int64, message, botToken string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	
	data := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error: %d", resp.StatusCode)
	}
	
	return nil
}

func (s *Server) getContainerLogs(lines string) (string, error) {
	cmd := exec.Command("docker", "logs", "--tail", lines, "remnawave-telegram-shop-bot-1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	
	return string(output), nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// generateRandomString - генерирует случайную строку заданной длины
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// checkAuth - проверяет аутентификацию пользователя
func (s *Server) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	// Проверяем, есть ли уже сессия
	cookie, err := r.Cookie("admin_session")
	if err == nil && cookie.Value == "authenticated" {
		return true
	}
	
	// Для AJAX запросов возвращаем JSON ошибку
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || 
	   r.Header.Get("Content-Type") == "application/json" ||
	   r.URL.Path == "/admin/translations" ||
	   r.URL.Path == "/admin/translations/update" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Authentication required",
		})
		return false
	}
	
	// Показываем форму входа
	s.showLoginForm(w, "")
	return false
}

// showLoginForm - показывает форму входа
func (s *Server) showLoginForm(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	html := `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>VPN Admin - Вход</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; display: flex; justify-content: center; align-items: center; min-height: 100vh; }
        .login-container { background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); width: 100%; max-width: 400px; }
        .login-container h1 { text-align: center; margin-bottom: 30px; color: #333; }
        .form-group { margin-bottom: 20px; }
        .form-group label { display: block; margin-bottom: 8px; font-weight: bold; color: #333; }
        .form-group input { width: 100%; padding: 12px; border: 2px solid #e1e5e9; border-radius: 4px; font-size: 16px; box-sizing: border-box; }
        .form-group input:focus { outline: none; border-color: #007bff; }
        .btn { width: 100%; padding: 12px; background: #007bff; color: white; border: none; border-radius: 4px; font-size: 16px; cursor: pointer; }
        .btn:hover { background: #0056b3; }
        .error { color: #dc3545; text-align: center; margin-bottom: 20px; }
        .info { background: #e7f3ff; padding: 15px; border-radius: 4px; margin-top: 20px; border-left: 4px solid #007bff; }
        .info h3 { margin: 0 0 10px 0; color: #333; }
        .info p { margin: 5px 0; color: #666; }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>🔐 VPN Admin Panel</h1>
        <form method="POST" action="/login">
            <div class="form-group">
                <label for="username">Логин:</label>
                <input type="text" id="username" name="username" required>
            </div>
            <div class="form-group">
                <label for="password">Пароль:</label>
                <input type="password" id="password" name="password" required>
            </div>
            <button type="submit" class="btn">Войти</button>
        </form>
        <div class="info">
            <h3>ℹ️ Информация:</h3>
            <p>Данные для входа доступны в логах контейнера.</p>
            <p>Выполните команду: <code>docker logs vpn-admin-server</code></p>
        </div>
    </div>
</body>
</html>`
	
	if errorMsg != "" {
		html = strings.Replace(html, `<form method="POST" action="/login">`, 
			`<div class="error">`+errorMsg+`</div><form method="POST" action="/login">`, 1)
	}
	
	fmt.Fprint(w, html)
}

// translationsHandler - получение всех переводов
func (s *Server) translationsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Получен запрос на загрузку переводов от %s", r.RemoteAddr)
	
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		log.Printf("❌ Не пройдена аутентификация для %s", r.RemoteAddr)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("📁 Загружаем переводы из директории translations...")
	translations, err := s.loadAllTranslations()
	
	response := TranslationResponse{
		Success:      err == nil,
		Translations: translations,
	}

	if err != nil {
		log.Printf("❌ Ошибка загрузки переводов: %v", err)
		response.Error = err.Error()
	} else {
		log.Printf("✅ Переводы успешно загружены: %d языков", len(translations))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateTranslationHandler - обновление переводов
func (s *Server) updateTranslationHandler(w http.ResponseWriter, r *http.Request) {
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req UpdateTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Language == "" || req.Data == nil {
		http.Error(w, "Language and data are required", http.StatusBadRequest)
		return
	}

	err := s.saveTranslation(req.Language, req.Data)
	
	response := map[string]interface{}{
		"success": err == nil,
	}

	if err != nil {
		response["error"] = err.Error()
	} else {
		response["message"] = fmt.Sprintf("Переводы для языка %s успешно обновлены", req.Language)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// loadAllTranslations - загружает все файлы переводов
func (s *Server) loadAllTranslations() (map[string]map[string]string, error) {
	translationsDir := "translations"
	translations := make(map[string]map[string]string)

	files, err := os.ReadDir(translationsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read translations directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		langCode := strings.TrimSuffix(file.Name(), ".json")
		filePath := filepath.Join(translationsDir, file.Name())

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read translation file %s: %w", file.Name(), err)
		}

		var translation map[string]string
		if err := json.Unmarshal(content, &translation); err != nil {
			return nil, fmt.Errorf("failed to parse translation file %s: %w", file.Name(), err)
		}

		translations[langCode] = translation
	}

	return translations, nil
}

// saveTranslation - сохраняет переводы в файл
func (s *Server) saveTranslation(language string, data map[string]string) error {
	translationsDir := "translations"
	filePath := filepath.Join(translationsDir, language+".json")

	// Создаем директорию если она не существует
	if err := os.MkdirAll(translationsDir, 0755); err != nil {
		return fmt.Errorf("failed to create translations directory: %w", err)
	}

	// Форматируем JSON с отступами для читаемости
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal translation data: %w", err)
	}

	// Записываем файл
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write translation file: %w", err)
	}

	return nil
}

// restartBotHandler - перезапуск основного бота
func (s *Server) restartBotHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("🔄 Получен запрос на перезапуск бота от %s", r.RemoteAddr)
	
	// Проверяем аутентификацию
	if !s.checkAuth(w, r) {
		log.Printf("❌ Не пройдена аутентификация для перезапуска бота %s", r.RemoteAddr)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("🤖 Начинаем перезапуск основного бота...")
	err := s.restartMainBot()
	
	response := RestartBotResponse{
		Success: err == nil,
	}

	if err != nil {
		log.Printf("❌ Ошибка перезапуска бота: %v", err)
		response.Error = err.Error()
		response.Message = "Ошибка при перезапуске бота"
	} else {
		log.Printf("✅ Бот успешно перезапущен")
		response.Message = "Бот успешно перезапущен и загрузил новые переводы"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// restartMainBot - безопасный перезапуск основного бота
func (s *Server) restartMainBot() error {
	botContainerName := "remnawave-telegram-shop-bot-1"
	
	// Сначала проверяем, что контейнер существует
	checkCmd := exec.Command("docker", "inspect", botContainerName)
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("контейнер %s не найден: %w", botContainerName, err)
	}
	
	log.Printf("📋 Контейнер %s найден, выполняем graceful restart...", botContainerName)
	
	// Выполняем graceful restart контейнера
	restartCmd := exec.Command("docker", "restart", botContainerName)
	output, err := restartCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ошибка перезапуска контейнера %s: %w. Вывод: %s", 
			botContainerName, err, string(output))
	}
	
	log.Printf("🔄 Контейнер %s успешно перезапущен", botContainerName)
	
	// Ждем несколько секунд, чтобы контейнер успел запуститься
	time.Sleep(3 * time.Second)
	
	// Проверяем статус контейнера
	statusCmd := exec.Command("docker", "ps", "--filter", "name="+botContainerName, "--format", "{{.Status}}")
	statusOutput, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("не удалось проверить статус контейнера: %w", err)
	}
	
	status := strings.TrimSpace(string(statusOutput))
	if !strings.Contains(status, "Up") {
		return fmt.Errorf("контейнер не запустился корректно. Статус: %s", status)
	}
	
	log.Printf("✅ Контейнер %s запущен и работает. Статус: %s", botContainerName, status)
	return nil
} 