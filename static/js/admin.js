function showTab(tabName) {
    document.querySelectorAll(".tab-content").forEach(tab => tab.classList.remove("active"));
    document.querySelectorAll(".tab-btn").forEach(btn => btn.classList.remove("active"));
    document.getElementById(tabName + "-tab").classList.add("active");
    event.target.classList.add("active");
    if (tabName === "logs") loadLogs();
}

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
