"use strict";

const STATUSES = [
  { id: 0, name: "Todo", desc: "This item hasn't been started" },
  { id: 1, name: "In Progress", desc: "This is actively being worked on" },
  { id: 2, name: "Done", desc: "This has been completed" },
];

// Octicons: issue-opened, dot-fill, issue-closed.
const ICONS = [
  '<svg viewBox="0 0 16 16" width="16" height="16"><path d="M8 9.5a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3Z"/><path d="M8 0a8 8 0 1 1 0 16A8 8 0 0 1 8 0ZM1.5 8a6.5 6.5 0 1 0 13 0 6.5 6.5 0 0 0-13 0Z"/></svg>',
  '<svg viewBox="0 0 16 16" width="16" height="16"><path d="M8 4a4 4 0 1 1 0 8 4 4 0 0 1 0-8Z"/><path d="M8 0a8 8 0 1 1 0 16A8 8 0 0 1 8 0ZM1.5 8a6.5 6.5 0 1 0 13 0 6.5 6.5 0 0 0-13 0Z"/></svg>',
  '<svg viewBox="0 0 16 16" width="16" height="16"><path d="M11.28 6.78a.75.75 0 0 0-1.06-1.06L7.25 8.69 5.78 7.22a.75.75 0 0 0-1.06 1.06l2 2a.75.75 0 0 0 1.06 0l3.5-3.5Z"/><path d="M16 8A8 8 0 1 1 0 8a8 8 0 0 1 16 0Zm-1.5 0a6.5 6.5 0 1 0-13 0 6.5 6.5 0 0 0 13 0Z"/></svg>',
];

const PLUS = '<svg viewBox="0 0 16 16" width="16" height="16"><path d="M7.75 2a.75.75 0 0 1 .75.75V7h4.25a.75.75 0 0 1 0 1.5H8.5v4.25a.75.75 0 0 1-1.5 0V8.5H2.75a.75.75 0 0 1 0-1.5H7V2.75A.75.75 0 0 1 7.75 2Z"/></svg>';

const ORDER_KEY = "tasker.order";

const state = {
  tasks: [],
  order: loadOrder(),
  filter: "",
  dragId: null,
  openId: null,
  addingIn: null,
};

const board = document.getElementById("board");
const summary = document.getElementById("summary");
const panel = document.getElementById("panel");
const toastEl = document.getElementById("toast");

// ---------- API ----------

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) {
    const text = (await res.text()).trim();
    throw new Error(text || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) return null;
  const type = res.headers.get("Content-Type") || "";
  return type.includes("application/json") ? res.json() : null;
}

async function loadTasks() {
  try {
    state.tasks = await api("GET", "/tasks");
    render();
  } catch (err) {
    toast(`Could not load tasks: ${err.message}`);
  }
}

async function createTask(title, description, status) {
  let created = await api("POST", "/tasks", { title, description });
  state.tasks.push(created);
  if (status !== 0) {
    await api("PATCH", `/tasks/${created.ID}`, { status });
    created.Status = status;
    created.DoneAt = status === 2 ? new Date().toISOString() : null;
  }
  render();
}

async function moveTask(id, status, beforeId) {
  const task = findTask(id);
  if (!task) return;

  placeInOrder(id, beforeId);
  if (task.Status === status) {
    render();
    return;
  }

  const previous = { Status: task.Status, DoneAt: task.DoneAt };
  task.Status = status;
  task.DoneAt = status === 2 ? new Date().toISOString() : null;
  render();

  try {
    await api("PATCH", `/tasks/${id}`, { status });
  } catch (err) {
    Object.assign(task, previous);
    render();
    toast(`Could not move item: ${err.message}`);
  }
}

async function deleteTask(id) {
  try {
    await api("DELETE", `/tasks/${id}`);
    state.tasks = state.tasks.filter((t) => t.ID !== id);
    closePanel();
    render();
    toast("Item deleted");
  } catch (err) {
    toast(`Could not delete item: ${err.message}`);
  }
}

// ---------- Ordering (kept per browser; the API has no position field) ----------

function loadOrder() {
  try {
    return JSON.parse(localStorage.getItem(ORDER_KEY)) || [];
  } catch {
    return [];
  }
}

function saveOrder() {
  try {
    localStorage.setItem(ORDER_KEY, JSON.stringify(state.order));
  } catch {
    // Storage unavailable; ordering just won't persist.
  }
}

function placeInOrder(id, beforeId) {
  const order = state.order.filter((x) => x !== id);
  const index = beforeId == null ? -1 : order.indexOf(beforeId);
  if (index === -1) order.push(id);
  else order.splice(index, 0, id);
  state.order = order;
  saveOrder();
}

function sortTasks(tasks) {
  const rank = new Map(state.order.map((id, i) => [id, i]));
  return [...tasks].sort((a, b) => {
    const ra = rank.has(a.ID) ? rank.get(a.ID) : Infinity;
    const rb = rank.has(b.ID) ? rank.get(b.ID) : Infinity;
    return ra === rb ? a.ID - b.ID : ra - rb;
  });
}

// ---------- Rendering ----------

function el(tag, attrs = {}, children = []) {
  const node = document.createElement(tag);
  for (const [key, value] of Object.entries(attrs)) {
    if (value == null || value === false) continue;
    if (key === "class") node.className = value;
    else if (key === "html") node.innerHTML = value;
    else if (key.startsWith("on")) node.addEventListener(key.slice(2), value);
    else node.setAttribute(key, value === true ? "" : value);
  }
  for (const child of [].concat(children)) {
    if (child == null) continue;
    node.append(child instanceof Node ? child : document.createTextNode(child));
  }
  return node;
}

function matchesFilter(task) {
  if (!state.filter) return true;
  const q = state.filter.toLowerCase();
  return (
    task.Title.toLowerCase().includes(q) ||
    task.Description.toLowerCase().includes(q) ||
    `#${task.ID}` === q
  );
}

function render() {
  const visible = sortTasks(state.tasks.filter(matchesFilter));
  board.replaceChildren(...STATUSES.map((s) => renderColumn(s, visible.filter((t) => t.Status === s.id))));

  const total = state.tasks.length;
  summary.textContent = state.filter
    ? `${visible.length} of ${total} items`
    : `${total} ${total === 1 ? "item" : "items"}`;

  renderProgress();
  if (state.openId != null) renderPanel();
}

function renderProgress() {
  const total = state.tasks.length;
  const bar = document.getElementById("progress-bar");
  bar.replaceChildren(
    ...STATUSES.slice(1).map((s) => {
      const count = state.tasks.filter((t) => t.Status === s.id).length;
      return el("span", { class: `status-${s.id}`, style: `width: ${total ? (count / total) * 100 : 0}%` });
    })
  );
  const done = state.tasks.filter((t) => t.Status === 2).length;
  document.getElementById("progress-text").textContent = total ? `${done} of ${total} done` : "No items yet";
}

function renderColumn(status, tasks) {
  const cards = el(
    "div",
    { class: "cards", "data-status": status.id },
    tasks.length
      ? tasks.map(renderCard)
      : el("div", { class: "empty" }, [
          el("span", { html: ICONS[status.id] }),
          state.filter ? "No matching items" : "Drop items here",
        ])
  );

  const column = el("div", { class: `column status-${status.id}` }, [
    el("div", { class: "column-head" }, [
      el("div", { class: "column-title" }, [
        el("span", { class: "status-dot" }),
        status.name,
        el("span", { class: "counter" }, String(tasks.length)),
      ]),
      el("p", { class: "column-desc muted small" }, status.desc),
    ]),
    cards,
    state.addingIn === status.id ? renderAddForm(status.id) : renderAddButton(status.id),
  ]);

  column.addEventListener("dragover", (e) => onDragOver(e, column, cards));
  column.addEventListener("dragleave", (e) => {
    if (!column.contains(e.relatedTarget)) clearDropState(column);
  });
  column.addEventListener("drop", (e) => onDrop(e, column, status.id));
  return column;
}

function renderCard(task) {
  const card = el(
    "article",
    {
      class: `card status-${task.Status}`,
      draggable: "true",
      tabindex: "0",
      "data-id": task.ID,
      ondragstart: (e) => {
        state.dragId = task.ID;
        e.dataTransfer.effectAllowed = "move";
        e.dataTransfer.setData("text/plain", String(task.ID));
        requestAnimationFrame(() => card.classList.add("dragging"));
      },
      ondragend: () => {
        state.dragId = null;
        card.classList.remove("dragging");
        document.querySelectorAll(".column").forEach(clearDropState);
      },
      onclick: () => openPanel(task.ID),
      onkeydown: (e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          openPanel(task.ID);
        }
      },
    },
    [
      el("div", { class: "card-ref" }, [el("span", { html: ICONS[task.Status] }), `tasker #${task.ID}`]),
      el("p", { class: "card-title" }, task.Title),
      task.Description ? el("p", { class: "card-desc" }, task.Description) : null,
      el("div", { class: "card-meta" }, [
        task.Status === 2 && isSet(task.DoneAt)
          ? el("span", { class: "label", title: fullDate(task.DoneAt) }, `Done ${relative(task.DoneAt)}`)
          : el("span", { class: "label", title: fullDate(task.CreatedAt) }, `Opened ${relative(task.CreatedAt)}`),
      ]),
    ]
  );
  return card;
}

function startAdding(status) {
  state.addingIn = status;
  render();
  board.querySelector(".add-form input")?.focus();
}

function renderAddButton(status) {
  return el(
    "button",
    {
      class: "add-btn",
      onclick: () => startAdding(status),
    },
    [el("span", { html: PLUS }), "Add item"]
  );
}

function renderAddForm(status) {
  const title = el("input", { type: "text", placeholder: "Title", maxlength: "200", required: true });
  const description = el("textarea", { rows: "2", placeholder: "Description (optional)" });

  const cancel = () => {
    state.addingIn = null;
    render();
  };

  const form = el(
    "form",
    {
      class: "add-form",
      onsubmit: async (e) => {
        e.preventDefault();
        if (!title.value.trim()) {
          title.focus();
          return;
        }
        form.querySelectorAll("input, textarea, button").forEach((n) => (n.disabled = true));
        try {
          await createTask(title.value.trim(), description.value.trim(), status);
          board.querySelector(".add-form input")?.focus();
        } catch (err) {
          toast(`Could not create item: ${err.message}`);
          render();
        }
      },
      onkeydown: (e) => {
        if (e.key === "Escape") cancel();
        if (e.key === "Enter" && e.target === title) {
          e.preventDefault();
          form.requestSubmit();
        }
      },
    },
    [
      title,
      description,
      el("div", { class: "row" }, [
        el("button", { type: "button", class: "btn", onclick: cancel }, "Cancel"),
        el("button", { type: "submit", class: "btn primary" }, "Add"),
      ]),
    ]
  );
  return form;
}

// ---------- Drag and drop ----------

function onDragOver(e, column, cards) {
  if (state.dragId == null) return;
  e.preventDefault();
  e.dataTransfer.dropEffect = "move";
  column.classList.add("drag-over");

  const before = cardAfterPointer(cards, e.clientY);
  let marker = cards.querySelector(".drop-marker");
  if (!marker) marker = el("div", { class: "drop-marker" });
  if (before) cards.insertBefore(marker, before);
  else cards.appendChild(marker);
}

function onDrop(e, column, status) {
  e.preventDefault();
  const id = state.dragId ?? Number(e.dataTransfer.getData("text/plain"));
  const marker = column.querySelector(".drop-marker");
  const next = marker?.nextElementSibling;
  const beforeId = next?.classList.contains("card") ? Number(next.dataset.id) : null;
  clearDropState(column);
  if (beforeId !== id) moveTask(id, status, beforeId);
}

function cardAfterPointer(cards, y) {
  const candidates = [...cards.querySelectorAll(".card:not(.dragging)")];
  return candidates.find((card) => {
    const box = card.getBoundingClientRect();
    return y < box.top + box.height / 2;
  });
}

function clearDropState(column) {
  column.classList.remove("drag-over");
  column.querySelector(".drop-marker")?.remove();
}

// ---------- Side panel ----------

const panelStatus = document.getElementById("panel-status");
const panelDelete = document.getElementById("panel-delete");

STATUSES.forEach((s) => panelStatus.append(el("option", { value: s.id }, s.name)));

panelStatus.addEventListener("change", () => {
  if (state.openId != null) moveTask(state.openId, Number(panelStatus.value), null);
});

panelDelete.addEventListener("click", () => {
  if (!panelDelete.classList.contains("confirm")) {
    panelDelete.classList.add("confirm");
    panelDelete.textContent = "Click again to delete";
    return;
  }
  deleteTask(state.openId);
});

panel.addEventListener("click", (e) => {
  if (e.target.closest("[data-close]")) closePanel();
});

function openPanel(id) {
  state.openId = id;
  panelDelete.classList.remove("confirm");
  panelDelete.textContent = "Delete item";
  panel.hidden = false;
  renderPanel();
  panelStatus.focus();
}

function closePanel() {
  const id = state.openId;
  state.openId = null;
  panel.hidden = true;
  board.querySelector(`.card[data-id="${id}"]`)?.focus();
}

function renderPanel() {
  const task = findTask(state.openId);
  if (!task) {
    closePanel();
    return;
  }
  document.getElementById("panel-ref").textContent = `tasker #${task.ID}`;
  document.getElementById("panel-title").textContent = task.Title;
  panelStatus.value = String(task.Status);
  document.getElementById("panel-created").textContent = fullDate(task.CreatedAt);
  document.getElementById("panel-done").textContent = isSet(task.DoneAt) ? fullDate(task.DoneAt) : "—";
  const desc = document.getElementById("panel-description");
  desc.textContent = task.Description || "No description provided.";
  desc.classList.toggle("none", !task.Description);
}

// ---------- Helpers ----------

function findTask(id) {
  return state.tasks.find((t) => t.ID === id);
}

// Go encodes a zero time.Time as 0001-01-01T00:00:00Z.
function isSet(value) {
  return value != null && new Date(value).getFullYear() > 1;
}

function fullDate(value) {
  return new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

function relative(value) {
  const seconds = Math.round((Date.now() - new Date(value)) / 1000);
  const units = [
    ["year", 31536000],
    ["month", 2592000],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
  ];
  const fmt = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  for (const [unit, size] of units) {
    if (seconds >= size) return fmt.format(-Math.floor(seconds / size), unit);
  }
  return "just now";
}

let toastTimer;
function toast(message) {
  toastEl.textContent = message;
  toastEl.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toastEl.hidden = true), 3500);
}

// ---------- Init ----------

document.getElementById("new-item").addEventListener("click", () => startAdding(0));

document.getElementById("filter").addEventListener("input", (e) => {
  state.filter = e.target.value.trim();
  render();
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && state.openId != null) closePanel();
  if (e.key === "/" && !e.target.closest("input, textarea, select")) {
    e.preventDefault();
    document.getElementById("filter").focus();
  }
});

window.addEventListener("focus", () => {
  if (state.dragId == null && state.addingIn == null) loadTasks();
});

loadTasks();
