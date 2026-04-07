// DOM элементы
const sumA = document.getElementById('sumA');
const sumB = document.getElementById('sumB');
const sumBtn = document.getElementById('sumBtn');
const sumResult = document.getElementById('sumResult');

const fibN = document.getElementById('fibN');
const fibBtn = document.getElementById('fibBtn');
const fibResult = document.getElementById('fibResult');

const allocSize = document.getElementById('allocSize');
const allocBtn = document.getElementById('allocBtn');
const allocResult = document.getElementById('allocResult');

// Общая обработка ошибок
async function handleResponse(response) {
    const data = await response.json();
    if (!response.ok) {
        throw new Error(data.error || `HTTP ${response.status}`);
    }
    return data;
}

// Сумма
sumBtn.addEventListener('click', async () => {
    const a = parseInt(sumA.value, 10);
    const b = parseInt(sumB.value, 10);
    if (isNaN(a) || isNaN(b)) {
        sumResult.textContent = 'Ошибка: введите числа';
        return;
    }
    try {
        const resp = await fetch(`/sum?a=${a}&b=${b}`);
        const data = await handleResponse(resp);
        sumResult.textContent = data.result;
    } catch (err) {
        sumResult.textContent = `Ошибка: ${err.message}`;
    }
});

// Фибоначчи
fibBtn.addEventListener('click', async () => {
    let n = parseInt(fibN.value, 10);
    if (isNaN(n) || n < 1 || n > 45) {
        fibResult.textContent = 'Ошибка: n должно быть от 1 до 45';
        return;
    }
    try {
        const resp = await fetch(`/fib?n=${n}`);
        const data = await handleResponse(resp);
        fibResult.textContent = data.result;
    } catch (err) {
        fibResult.textContent = `Ошибка: ${err.message}`;
    }
});

// Выделение памяти
allocBtn.addEventListener('click', async () => {
    let size = parseInt(allocSize.value, 10);
    if (isNaN(size) || size <= 0 || size > 100_000_000) {
        allocResult.textContent = 'Ошибка: размер должен быть от 1 до 100 млн';
        return;
    }
    try {
        const resp = await fetch('/allocate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ size })
        });
        const data = await handleResponse(resp);
        const allocated = data.result.allocated_bytes;
        allocResult.textContent = `${allocated.toLocaleString()} байт`;
    } catch (err) {
        allocResult.textContent = `Ошибка: ${err.message}`;
    }
});