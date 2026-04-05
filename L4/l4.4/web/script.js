let refreshIntervalMs = 3000;
let intervalId = null;

// Загрузка метрик с /api/stats
async function fetchMetrics() {
    try {
        const response = await fetch('/api/stats');
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        const data = await response.json();
        renderMetrics(data);
        document.getElementById('currentGcPercent').innerText = data.gc_percent;
        document.getElementById('updateIndicator').innerHTML = `Обновлено: ${new Date().toLocaleTimeString()}`;
    } catch (err) {
        console.error('Ошибка загрузки метрик:', err);
        document.getElementById('metricsGrid').innerHTML = '<div class="card" style="grid-column:1/-1; text-align:center;">⚠️ Ошибка загрузки данных</div>';
    }
}

function renderMetrics(data) {
    const grid = document.getElementById('metricsGrid');
    const cards = [
        { title: '💾 Используемая память', value: data.alloc_bytes, desc: 'Байт, выделенных и используемых в данный момент (HeapAlloc)' },
        { title: '📈 Всего выделено памяти', value: data.total_alloc_bytes, desc: 'Общее количество байт, выделенных за всё время' },
        { title: '🔢 Количество аллокаций', value: data.mallocs.toLocaleString(), desc: 'Общее количество выделений памяти (mallocs)' },
        { title: '♻️ Количество сборок GC', value: data.num_gc, desc: 'Завершённых циклов сборки мусора' },
        { title: '🕒 Время последнего GC', value: data.last_gc_time, desc: 'Дата и время последней сборки мусора' },
        { title: '⚙️ Доля CPU на GC', value: (data.gc_cpu_fraction * 100).toFixed(3) + '%', desc: 'Доля процессорного времени, потраченная на GC' },
        { title: '🎯 GOGC процент', value: data.gc_percent + '%', desc: 'Целевой процент кучи, при котором запускается GC (можно менять)' }
    ];
    grid.innerHTML = '';
    cards.forEach(card => {
        const cardDiv = document.createElement('div');
        cardDiv.className = 'card';
        cardDiv.innerHTML = `
            <h3>${card.title}</h3>
            <div class="metric-value">${card.value}</div>
            <div class="metric-desc">${card.desc}</div>
        `;
        grid.appendChild(cardDiv);
    });
}

async function setGcPercent(percent) {
    try {
        const response = await fetch(`/gc_percent?percent=${percent}`, { method: 'POST' });
        if (!response.ok) {
            const text = await response.text();
            throw new Error(text);
        }
        const result = await response.json();
        const statusDiv = document.getElementById('gcStatus');
        statusDiv.innerHTML = `✅ GOGC изменён с ${result.old_percent}% на ${result.new_percent}%`;
        setTimeout(() => {
            if (statusDiv.innerHTML.includes('✅')) statusDiv.innerHTML = '';
        }, 3000);
        fetchMetrics();
    } catch (err) {
        document.getElementById('gcStatus').innerHTML = `❌ Ошибка: ${err.message}`;
    }
}

async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        if (!response.ok) throw new Error('Не удалось загрузить конфигурацию');
        const config = await response.json();
        if (config.metrics_timeout && typeof config.metrics_timeout === 'number') {
            refreshIntervalMs = config.metrics_timeout * 1000;
        }
    } catch (err) {
        console.warn('Не удалось загрузить конфиг, интервал по умолчанию 3с');
    }
}

function startAutoRefresh() {
    if (intervalId) clearInterval(intervalId);
    intervalId = setInterval(fetchMetrics, refreshIntervalMs);
    console.log(`Автообновление запущено: ${refreshIntervalMs / 1000} сек`);
}

document.addEventListener('DOMContentLoaded', async () => {
    await loadConfig();
    startAutoRefresh();
    await fetchMetrics();

    const setBtn = document.getElementById('setGcBtn');
    const refreshBtn = document.getElementById('refreshGcBtn');
    const newPercentInput = document.getElementById('newGcPercent');

    setBtn.addEventListener('click', () => {
        let val = parseInt(newPercentInput.value, 10);
        if (isNaN(val)) val = 100;
        setGcPercent(val);
    });
    refreshBtn.addEventListener('click', fetchMetrics);
});