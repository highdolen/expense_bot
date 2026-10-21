const PAGE_SIZE = 100;

const table = document.getElementById("expensesTable");
const totalAmount = document.getElementById("totalAmount");
const expenseCount = document.getElementById("expenseCount");
const userCount = document.getElementById("userCount");
const status = document.getElementById("status");
const error = document.getElementById("error");
const formError = document.getElementById("formError");
const refreshButton = document.getElementById("refreshButton");

const filterForm = document.getElementById("filterForm");
const filterUser = document.getElementById("filterUser");
const filterCategory = document.getElementById("filterCategory");
const filterFrom = document.getElementById("filterFrom");
const filterTo = document.getElementById("filterTo");
const resetFiltersButton = document.getElementById("resetFiltersButton");

const expenseForm = document.getElementById("expenseForm");
const expenseFormTitle = document.getElementById("expenseFormTitle");
const expenseUser = document.getElementById("expenseUser");
const expenseAmount = document.getElementById("expenseAmount");
const expenseCategory = document.getElementById("expenseCategory");
const expenseDate = document.getElementById("expenseDate");
const expenseComment = document.getElementById("expenseComment");
const saveExpenseButton = document.getElementById("saveExpenseButton");
const cancelEditButton = document.getElementById("cancelEditButton");

const moneyFormatter = new Intl.NumberFormat("ru-RU", {
    style: "currency",
    currency: "RUB",
});

const dateFormatter = new Intl.DateTimeFormat("ru-RU", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    timeZone: "UTC",
});

const categoryNames = {
    food: "Еда",
    transport: "Транспорт",
    housing: "Жильё",
    entertainment: "Развлечения",
    health: "Здоровье",
    shopping: "Покупки",
    other: "Другое",
};

let users = [];
let expenses = [];
let editingExpenseID = null;

function escapeHTML(value) {
    return String(value ?? "").replace(/[&<>"']/g, (char) => ({
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        '"': "&quot;",
        "'": "&#39;",
    })[char]);
}

async function requestJSON(url, options = {}) {
    const response = await fetch(url, {
        ...options,
        headers: {
            ...(options.body ? { "Content-Type": "application/json" } : {}),
            ...options.headers,
        },
    });

    if (!response.ok) {
        let message = `Ошибка HTTP ${response.status}`;

        try {
            const result = await response.json();
            message = result.error?.message || result.message || message;
        } catch {
            // Ответ сервера может быть не в формате JSON.
        }

        throw new Error(message);
    }

    if (response.status === 204) {
        return null;
    }

    return response.json();
}

function getUserName(user) {
    if (user.first_name && user.first_name.trim()) {
        return user.first_name;
    }

    if (user.username && user.username.trim()) {
        return `@${user.username}`;
    }

    return `Пользователь #${user.id}`;
}

function formatDate(value) {
    if (!value) {
        return "—";
    }

    const date = new Date(`${String(value).slice(0, 10)}T00:00:00Z`);

    if (Number.isNaN(date.getTime())) {
        return escapeHTML(value);
    }

    return dateFormatter.format(date);
}

function formatMoney(value) {
    const amount = Number(value);

    if (!Number.isFinite(amount)) {
        return "—";
    }

    return moneyFormatter.format(amount);
}

function getToday() {
    const now = new Date();
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, "0");
    const day = String(now.getDate()).padStart(2, "0");

    return `${year}-${month}-${day}`;
}

function showError(element, message) {
    element.textContent = message;
    element.hidden = false;
}

function clearError(element) {
    element.textContent = "";
    element.hidden = true;
}

function fillUserSelects() {
    const options = users.map((user) => `
        <option value="${escapeHTML(user.id)}">
            ${escapeHTML(getUserName(user))}
        </option>
    `).join("");

    filterUser.innerHTML = `
        <option value="">Все участники</option>
        ${options}
    `;

    expenseUser.innerHTML = `
        <option value="">Выбери участника</option>
        ${options}
    `;
}

async function loadExpenses() {
    const resultExpenses = [];
    let offset = 0;
    let total = 0;

    while (true) {
        const params = new URLSearchParams({
            limit: String(PAGE_SIZE),
            offset: String(offset),
        });

        if (filterUser.value) {
            params.set("user_id", filterUser.value);
        }

        if (filterCategory.value) {
            params.set("category", filterCategory.value);
        }

        if (filterFrom.value) {
            params.set("from", filterFrom.value);
        }

        if (filterTo.value) {
            params.set("to", filterTo.value);
        }

        const result = await requestJSON(`/api/expenses?${params}`);

        if (!Array.isArray(result.items)) {
            throw new Error("Сервер вернул неожиданный формат расходов");
        }

        if (offset === 0) {
            // Сервер возвращает общую сумму всех записей,
            // соответствующих фильтрам, а не только текущей странице.
            total = Number(result.total) || 0;
        }

        resultExpenses.push(...result.items);

        if (result.items.length === 0 || result.items.length < PAGE_SIZE) {
            break;
        }

        offset += result.items.length;
    }

    return {
        expenses: resultExpenses,
        total,
    };
}

function renderExpenses() {
    const usersByID = new Map(
        users.map((user) => [Number(user.id), getUserName(user)])
    );

    if (expenses.length === 0) {
        table.innerHTML = `
            <tr>
                <td colspan="6" class="empty">
                    По заданным условиям расходы не найдены
                </td>
            </tr>
        `;
        return;
    }

    table.innerHTML = expenses.map((expense) => {
        const userName = usersByID.get(Number(expense.user_id))
            ?? `Пользователь #${expense.user_id}`;

        const categoryName = categoryNames[expense.category]
            ?? expense.category
            ?? "Без категории";

        return `
            <tr>
                <td>${formatDate(expense.spent_at)}</td>
                <td>${escapeHTML(userName)}</td>
                <td>${escapeHTML(expense.comment || "Без комментария")}</td>
                <td>
                    <span class="category">
                        ${escapeHTML(categoryName)}
                    </span>
                </td>
                <td class="amount amount-column">
                    ${formatMoney(expense.amount)}
                </td>
                <td>
                    <div class="actions">
                        <button
                            class="button button-light button-small"
                            type="button"
                            data-action="edit"
                            data-id="${escapeHTML(expense.id)}"
                        >
                            Изменить
                        </button>
                        <button
                            class="button button-danger button-small"
                            type="button"
                            data-action="delete"
                            data-id="${escapeHTML(expense.id)}"
                        >
                            Удалить
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join("");
}

async function refreshDashboard() {
    refreshButton.disabled = true;
    status.textContent = "Загрузка...";
    clearError(error);

    try {
        const [expenseData, loadedUsers] = await Promise.all([
            loadExpenses(),
            requestJSON("/api/users"),
        ]);

        if (!Array.isArray(loadedUsers)) {
            throw new Error("Сервер вернул неожиданный формат пользователей");
        }

        expenses = expenseData.expenses;
        users = loadedUsers;

        fillUserSelects();
        renderExpenses();

        // После обновления сохраняем выбранного участника фильтра,
        // если он всё ещё присутствует в списке.
        if (filterUser.dataset.selected) {
            filterUser.value = filterUser.dataset.selected;
            delete filterUser.dataset.selected;
        }

        totalAmount.textContent = formatMoney(expenseData.total);
        expenseCount.textContent = new Intl.NumberFormat("ru-RU")
            .format(expenses.length);
        userCount.textContent = new Intl.NumberFormat("ru-RU")
            .format(users.length);

        status.textContent = "Данные обновлены";
    } catch (err) {
        console.error("Не удалось загрузить панель:", err);
        status.textContent = "Ошибка загрузки";
        showError(
            error,
            `Не удалось загрузить данные: ${err.message}. Попробуйте обновить страницу.`
        );

        table.innerHTML = `
            <tr>
                <td colspan="6" class="empty">
                    Не удалось загрузить расходы
                </td>
            </tr>
        `;
    } finally {
        refreshButton.disabled = false;
    }
}

function resetExpenseForm() {
    editingExpenseID = null;
    expenseForm.reset();

    expenseFormTitle.textContent = "Добавить расход";
    saveExpenseButton.textContent = "Добавить расход";
    cancelEditButton.hidden = true;

    expenseDate.value = getToday();
    clearError(formError);
}

function startEditing(expense) {
    editingExpenseID = Number(expense.id);

    expenseFormTitle.textContent = "Редактировать расход";
    saveExpenseButton.textContent = "Сохранить изменения";
    cancelEditButton.hidden = false;

    expenseUser.value = String(expense.user_id);
    expenseUser.disabled = true;

    expenseAmount.value = (Number(expense.amount) || 0).toFixed(2);
    expenseCategory.value = expense.category;
    expenseDate.value = String(expense.spent_at).slice(0, 10);
    expenseComment.value = expense.comment || "";

    clearError(formError);

    expenseFormTitle.scrollIntoView({
        behavior: "smooth",
        block: "center",
    });
}

async function saveExpense(event) {
    event.preventDefault();
    clearError(formError);

    const amountText = expenseAmount.value.trim();

    if (!/^\d+(\.\d{1,2})?$/.test(amountText)) {
        showError(formError, "Укажи сумму с максимум двумя знаками после запятой.");
        return;
    }

    const amount = Number(amountText);

    if (!Number.isFinite(amount) || amount <= 0) {
        showError(formError, "Сумма должна быть больше нуля.");
        return;
    }

    if (!expenseUser.value) {
        showError(formError, "Выбери участника семьи.");
        return;
    }

    const payload = {
        amount,
        category: expenseCategory.value,
        spent_at: expenseDate.value,
        comment: expenseComment.value.trim(),
    };

    let url = "/api/expenses";
    let method = "POST";

    if (editingExpenseID !== null) {
        url = `/api/expenses/${editingExpenseID}`;
        method = "PUT";
    } else {
        payload.user_id = Number(expenseUser.value);
    }

    saveExpenseButton.disabled = true;

    try {
        await requestJSON(url, {
            method,
            body: JSON.stringify(payload),
        });

        resetExpenseForm();
        expenseUser.disabled = false;
        await refreshDashboard();
    } catch (err) {
        console.error("Не удалось сохранить расход:", err);
        showError(formError, `Не удалось сохранить расход: ${err.message}`);
    } finally {
        saveExpenseButton.disabled = false;
    }
}

async function deleteExpense(expense) {
    const confirmed = confirm(
        `Удалить расход "${expense.comment || "Без комментария"}" на сумму ${formatMoney(expense.amount)}?`
    );

    if (!confirmed) {
        return;
    }

    status.textContent = "Удаление...";
    clearError(error);

    try {
        await requestJSON(`/api/expenses/${expense.id}`, {
            method: "DELETE",
        });

        if (editingExpenseID === Number(expense.id)) {
            resetExpenseForm();
            expenseUser.disabled = false;
        }

        await refreshDashboard();
    } catch (err) {
        console.error("Не удалось удалить расход:", err);
        status.textContent = "Ошибка удаления";
        showError(error, `Не удалось удалить расход: ${err.message}`);
    }
}

filterForm.addEventListener("submit", async (event) => {
    event.preventDefault();

    if (filterFrom.value && filterTo.value && filterFrom.value > filterTo.value) {
        showError(error, "Начальная дата не может быть позже конечной.");
        return;
    }

    filterUser.dataset.selected = filterUser.value;
    await refreshDashboard();
});

resetFiltersButton.addEventListener("click", async () => {
    filterForm.reset();
    delete filterUser.dataset.selected;
    await refreshDashboard();
});

expenseForm.addEventListener("submit", saveExpense);

cancelEditButton.addEventListener("click", () => {
    resetExpenseForm();
    expenseUser.disabled = false;
});

refreshButton.addEventListener("click", refreshDashboard);

table.addEventListener("click", async (event) => {
    const button = event.target.closest("button[data-action]");

    if (!button) {
        return;
    }

    const id = Number(button.dataset.id);
    const expense = expenses.find((item) => Number(item.id) === id);

    if (!expense) {
        showError(error, "Не удалось найти выбранный расход. Обнови страницу.");
        return;
    }

    if (button.dataset.action === "edit") {
        startEditing(expense);
    }

    if (button.dataset.action === "delete") {
        await deleteExpense(expense);
    }
});

expenseDate.value = getToday();
refreshDashboard();
