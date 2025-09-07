
function clearForm() {
    document.getElementById("message").value = "";
    document.getElementById("broadcast-result").style.display = "none";
}

document.getElementById("broadcast-form").addEventListener("submit", async function(e) {
    e.preventDefault();
    const message = document.getElementById("message").value.trim();
    if (!message) { alert("Введите сообщение"); return; }
    if (!confirm("Отправить сообщение всем пользователям?")) return;
    
    const submitBtn = e.target.querySelector("button[type=submit]");
    submitBtn.textContent = "Отправка...";
    submitBtn.disabled = true;
    
    try {
        const response = await fetch("/admin/broadcast", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ message: message })
        });
        const result = await response.json();
        
        const resultBox = document.getElementById("broadcast-result");
        const statusDiv = document.getElementById("broadcast-status");
        statusDiv.innerHTML = result.success ? 
            `<div style="color: green;">✅ ${result.message}</div>` : 
            `<div style="color: red;">❌ ${result.message}</div>`;
        resultBox.style.display = "block";
    } catch (error) {
        console.error("Error:", error);
        document.getElementById("broadcast-status").innerHTML = "<div style=\"color: red;\">❌ Ошибка сети</div>";
        document.getElementById("broadcast-result").style.display = "block";
    } finally {
        submitBtn.textContent = "🚀 Отправить рассылку";
        submitBtn.disabled = false;
    }
});

async function loadLogs() {
    const logsContent = document.getElementById("logs-content");
    const lines = document.getElementById("log-lines").value;
    logsContent.textContent = "Загрузка логов...";
    
    try {
        const response = await fetch(`/admin/logs?lines=${lines}`);
        const result = await response.json();
        logsContent.textContent = result.success ? (result.logs || "Логи пусты") : `Ошибка: ${result.error}`;
    } catch (error) {
        logsContent.textContent = "Ошибка загрузки логов: " + error.message;
    }
}

// Переменные для работы с переводами
let allTranslations = {};
let currentLanguage = "";
let originalTranslations = {};

// Загрузка всех переводов
async function loadTranslations() {
    try {
        const response = await fetch("/admin/translations", {
            method: "GET",
            credentials: "same-origin", // Включаем куки
            headers: {
                "X-Requested-With": "XMLHttpRequest"
            }
        });
        
        if (response.status === 401) {
            alert("Требуется повторная авторизация. Обновите страницу.");
            window.location.reload();
            return;
        }
        
        const result = await response.json();
        
        if (result.success) {
            allTranslations = result.translations;
            populateLanguageSelect();
            console.log("Переводы загружены:", allTranslations);
        } else {
            console.error("Ошибка загрузки переводов:", result.error);
            alert("Ошибка загрузки переводов: " + result.error);
        }
    } catch (error) {
        console.error("Ошибка сети при загрузке переводов:", error);
        alert("Ошибка сети при загрузке переводов: " + error.message);
    }
}

// Заполнение выпадающего списка языков
function populateLanguageSelect() {
    const select = document.getElementById("language-select");
    select.innerHTML = '<option value="">Выберите язык...</option>';
    
    const languageNames = {
        'ru': 'Русский (ru.json)',
        'en': 'English (en.json)'
    };
    
    for (const [langCode, translations] of Object.entries(allTranslations)) {
        const option = document.createElement("option");
        option.value = langCode;
        option.textContent = languageNames[langCode] || `${langCode.toUpperCase()} (${langCode}.json)`;
        select.appendChild(option);
    }
}

// Загрузка переводов для выбранного языка
function loadTranslationForLanguage() {
    const select = document.getElementById("language-select");
    const selectedLang = select.value;
    
    if (!selectedLang) {
        document.getElementById("translation-editor").style.display = "none";
        return;
    }
    
    currentLanguage = selectedLang;
    const translations = allTranslations[selectedLang];
    
    if (!translations) {
        alert("Переводы для выбранного языка не найдены");
        return;
    }
    
    // Сохраняем оригинальные переводы для возможности отмены
    originalTranslations = JSON.parse(JSON.stringify(translations));
    
    displayTranslationEditor(selectedLang, translations);
}

// Отображение редактора переводов
function displayTranslationEditor(language, translations) {
    const editor = document.getElementById("translation-editor");
    const currentLangSpan = document.getElementById("current-language");
    const fieldsContainer = document.getElementById("translation-fields");
    
    currentLangSpan.textContent = language.toUpperCase();
    fieldsContainer.innerHTML = "";
    
    // Создаем поля для каждого перевода
    for (const [key, value] of Object.entries(translations)) {
        const fieldDiv = document.createElement("div");
        fieldDiv.className = "translation-field";
        
        fieldDiv.innerHTML = `
            <label for="trans_${key}">Перевод для ключа:</label>
            <div class="field-key">${key}</div>
            <textarea id="trans_${key}" name="${key}">${escapeHtml(value)}</textarea>
        `;
        
        fieldsContainer.appendChild(fieldDiv);
    }
    
    editor.style.display = "block";
    document.getElementById("translation-result").style.display = "none";
}

// Сохранение переводов
async function saveTranslations() {
    if (!currentLanguage) {
        alert("Язык не выбран");
        return;
    }
    
    const fieldsContainer = document.getElementById("translation-fields");
    const textareas = fieldsContainer.querySelectorAll("textarea");
    const updatedTranslations = {};
    
    // Собираем обновленные переводы
    textareas.forEach(textarea => {
        const key = textarea.name;
        const value = textarea.value;
        updatedTranslations[key] = value;
    });
    
    try {
        const response = await fetch("/admin/translations/update", {
            method: "POST",
            credentials: "same-origin", // Включаем куки
            headers: {
                "Content-Type": "application/json",
                "X-Requested-With": "XMLHttpRequest"
            },
            body: JSON.stringify({
                language: currentLanguage,
                data: updatedTranslations
            })
        });
        
        const result = await response.json();
        const resultBox = document.getElementById("translation-result");
        const statusDiv = document.getElementById("translation-status");
        
        if (result.success) {
            // Обновляем локальные данные
            allTranslations[currentLanguage] = updatedTranslations;
            originalTranslations = JSON.parse(JSON.stringify(updatedTranslations));
            
            statusDiv.innerHTML = `<div style="color: green;">✅ ${result.message}<br/>🔄 Перезапускаем бота для применения изменений...</div>`;
            resultBox.className = "result-box result-success";
            
            // Автоматически перезапускаем бота
            restartBot();
        } else {
            statusDiv.innerHTML = `<div style="color: red;">❌ Ошибка: ${result.error}</div>`;
            resultBox.className = "result-box result-error";
        }
        
        resultBox.style.display = "block";
        
    } catch (error) {
        console.error("Ошибка сохранения:", error);
        const resultBox = document.getElementById("translation-result");
        const statusDiv = document.getElementById("translation-status");
        statusDiv.innerHTML = `<div style="color: red;">❌ Ошибка сети: ${error.message}</div>`;
        resultBox.className = "result-box result-error";
        resultBox.style.display = "block";
    }
}

// Отмена редактирования
function cancelEdit() {
    if (!currentLanguage) return;
    
    if (confirm("Отменить все изменения и вернуть исходные значения?")) {
        displayTranslationEditor(currentLanguage, originalTranslations);
        document.getElementById("translation-result").style.display = "none";
    }
}

// Вспомогательная функция для экранирования HTML
function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Модифицируем функцию showTab для автозагрузки переводов
function showTab(tabName) {
    document.querySelectorAll(".tab-content").forEach(tab => tab.classList.remove("active"));
    document.querySelectorAll(".tab-btn").forEach(btn => btn.classList.remove("active"));
    document.getElementById(tabName + "-tab").classList.add("active");
    event.target.classList.add("active");
    
    if (tabName === "logs") loadLogs();
    
    // Если открываем вкладку переводов, загружаем данные
    if (tabName === 'translations' && Object.keys(allTranslations).length === 0) {
        loadTranslations();
    }
}

// Функция перезапуска основного бота
async function restartBot() {
    try {
        const response = await fetch("/admin/restart-bot", {
            method: "POST",
            credentials: "same-origin",
            headers: {
                "X-Requested-With": "XMLHttpRequest",
                "Content-Type": "application/json"
            }
        });
        
        if (response.status === 401) {
            alert("Требуется повторная авторизация. Обновите страницу.");
            window.location.reload();
            return;
        }
        
        const result = await response.json();
        const statusDiv = document.getElementById("translation-status");
        
        if (result.success) {
            statusDiv.innerHTML = `<div style="color: green;">✅ Переводы сохранены<br/>🎉 Бот успешно перезапущен и загрузил новые переводы!</div>`;
        } else {
            statusDiv.innerHTML = `<div style="color: orange;">⚠️ Переводы сохранены, но не удалось перезапустить бота автоматически.<br/>❌ ${result.error}<br/>💡 Попробуйте перезапустить бота вручную.</div>`;
        }
        
    } catch (error) {
        console.error("Ошибка перезапуска бота:", error);
        const statusDiv = document.getElementById("translation-status");
        statusDiv.innerHTML = `<div style="color: orange;">⚠️ Переводы сохранены, но произошла ошибка при перезапуске бота.<br/>💡 Попробуйте перезапустить бота вручную.</div>`;
    }
}
