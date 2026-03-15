(function() {
    // --- Конфигурация API ---
    const API_BASE = ''; 

    // --- Элементы DOM ---
    const form = document.getElementById('itemForm');
    const editId = document.getElementById('editId');
    const categorySelect = document.getElementById('category');
    const amountInput = document.getElementById('amount');
    const dateInput = document.getElementById('date');
    const formSubmitBtn = document.getElementById('formSubmitBtn');
    const cancelEditBtn = document.getElementById('cancelEditBtn');

    const fromDate = document.getElementById('fromDate');
    const toDate = document.getElementById('toDate');
    const applyFilterBtn = document.getElementById('applyFilterBtn');
    const exportCsvBtn = document.getElementById('exportCsvBtn');
    const groupBtn = document.getElementById('groupBtn');
    const groupSection = document.getElementById('groupSection');
    const groupBody = document.getElementById('groupBody');

    const metrics = {
        sum: document.getElementById('sumValue'),
        avg: document.getElementById('avgValue'),
        count: document.getElementById('countValue'),
        median: document.getElementById('medianValue'),
        percentile90: document.getElementById('percentile90Value')
    };

    const itemsBody = document.getElementById('itemsBody');
    const notification = document.getElementById('notification');

    // --- Состояние приложения ---
    let items = [];
    let sortField = 'date';        // поле для сортировки
    let sortDirection = 'desc';    // desc - новые сверху

    // --- Установка начальных дат (текущий месяц) ---
    function setDefaultDates() {
        const today = new Date();
        const firstDay = new Date(today.getFullYear(), today.getMonth(), 1);
        const lastDay = new Date(today.getFullYear(), today.getMonth() + 1, 0);
        fromDate.value = formatDateLocal(firstDay);
        toDate.value = formatDateLocal(lastDay);
    }
    function formatDateLocal(date) {
        const year = date.getFullYear();
        const month = String(date.getMonth() + 1).padStart(2, '0');
        const day = String(date.getDate()).padStart(2, '0');
        return `${year}-${month}-${day}`;
    }
    setDefaultDates();

    // --- Вспомогательные функции ---
    function showNotification(message, type = 'info') {
        notification.textContent = message;
        notification.className = `notification show ${type}`;
        setTimeout(() => {
            notification.classList.remove('show');
        }, 3000);
    }

    function formatCurrency(value) {
        return new Intl.NumberFormat('ru-RU', { style: 'currency', currency: 'RUB', minimumFractionDigits: 0 }).format(value);
    }

    // --- Загрузка данных с сервера ---
    async function loadItems(from, to) {
        try {
            const url = new URL(`${API_BASE}/items`, window.location.origin);
            if (from) url.searchParams.append('from', from);
            if (to) url.searchParams.append('to', to);
            if (sortField) {
                url.searchParams.append('sort_by', sortField);
                url.searchParams.append('sort_order', sortDirection);
            }
            const response = await fetch(url);
            if (!response.ok) throw new Error(`Ошибка загрузки: ${response.status}`);
            items = await response.json();
            renderTable();
        } catch (err) {
            showNotification(err.message, 'error');
            items = [];
            renderTable();
        }
    }

    async function loadAnalytics(from, to) {
        try {
            const url = new URL(`${API_BASE}/analytics`, window.location.origin);
            if (from) url.searchParams.append('from', from);
            if (to) url.searchParams.append('to', to);
            const response = await fetch(url);
            if (!response.ok) throw new Error(`Ошибка аналитики: ${response.status}`);
            const data = await response.json();
            metrics.sum.textContent = formatCurrency(data.sum || 0);
            metrics.avg.textContent = formatCurrency(data.avg || 0);
            metrics.count.textContent = data.count || 0;
            metrics.median.textContent = formatCurrency(data.median || 0);
            metrics.percentile90.textContent = formatCurrency(data.percentile90 || 0);
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }

    async function loadGroupByCategory(from, to) {
        try {
            const url = new URL(`${API_BASE}/analytics/by-category`, window.location.origin);
            if (from) url.searchParams.append('from', from);
            if (to) url.searchParams.append('to', to);
            const response = await fetch(url);
            if (!response.ok) throw new Error(`Ошибка группировки: ${response.status}`);
            const data = await response.json();
            if (!data.length) {
                groupBody.innerHTML = '<tr><td colspan="3" style="text-align:center;">Нет данных</td></tr>';
            } else {
                const rows = data.map(item => `
                    <tr><td>${escapeHtml(item.category)}</td><td>${formatCurrency(item.sum)}</td><td>${item.count}</td></tr>
                `).join('');
                groupBody.innerHTML = rows;
            }
            groupSection.classList.remove('hidden');
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }

    // --- Экспорт CSV (через бэкенд) ---
    async function exportCSV() {
        const from = fromDate.value;
        const to = toDate.value;
        try {
            const url = new URL(`${API_BASE}/export/csv`, window.location.origin);
            if (from) url.searchParams.append('from', from);
            if (to) url.searchParams.append('to', to);
            const response = await fetch(url);
            if (!response.ok) throw new Error(`Ошибка экспорта: ${response.status}`);
            const blob = await response.blob();
            const link = document.createElement('a');
            link.href = URL.createObjectURL(blob);
            link.download = `sales_${from}_${to}.csv`;
            link.click();
            URL.revokeObjectURL(link.href);
            showNotification('CSV скачан', 'success');
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }
    exportCsvBtn.addEventListener('click', exportCSV);

    // --- Отрисовка таблицы ---
    function renderTable() {
        if (!items.length) {
            itemsBody.innerHTML = '<tr><td colspan="4" style="text-align:center; padding:40px;">📭 Нет записей за выбранный период</td></tr>';
            return;
        }
        const rows = items.map(item => `
            <tr>
                <td>${item.date}</td>
                <td>${escapeHtml(item.category)}</td>
                <td class="amount-cell">${formatCurrency(item.amount)}</td>
                <td class="actions">
                    <button class="edit-btn" data-id="${item.id}">✏️</button>
                    <button class="delete-btn" data-id="${item.id}">🗑️</button>
                </td>
            </tr>
        `).join('');
        itemsBody.innerHTML = rows;

        document.querySelectorAll('.edit-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const id = Number(e.target.dataset.id);
                const item = items.find(i => i.id === id);
                if (item) fillFormForEdit(item);
            });
        });
        document.querySelectorAll('.delete-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const id = Number(e.target.dataset.id);
                deleteItem(id);
            });
        });
    }

    function escapeHtml(unsafe) {
        return unsafe.replace(/[&<>"]/g, function(m) {
            if(m === '&') return '&amp;'; if(m === '<') return '&lt;'; if(m === '>') return '&gt;'; if(m === '"') return '&quot;';
            return m;
        });
    }

    // --- Сортировка (клик на заголовки) ---
    document.querySelectorAll('#itemsTable th[data-field]').forEach(th => {
        th.addEventListener('click', () => {
            const field = th.dataset.field;
            if (sortField === field) {
                sortDirection = sortDirection === 'asc' ? 'desc' : 'asc';
            } else {
                sortField = field;
                sortDirection = 'asc';
            }
            document.querySelectorAll('#itemsTable th[data-field]').forEach(el => {
                el.classList.remove('sort-asc', 'sort-desc');
            });
            th.classList.add(sortDirection === 'asc' ? 'sort-asc' : 'sort-desc');
            refreshData();
        });
    });

    // --- CRUD операции ---
    async function createItem(data) {
        try {
            const response = await fetch(`${API_BASE}/items`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || 'Ошибка создания');
            }
            showNotification('Запись добавлена', 'success');
            resetForm();
            refreshData();
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }

    async function updateItem(id, data) {
        try {
            const response = await fetch(`${API_BASE}/items/${id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || 'Ошибка обновления');
            }
            showNotification('Запись обновлена', 'success');
            resetForm();
            refreshData();
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }

    async function deleteItem(id) {
        if (!confirm('Удалить запись?')) return;
        try {
            const response = await fetch(`${API_BASE}/items/${id}`, {
                method: 'DELETE'
            });
            if (!response.ok) {
                const err = await response.json();
                throw new Error(err.error || 'Ошибка удаления');
            }
            showNotification('Запись удалена', 'success');
            refreshData();
        } catch (err) {
            showNotification(err.message, 'error');
        }
    }

    // --- Форма ---
    function fillFormForEdit(item) {
        editId.value = item.id;
        categorySelect.value = item.category;
        amountInput.value = item.amount;
        dateInput.value = item.date;
        formSubmitBtn.innerHTML = '<span>🔄</span> Обновить';
        cancelEditBtn.classList.remove('hidden');
    }

    function resetForm() {
        editId.value = '';
        form.reset();
        formSubmitBtn.innerHTML = '<span>💾</span> Сохранить';
        cancelEditBtn.classList.add('hidden');
    }

    cancelEditBtn.addEventListener('click', resetForm);

    form.addEventListener('submit', (e) => {
        e.preventDefault();
        const data = {
            category: categorySelect.value,
            amount: parseFloat(amountInput.value),
            date: dateInput.value
        };
        if (!data.category || !data.amount || !data.date) {
            showNotification('Заполните все поля', 'error');
            return;
        }
        const id = editId.value;
        if (id) {
            updateItem(Number(id), data);
        } else {
            createItem(data);
        }
    });

    // --- Применение фильтра ---
    function refreshData() {
        const from = fromDate.value;
        const to = toDate.value;
        loadItems(from, to);
        loadAnalytics(from, to);
    }
    applyFilterBtn.addEventListener('click', refreshData);

    // --- Группировка ---
    groupBtn.addEventListener('click', () => {
        const from = fromDate.value;
        const to = toDate.value;
        loadGroupByCategory(from, to);
    });

    // --- Инициализация ---
    refreshData();
})();