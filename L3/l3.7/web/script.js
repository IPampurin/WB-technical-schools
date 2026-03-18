// web/script.js

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

// Получение токена из localStorage
function getToken() {
    return localStorage.getItem('jwtToken');
}

// Сохранение токена и роли
function setAuthData(token, role) {
    localStorage.setItem('jwtToken', token);
    localStorage.setItem('userRole', role);
}

// Удаление данных аутентификации (выход)
function clearAuth() {
    localStorage.removeItem('jwtToken');
    localStorage.removeItem('userRole');
}

// Получение текущей роли из localStorage
function getRole() {
    return localStorage.getItem('userRole');
}

// Проверка, авторизован ли пользователь и имеет ли нужную роль
function checkAuth(expectedRole) {
    const token = getToken();
    const role = getRole();
    if (!token || !role) {
        window.location.href = '/';
        return false;
    }
    if (expectedRole && role !== expectedRole) {
        window.location.href = '/';
        return false;
    }
    return true;
}

// Функция для выполнения fetch с авторизацией
async function authFetch(url, options = {}) {
    const token = getToken();
    if (!token) {
        throw new Error('Нет токена');
    }
    const headers = {
        'Content-Type': 'application/json',
        ...options.headers,
        'Authorization': `Bearer ${token}`
    };
    const response = await fetch(url, { ...options, headers });
    if (response.status === 401) {
        clearAuth();
        window.location.href = '/';
        throw new Error('Сессия истекла');
    }
    return response;
}

// --- Общие функции для страниц ролей ---

// Загрузка списка товаров
async function loadItems() {
    const container = document.getElementById('itemsContainer');
    if (!container) return;

    try {
        const response = await authFetch('/items');
        const items = await response.json();
        renderItems(items);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка загрузки товаров: ' + err.message, true);
        }
    }
}

// Отрисовка товаров с учётом роли
function renderItems(items) {
    const container = document.getElementById('itemsContainer');
    if (!container) return;

    const role = getRole();
    const header = container.querySelector('.item-header');
    container.innerHTML = '';
    if (header) {
        container.appendChild(header);
    } else {
        const newHeader = document.createElement('div');
        newHeader.className = 'item-header';
        newHeader.innerHTML = `
            <span class="item-id">ID</span>
            <span class="item-name">Название</span>
            <span class="item-quantity">Количество</span>
            <span class="item-price">Цена</span>
            ${role !== 'viewer' ? '<span class="item-actions-header">Действия</span>' : ''}
        `;
        container.appendChild(newHeader);
    }

    items.forEach(item => {
        const row = document.createElement('div');
        row.className = 'item-row';
        row.dataset.id = item.id;

        let actionsHtml = '';
        if (role !== 'viewer') {
            actionsHtml += `<button class="edit-btn" data-id="${item.id}" data-name="${item.name}" data-quantity="${item.quantity}" data-price="${item.price}">Редактировать</button>`;
            if (role === 'admin') {
                actionsHtml += `<button class="delete-btn" data-id="${item.id}">Удалить</button>`;
                actionsHtml += `<button class="history-btn" data-id="${item.id}">История</button>`;
            }
        }

        row.innerHTML = `
            <span class="item-id">${item.id}</span>
            <span class="item-name">${item.name}</span>
            <span class="item-quantity">${item.quantity}</span>
            <span class="item-price">${item.price}</span>
            ${role !== 'viewer' ? `<div class="item-actions">${actionsHtml}</div>` : ''}
        `;

        container.appendChild(row);
    });

    // Привязываем обработчики (будут определены ниже в зависимости от страницы)
    document.querySelectorAll('.edit-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const id = e.target.dataset.id;
            const name = e.target.dataset.name;
            const quantity = e.target.dataset.quantity;
            const price = e.target.dataset.price;
            if (typeof window.editItem === 'function') {
                window.editItem(id, name, quantity, price);
            }
        });
    });

    document.querySelectorAll('.delete-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const id = e.target.dataset.id;
            if (typeof window.deleteItem === 'function') {
                window.deleteItem(id);
            }
        });
    });

    document.querySelectorAll('.history-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const id = e.target.dataset.id;
            if (typeof window.showItemHistory === 'function') {
                window.showItemHistory(id);
            }
        });
    });
}

// --- Функции для страницы администратора (и менеджера, где применимо) ---

// Создание товара
async function createItem() {
    const name = document.getElementById('itemName').value;
    const quantity = parseInt(document.getElementById('itemQuantity').value);
    const price = parseFloat(document.getElementById('itemPrice').value);

    if (!name || isNaN(quantity) || isNaN(price)) {
        showNotification('Заполните все поля', true);
        return;
    }

    try {
        const response = await authFetch('/items', {
            method: 'POST',
            body: JSON.stringify({ name, quantity, price })
        });
        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.error || 'Ошибка создания');
        }
        showNotification('Товар добавлен');
        document.getElementById('createForm').style.display = 'none';
        clearCreateForm();
        loadItems();
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка: ' + err.message, true);
        }
    }
}

function clearCreateForm() {
    document.getElementById('itemName').value = '';
    document.getElementById('itemQuantity').value = '';
    document.getElementById('itemPrice').value = '';
}

// Редактирование товара (открытие модалки)
window.editItem = function(id, name, quantity, price) {
    document.getElementById('editItemId').value = id;
    document.getElementById('editItemName').value = name;
    document.getElementById('editItemQuantity').value = quantity;
    document.getElementById('editItemPrice').value = price;
    document.getElementById('editModal').style.display = 'block';
};

// Обновление товара
async function updateItem() {
    const id = document.getElementById('editItemId').value;
    const name = document.getElementById('editItemName').value;
    const quantity = parseInt(document.getElementById('editItemQuantity').value);
    const price = parseFloat(document.getElementById('editItemPrice').value);

    if (!name || isNaN(quantity) || isNaN(price)) {
        showNotification('Заполните все поля', true);
        return;
    }

    try {
        const response = await authFetch(`/items/${id}`, {
            method: 'PUT',
            body: JSON.stringify({ name, quantity, price })
        });
        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.error || 'Ошибка обновления');
        }
        showNotification('Товар обновлён');
        document.getElementById('editModal').style.display = 'none';
        loadItems();
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка: ' + err.message, true);
        }
    }
}

// Удаление товара
window.deleteItem = async function(id) {
    if (!confirm('Удалить товар?')) return;
    try {
        const response = await authFetch(`/items/${id}`, { method: 'DELETE' });
        if (!response.ok) {
            const err = await response.json();
            throw new Error(err.error || 'Ошибка удаления');
        }
        showNotification('Товар удалён');
        loadItems();
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка: ' + err.message, true);
        }
    }
};

// --- История (для админа) ---

// Загрузка истории конкретного товара
window.showItemHistory = async function(id) {
    document.getElementById('itemHistoryItemId').textContent = id;
    // Сброс фильтров
    document.getElementById('itemHistoryFromDate').value = '';
    document.getElementById('itemHistoryToDate').value = '';
    document.getElementById('itemHistoryUserId').value = '';
    document.getElementById('itemHistoryAction').value = '';
    await loadItemHistory(id);
    document.getElementById('itemHistoryModal').style.display = 'block';
};

async function loadItemHistory(itemId) {
    const params = new URLSearchParams();
    params.append('item_id', itemId);
    const fromDate = document.getElementById('itemHistoryFromDate').value;
    const toDate = document.getElementById('itemHistoryToDate').value;
    const userId = document.getElementById('itemHistoryUserId').value;
    const action = document.getElementById('itemHistoryAction').value;
    if (fromDate) params.append('from_date', new Date(fromDate).toISOString());
    if (toDate) params.append('to_date', new Date(toDate).toISOString());
    if (userId) params.append('user_id', userId);
    if (action) params.append('action', action);

    try {
        const response = await authFetch(`/items/${itemId}/history?${params.toString()}`);
        const history = await response.json();
        renderItemHistory(history);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка загрузки истории: ' + err.message, true);
        }
    }
}

function renderItemHistory(history) {
    const container = document.getElementById('itemHistoryRows');
    container.innerHTML = '';
    history.forEach(record => {
        const row = document.createElement('div');
        row.className = 'history-row';
        row.dataset.id = record.id;
        row.innerHTML = `
            <span class="history-select"><input type="checkbox" class="history-checkbox" data-id="${record.id}"></span>
            <span class="history-id">${record.id}</span>
            <span class="history-action">${record.action}</span>
            <span class="history-user">${record.changed_by || '-'}</span>
            <span class="history-date">${new Date(record.changed_at).toLocaleString()}</span>
            <span class="history-changes">${formatChanges(record)}</span>
        `;
        container.appendChild(row);
    });
    updateCompareButton('itemHistoryRows', 'compareItemSelectedBtn');
    document.querySelectorAll('#itemHistoryRows .history-checkbox').forEach(cb => {
        cb.addEventListener('change', () => updateCompareButton('itemHistoryRows', 'compareItemSelectedBtn'));
    });
}

// Загрузка глобальной истории
async function loadGlobalHistory() {
    const params = new URLSearchParams();
    const itemId = document.getElementById('historyItemId').value;
    const fromDate = document.getElementById('historyFromDate').value;
    const toDate = document.getElementById('historyToDate').value;
    const userId = document.getElementById('historyUserId').value;
    const action = document.getElementById('historyAction').value;
    if (itemId) params.append('item_id', itemId);
    if (fromDate) params.append('from_date', new Date(fromDate).toISOString());
    if (toDate) params.append('to_date', new Date(toDate).toISOString());
    if (userId) params.append('user_id', userId);
    if (action) params.append('action', action);

    try {
        const response = await authFetch(`/history?${params.toString()}`);
        const history = await response.json();
        renderGlobalHistory(history);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка загрузки глобальной истории: ' + err.message, true);
        }
    }
}

function renderGlobalHistory(history) {
    const container = document.getElementById('historyRows');
    container.innerHTML = '';
    history.forEach(record => {
        const row = document.createElement('div');
        row.className = 'history-row';
        row.dataset.id = record.id;
        row.innerHTML = `
            <span class="history-select"><input type="checkbox" class="history-checkbox" data-id="${record.id}"></span>
            <span class="history-id">${record.id}</span>
            <span class="history-item-id">${record.item_id}</span>
            <span class="history-action">${record.action}</span>
            <span class="history-user">${record.changed_by || '-'}</span>
            <span class="history-date">${new Date(record.changed_at).toLocaleString()}</span>
            <span class="history-changes">${formatChanges(record)}</span>
        `;
        container.appendChild(row);
    });
    updateCompareButton('historyRows', 'compareSelectedBtn');
    document.querySelectorAll('#historyRows .history-checkbox').forEach(cb => {
        cb.addEventListener('change', () => updateCompareButton('historyRows', 'compareSelectedBtn'));
    });
}

// Форматирование изменений
function formatChanges(record) {
    if (record.action === 'INSERT') {
        return `Добавлен: ${JSON.stringify(record.new_data)}`;
    } else if (record.action === 'DELETE') {
        return `Удалён: ${JSON.stringify(record.old_data)}`;
    } else if (record.action === 'UPDATE') {
        const old = record.old_data || {};
        const newd = record.new_data || {};
        const changes = [];
        if (old.name !== newd.name) changes.push(`name: ${old.name} → ${newd.name}`);
        if (old.quantity !== newd.quantity) changes.push(`quantity: ${old.quantity} → ${newd.quantity}`);
        if (old.price !== newd.price) changes.push(`price: ${old.price} → ${newd.price}`);
        return changes.join('; ') || 'нет изменений';
    }
    return '';
}

// Обновление кнопки сравнения
function updateCompareButton(rowsId, btnId) {
    const checkboxes = document.querySelectorAll(`#${rowsId} .history-checkbox:checked`);
    const btn = document.getElementById(btnId);
    if (checkboxes.length === 2) {
        btn.disabled = false;
    } else {
        btn.disabled = true;
    }
}

// Сравнение версий
async function compareVersions(id1, id2) {
    try {
        const response = await authFetch(`/history/compare?from=${id1}&to=${id2}`);
        const diff = await response.json();
        showDiff(diff);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка сравнения: ' + err.message, true);
        }
    }
}

function showDiff(diff) {
    const container = document.getElementById('diffContent');
    if (!diff.changes || diff.changes.length === 0) {
        container.innerHTML = '<p>Нет различий</p>';
    } else {
        let html = '<table class="diff-table"><tr><th>Поле</th><th>Было</th><th>Стало</th></tr>';
        diff.changes.forEach(change => {
            html += `<tr><td>${change.field}</td><td>${JSON.stringify(change.old_value)}</td><td>${JSON.stringify(change.new_value)}</td></tr>`;
        });
        html += '</table>';
        container.innerHTML = html;
    }
    document.getElementById('diffModal').style.display = 'block';
}

// Экспорт товаров CSV
async function exportItemsCSV() {
    try {
        const response = await authFetch('/export/items/csv');
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'items.csv';
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка экспорта: ' + err.message, true);
        }
    }
}

// Экспорт истории CSV
async function exportHistoryCSV(endpoint, params) {
    try {
        const response = await authFetch(`${endpoint}?${params.toString()}`);
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'history.csv';
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
    } catch (err) {
        if (err.message !== 'Сессия истекла') {
            showNotification('Ошибка экспорта: ' + err.message, true);
        }
    }
}

// --- Инициализация в зависимости от страницы ---
document.addEventListener('DOMContentLoaded', () => {
    const path = window.location.pathname;
    const role = getRole();

    // Если мы на главной странице — навешиваем обработчики на кнопки
    if (path === '/' || path.endsWith('index.html')) {
        console.log('Main page: attaching login handlers');
        document.querySelectorAll('.circle-button').forEach(btn => {
            btn.addEventListener('click', async (e) => {
                const role = btn.dataset.role; // 'admin', 'manager', 'viewer'
                console.log('Login clicked for role:', role);
                try {
                    const response = await fetch('/login', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ role })
                    });
                    if (!response.ok) {
                        const err = await response.json();
                        throw new Error(err.error || 'Ошибка входа');
                    }
                    const data = await response.json();
                    console.log('Login success, token:', data.token);
                    setAuthData(data.token, role);
                    window.location.href = `/${role}.html`;
                } catch (err) {
                    console.error('Login error:', err);
                    showNotification('Ошибка входа: ' + err.message, true);
                }
            });
        });
        return; // Выходим, дальше не идём
    }

    // Для остальных страниц проверяем авторизацию
    if (!checkAuth()) return;

    // Определяем текущую страницу и навешиваем обработчики
    if (path.endsWith('admin.html')) {
        if (role !== 'admin') {
            window.location.href = '/';
            return;
        }
        // Инициализация для admin
        loadItems();

        // Вкладки
        document.querySelectorAll('.tab-button').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.tab-button').forEach(b => b.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
                e.target.classList.add('active');
                const tabId = e.target.dataset.tab;
                document.getElementById(`tab-${tabId}`).classList.add('active');
                if (tabId === 'history') {
                    loadGlobalHistory();
                }
            });
        });

        // Создание товара
        document.getElementById('createItemBtn').addEventListener('click', () => {
            document.getElementById('createForm').style.display = 'block';
        });
        document.getElementById('cancelCreateBtn').addEventListener('click', () => {
            document.getElementById('createForm').style.display = 'none';
            clearCreateForm();
        });
        document.getElementById('publishItemBtn').addEventListener('click', createItem);
        document.getElementById('exportItemsCsvBtn').addEventListener('click', exportItemsCSV);

        // Редактирование
        document.getElementById('closeEditModal').onclick = () => {
            document.getElementById('editModal').style.display = 'none';
        };
        document.getElementById('updateItemBtn').addEventListener('click', updateItem);

        // Модалка истории товара
        document.getElementById('closeItemHistoryModal').onclick = () => {
            document.getElementById('itemHistoryModal').style.display = 'none';
        };
        document.getElementById('applyItemHistoryFilters').addEventListener('click', () => {
            const itemId = document.getElementById('itemHistoryItemId').textContent;
            if (itemId) loadItemHistory(parseInt(itemId));
        });
        document.getElementById('exportItemHistoryCsvBtn').addEventListener('click', () => {
            const itemId = document.getElementById('itemHistoryItemId').textContent;
            if (!itemId) return;
            const params = new URLSearchParams();
            params.append('item_id', itemId);
            const fromDate = document.getElementById('itemHistoryFromDate').value;
            const toDate = document.getElementById('itemHistoryToDate').value;
            const userId = document.getElementById('itemHistoryUserId').value;
            const action = document.getElementById('itemHistoryAction').value;
            if (fromDate) params.append('from_date', new Date(fromDate).toISOString());
            if (toDate) params.append('to_date', new Date(toDate).toISOString());
            if (userId) params.append('user_id', userId);
            if (action) params.append('action', action);
            exportHistoryCSV('/export/history/csv', params);
        });
        document.getElementById('compareItemSelectedBtn').addEventListener('click', () => {
            const checkboxes = document.querySelectorAll('#itemHistoryRows .history-checkbox:checked');
            if (checkboxes.length !== 2) return;
            const ids = Array.from(checkboxes).map(cb => cb.dataset.id);
            compareVersions(ids[0], ids[1]);
        });

        // Глобальная история
        document.getElementById('applyHistoryFilters').addEventListener('click', loadGlobalHistory);
        document.getElementById('exportHistoryCsvBtn').addEventListener('click', () => {
            const params = new URLSearchParams();
            const itemId = document.getElementById('historyItemId').value;
            const fromDate = document.getElementById('historyFromDate').value;
            const toDate = document.getElementById('historyToDate').value;
            const userId = document.getElementById('historyUserId').value;
            const action = document.getElementById('historyAction').value;
            if (itemId) params.append('item_id', itemId);
            if (fromDate) params.append('from_date', new Date(fromDate).toISOString());
            if (toDate) params.append('to_date', new Date(toDate).toISOString());
            if (userId) params.append('user_id', userId);
            if (action) params.append('action', action);
            exportHistoryCSV('/export/history/csv', params);
        });
        document.getElementById('compareSelectedBtn').addEventListener('click', () => {
            const checkboxes = document.querySelectorAll('#historyRows .history-checkbox:checked');
            if (checkboxes.length !== 2) return;
            const ids = Array.from(checkboxes).map(cb => cb.dataset.id);
            compareVersions(ids[0], ids[1]);
        });

        // Модалка diff
        document.getElementById('closeDiffModal').onclick = () => {
            document.getElementById('diffModal').style.display = 'none';
        };
        window.onclick = (event) => {
            if (event.target == document.getElementById('editModal')) {
                document.getElementById('editModal').style.display = 'none';
            }
            if (event.target == document.getElementById('itemHistoryModal')) {
                document.getElementById('itemHistoryModal').style.display = 'none';
            }
            if (event.target == document.getElementById('diffModal')) {
                document.getElementById('diffModal').style.display = 'none';
            }
        };

    } else if (path.endsWith('manager.html')) {
        if (role !== 'manager') {
            window.location.href = '/';
            return;
        }
        // Инициализация для manager
        loadItems();

        document.getElementById('createItemBtn').addEventListener('click', () => {
            document.getElementById('createForm').style.display = 'block';
        });
        document.getElementById('cancelCreateBtn').addEventListener('click', () => {
            document.getElementById('createForm').style.display = 'none';
            clearCreateForm();
        });
        document.getElementById('publishItemBtn').addEventListener('click', createItem);

        document.getElementById('closeEditModal').onclick = () => {
            document.getElementById('editModal').style.display = 'none';
        };
        document.getElementById('updateItemBtn').addEventListener('click', updateItem);
        window.onclick = (event) => {
            if (event.target == document.getElementById('editModal')) {
                document.getElementById('editModal').style.display = 'none';
            }
        };

    } else if (path.endsWith('viewer.html')) {
        if (role !== 'viewer') {
            window.location.href = '/';
            return;
        }
        // Инициализация для viewer
        loadItems();
    }

    // Общий обработчик выхода
    const logoutBtn = document.getElementById('logoutBtn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', () => {
            clearAuth();
            window.location.href = '/';
        });
    }
});