// Общая функция уведомлений
function showNotification(message, isError = false) {
    const notif = document.getElementById('notification');
    if (!notif) return;
    notif.textContent = message;
    notif.className = 'notification' + (isError ? ' error' : '');
    notif.style.display = 'block';
    setTimeout(() => {
        notif.style.display = 'none';
    }, 3000);
}