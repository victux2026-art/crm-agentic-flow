const state = {
  token: localStorage.getItem("crmflow_token") || "",
  session: JSON.parse(localStorage.getItem("crmflow_session") || "null"),
  organizations: [],
  deals: [],
  people: [],
  tasks: [],
  outboxFailed: [],
  deliveryFailed: [],
  audit: [],
  selectedOrganizationId: null,
  editing: {
    organizationId: null,
    dealId: null,
    personId: null,
    taskId: null,
  },
};

const authPanel = document.getElementById("auth-panel");
const appPanel = document.getElementById("app-panel");
const loginForm = document.getElementById("login-form");
const loginError = document.getElementById("login-error");
const sessionBadge = document.getElementById("session-badge");
const workerLed = document.getElementById("worker-led");
const workerStatusText = document.getElementById("worker-status-text");
const commandSearch = document.getElementById("command-search");

const organizationForm = document.getElementById("organization-form");
const organizationResult = document.getElementById("organization-result");
const organizationSubmit = document.getElementById("organization-submit");
const organizationCancel = document.getElementById("organization-cancel");

const dealForm = document.getElementById("deal-form");
const dealResult = document.getElementById("deal-result");
const dealOrganizationSelect = document.getElementById("deal-organization-id");
const dealSubmit = document.getElementById("deal-submit");
const dealCancel = document.getElementById("deal-cancel");

const personForm = document.getElementById("person-form");
const personResult = document.getElementById("person-result");
const personOrganizationSelect = document.getElementById("person-organization-id");
const personSubmit = document.getElementById("person-submit");
const personCancel = document.getElementById("person-cancel");

const taskForm = document.getElementById("task-form");
const taskResult = document.getElementById("task-result");
const taskOrganizationSelect = document.getElementById("task-organization-id");
const taskDealSelect = document.getElementById("task-deal-id");
const taskSubmit = document.getElementById("task-submit");
const taskCancel = document.getElementById("task-cancel");

function setSession(token, session) {
  state.token = token;
  state.session = session;
  localStorage.setItem("crmflow_token", token);
  localStorage.setItem("crmflow_session", JSON.stringify(session));
}

function clearSession() {
  state.token = "";
  state.session = null;
  localStorage.removeItem("crmflow_token");
  localStorage.removeItem("crmflow_session");
}

async function api(path, options = {}) {
  const headers = new Headers(options.headers || {});
  if (state.token) {
    headers.set("Authorization", `Bearer ${state.token}`);
  }
  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(path, { ...options, headers });
  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `${response.status}`);
  }
  if (response.status === 204) {
    return null;
  }
  return response.json();
}

function renderStatusCards(targetId, items) {
  const target = document.getElementById(targetId);
  target.innerHTML = "";
  items.forEach((item) => {
    const card = document.createElement("div");
    card.className = "mini-card";
    card.innerHTML = `<span class="muted">${item.status}</span><strong>${item.count}</strong>`;
    target.appendChild(card);
  });
}

function renderRankedList(targetId, items, labelKey) {
  const target = document.getElementById(targetId);
  target.innerHTML = "";
  items.forEach((item) => {
    const row = document.createElement("div");
    row.className = "list-row";
    row.innerHTML = `<span>${item[labelKey]}</span><strong>${item.count}</strong>`;
    target.appendChild(row);
  });
}

function setWorkerStatus(isLive) {
  workerLed.style.background = isLive ? "var(--ok)" : "var(--bad)";
  workerStatusText.textContent = isLive ? "Worker Live" : "Worker Idle";
}

function renderPipeline(deals) {
  const board = document.getElementById("pipeline-board");
  const columns = [
    { id: "lead", label: "Lead" },
    { id: "proposal", label: "Proposal" },
    { id: "negotiation", label: "Negotiation" },
    { id: "won", label: "Won" },
  ];

  board.innerHTML = "";

  columns.forEach((column) => {
    const columnDeals = deals.filter((deal) => deal.stage === column.id || (column.id === "lead" && !["proposal", "negotiation", "won"].includes(deal.stage)));
    const node = document.createElement("div");
    node.className = "pipeline-column";
    node.innerHTML = `
      <header>
        <strong>${column.label}</strong>
        <span class="column-count">${columnDeals.length}</span>
      </header>
      <div class="pipeline-stack">
        ${columnDeals.map((deal) => {
          const healthClass = deal.status === "won" ? "health-good" : deal.stage === "negotiation" ? "health-warm" : "health-cold";
          const organization = state.organizations.find((item) => item.id === deal.organization_id);
          return `
            <article class="pipeline-card">
              <span class="pipeline-health ${healthClass}"></span>
              <div class="deal-topline">
                <strong>${deal.name}</strong>
                <span class="ia-badge">spark AI</span>
              </div>
              <p class="muted small">${organization ? organization.name : "No organization"}</p>
              <div class="entity-meta">
                <span class="pill">${deal.value_currency} ${Number(deal.value_amount || 0).toLocaleString()}</span>
                <span class="pill">${deal.status}</span>
              </div>
              <div class="quick-actions">
                <button type="button" class="ghost compact-action" data-edit-type="deal" data-edit-id="${deal.id}">Move Stage</button>
                <button type="button" class="ghost compact-action" data-edit-type="task-from-deal" data-edit-id="${deal.id}">Log Activity</button>
              </div>
            </article>`;
        }).join("") || `<div class="mini-card"><span class="muted small">No deals in this stage.</span></div>`}
      </div>
    `;
    board.appendChild(node);
  });
}

function renderEntityCards(targetId, items, formatter) {
  const target = document.getElementById(targetId);
  target.innerHTML = "";
  if (!items.length) {
    target.innerHTML = `<div class="event-card"><span class="muted">No items yet.</span></div>`;
    return;
  }

  items.forEach((item) => {
    const card = document.createElement("div");
    card.className = `entity-card${targetId === "organization-list" && state.selectedOrganizationId === item.id ? " selected" : ""}`;
    card.innerHTML = formatter(item);
    if (targetId === "organization-list") {
      card.dataset.organizationId = item.id;
    }
    target.appendChild(card);
  });
}

function renderOrganizations() {
  renderEntityCards("organization-list", state.organizations, (item) => `
    <strong>${item.name}</strong>
    <div class="muted small">${item.domain || "no domain"} • ${item.industry || "no industry"}</div>
    <div class="entity-meta">
      <span class="pill">id ${item.id}</span>
      <span class="pill">${new Date(item.created_at).toLocaleDateString()}</span>
    </div>
    <div class="entity-card-actions">
      <button type="button" class="ghost small-button" data-edit-type="organization" data-edit-id="${item.id}">Edit</button>
      <button type="button" class="ghost small-button" data-delete-type="organization" data-delete-id="${item.id}">Delete</button>
    </div>
  `);

  const selected = state.organizations.find((item) => item.id === state.selectedOrganizationId) || state.organizations[0];
  if (selected && state.selectedOrganizationId == null) {
    state.selectedOrganizationId = selected.id;
  }
  renderOrganizationDetail(selected || null);
}

function renderDeals() {
  renderEntityCards("deal-list", state.deals, (item) => {
    const organization = state.organizations.find((org) => org.id === item.organization_id);
    return `
      <strong>${item.name}</strong>
      <div class="muted small">${organization ? organization.name : "No organization"} • ${item.stage} • ${item.status}</div>
      <div class="entity-meta">
        <span class="pill">${item.value_currency} ${Number(item.value_amount || 0).toLocaleString()}</span>
        <span class="pill">id ${item.id}</span>
      </div>
      <div class="entity-card-actions">
        <button type="button" class="ghost small-button" data-edit-type="deal" data-edit-id="${item.id}">Edit</button>
        <button type="button" class="ghost small-button" data-delete-type="deal" data-delete-id="${item.id}">Delete</button>
      </div>
    `;
  });
}

function renderPeople() {
  renderEntityCards("person-list", state.people, (item) => {
    const organization = state.organizations.find((org) => org.id === item.organization_id);
    return `
      <strong>${item.first_name} ${item.last_name}</strong>
      <div class="muted small">${item.email || "no email"} • ${organization ? organization.name : "No organization"}</div>
      <div class="entity-meta">
        <span class="pill">${item.status}</span>
        <span class="pill">id ${item.id}</span>
      </div>
      <div class="entity-card-actions">
        <button type="button" class="ghost small-button" data-edit-type="person" data-edit-id="${item.id}">Edit</button>
        <button type="button" class="ghost small-button" data-delete-type="person" data-delete-id="${item.id}">Delete</button>
      </div>
    `;
  });
}

function renderTasks() {
  renderEntityCards("task-list", state.tasks, (item) => {
    const organization = state.organizations.find((org) => org.id === item.organization_id);
    const deal = state.deals.find((entry) => entry.id === item.deal_id);
    return `
      <strong>${item.title}</strong>
      <div class="muted small">${organization ? organization.name : "No organization"}${deal ? " • " + deal.name : ""}</div>
      <div class="entity-meta">
        <span class="pill">${item.status}</span>
        <span class="pill">${item.priority}</span>
        <span class="pill">id ${item.id}</span>
      </div>
      <div class="entity-card-actions">
        <button type="button" class="ghost small-button" data-edit-type="task" data-edit-id="${item.id}">Edit</button>
        <button type="button" class="ghost small-button" data-delete-type="task" data-delete-id="${item.id}">Delete</button>
      </div>
    `;
  });
}

function renderEventList(targetId, items, kind) {
  const target = document.getElementById(targetId);
  target.innerHTML = "";
  if (!items.length) {
    target.innerHTML = `<div class="event-card"><span class="muted">No items.</span></div>`;
    return;
  }

  items.forEach((item) => {
    const replayButton = kind === "outbox"
      ? `<button data-outbox-id="${item.id}" class="secondary compact-action">Replay</button>`
      : `<button data-delivery-id="${item.id}" class="secondary compact-action">Replay</button>`;
    const stateClass = item.status === "failed" ? "failed" : item.status;
    const statusLabel = item.status === "pending" && item.attempt_count > 0 ? `retrying ${item.attempt_count}/5` : item.status;
    const body = JSON.stringify(item.payload || item, null, 2);

    const detailLink = kind === "delivery" && item.outbox_event_id
      ? `<a href="#outbox-monitor" class="timeline-link" data-focus-outbox="${item.outbox_event_id}">Open Event ${item.outbox_event_id}</a>`
      : "";

    target.innerHTML += `
      <article class="event-card">
        <header>
          <div>
            <strong>${kind === "outbox" ? item.event_type : `delivery #${item.id}`}</strong>
            <div class="muted small">id ${item.id} • attempts ${item.attempt_count || 0}</div>
          </div>
          <span class="status ${stateClass}">${statusLabel}</span>
        </header>
        ${detailLink}
        <pre>${body}</pre>
        <div class="event-actions">${replayButton}</div>
      </article>`;
  });
}

function renderAudit() {
  const target = document.getElementById("audit-list");
  target.innerHTML = "";
  if (!state.audit.length) {
    target.innerHTML = `<div class="event-card"><span class="muted">No audit entries.</span></div>`;
    return;
  }

  state.audit.forEach((item) => {
    target.innerHTML += `
      <article class="event-card">
        <header>
          <div>
            <strong>${item.entity_type}</strong>
            <div class="muted small">${item.action} • entity ${item.entity_id}</div>
          </div>
          <span class="muted small">${new Date(item.created_at).toLocaleString()}</span>
        </header>
      </article>`;
  });
}

function buildTimeline(organization) {
  const items = [];

  state.people
    .filter((item) => item.organization_id === organization.id)
    .forEach((item) => {
      items.push({
        ts: item.created_at,
        type: "sales",
        title: `${item.first_name} ${item.last_name} added`,
        detail: item.email || "new contact",
      });
    });

  state.deals
    .filter((item) => item.organization_id === organization.id)
    .forEach((item) => {
      items.push({
        ts: item.created_at,
        type: "sales",
        title: `${item.name} opened`,
        detail: `${item.stage} • ${item.status}`,
      });
    });

  state.tasks
    .filter((item) => item.organization_id === organization.id)
    .forEach((item) => {
      items.push({
        ts: item.created_at,
        type: "sales",
        title: `Task: ${item.title}`,
        detail: `${item.status} • ${item.priority}`,
      });
    });

  state.deliveryFailed
    .slice(0, 5)
    .forEach((item) => {
      items.push({
        ts: item.created_at,
        type: "ops",
        title: `Webhook ${item.status}`,
        detail: item.last_error || `delivery ${item.id}`,
        link: item.outbox_event_id ? `<a class="timeline-link" href="#webhooks-monitor" data-delivery-id="${item.id}">Delivery Detail</a>` : "",
      });
    });

  items.sort((a, b) => new Date(b.ts) - new Date(a.ts));
  if (!items.length) {
    return `<span class="muted small">No timeline entries yet.</span>`;
  }

  return items.slice(0, 10).map((item) => `
    <div class="timeline-item">
      <span class="timeline-node ${item.type}">${item.type === "sales" ? "S" : "O"}</span>
      <div>
        <strong>${item.title}</strong>
        <div class="muted small">${item.detail}</div>
        ${item.link || ""}
      </div>
    </div>`).join("");
}

function renderOrganizationDetail(organization) {
  const detail = document.getElementById("organization-detail");
  if (!organization) {
    detail.innerHTML = `<span class="muted">Select an organization from the list.</span>`;
    return;
  }

  const relatedDeals = state.deals.filter((deal) => deal.organization_id === organization.id);
  const relatedPeople = state.people.filter((person) => person.organization_id === organization.id);
  const relatedTasks = state.tasks.filter((task) => task.organization_id === organization.id);

  const compact = (items, emptyText, formatter) => {
    if (!items.length) {
      return `<span class="muted small">${emptyText}</span>`;
    }
    return items.map(formatter).join("");
  };

  detail.innerHTML = `
    <div class="detail-grid">
      <div>
        <p class="eyebrow">Organization</p>
        <h3>${organization.name}</h3>
        <p class="muted">${organization.domain || "No domain"} • ${organization.industry || "No industry"}</p>
        <div class="entity-meta">
          <span class="pill">owner ${organization.owner_user_id || "n/a"}</span>
          <span class="pill">deals ${relatedDeals.length}</span>
          <span class="pill">people ${relatedPeople.length}</span>
          <span class="pill">tasks ${relatedTasks.length}</span>
        </div>
      </div>

      <div class="detail-section">
        <h4>Unified Timeline</h4>
        <div class="timeline">${buildTimeline(organization)}</div>
      </div>

      <div class="detail-section">
        <h4>People</h4>
        <div class="compact-list">
          ${compact(relatedPeople, "No people for this organization.", (item) => `
            <div class="compact-row">
              <span>${item.first_name} ${item.last_name}</span>
              <span class="muted small">${item.email || "no email"}</span>
            </div>`)}
        </div>
      </div>

      <div class="detail-section">
        <h4>Deals</h4>
        <div class="compact-list">
          ${compact(relatedDeals, "No deals for this organization.", (item) => `
            <div class="compact-row">
              <span>${item.name}</span>
              <span class="muted small">${item.stage} • ${item.status}</span>
            </div>`)}
        </div>
      </div>

      <div class="detail-section">
        <h4>Tasks</h4>
        <div class="compact-list">
          ${compact(relatedTasks, "No tasks for this organization.", (item) => `
            <div class="compact-row">
              <span>${item.title}</span>
              <span class="muted small">${item.status} • ${item.priority}</span>
            </div>`)}
        </div>
      </div>

      <div class="detail-section">
        <h4>Metadata</h4>
        <pre>${JSON.stringify(organization.metadata || {}, null, 2)}</pre>
      </div>
    </div>`;
}

function populateOrganizationSelect() {
  const selects = [dealOrganizationSelect, personOrganizationSelect, taskOrganizationSelect];
  selects.forEach((select) => {
    select.innerHTML = '<option value="">Select organization</option>';
    state.organizations.forEach((item) => {
      const option = document.createElement("option");
      option.value = item.id;
      option.textContent = item.name;
      if (state.selectedOrganizationId === item.id) {
        option.selected = true;
      }
      select.appendChild(option);
    });
  });
}

function populateDealSelect() {
  taskDealSelect.innerHTML = '<option value="">Optional deal</option>';
  state.deals.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.name;
    taskDealSelect.appendChild(option);
  });
}

function setEditMode(kind, id) {
  state.editing[`${kind}Id`] = id;
  const submitMap = {
    organization: organizationSubmit,
    deal: dealSubmit,
    person: personSubmit,
    task: taskSubmit,
  };
  const cancelMap = {
    organization: organizationCancel,
    deal: dealCancel,
    person: personCancel,
    task: taskCancel,
  };
  submitMap[kind].textContent = `Save ${kind.charAt(0).toUpperCase() + kind.slice(1)}`;
  cancelMap[kind].classList.remove("hidden");
}

function clearEditMode(kind) {
  state.editing[`${kind}Id`] = null;
  const submitText = {
    organization: "Create",
    deal: "Create Deal",
    person: "Create Person",
    task: "Create Task",
  };
  const submitMap = {
    organization: organizationSubmit,
    deal: dealSubmit,
    person: personSubmit,
    task: taskSubmit,
  };
  const cancelMap = {
    organization: organizationCancel,
    deal: dealCancel,
    person: personCancel,
    task: taskCancel,
  };
  submitMap[kind].textContent = submitText[kind];
  cancelMap[kind].classList.add("hidden");
}

function startEditOrganization(id) {
  const item = state.organizations.find((entry) => entry.id === id);
  if (!item) return;
  organizationForm.elements.namedItem("name").value = item.name || "";
  organizationForm.elements.namedItem("domain").value = item.domain || "";
  organizationForm.elements.namedItem("industry").value = item.industry || "";
  organizationResult.textContent = `Editing organization ${item.name}.`;
  setEditMode("organization", id);
}

function startEditDeal(id) {
  const item = state.deals.find((entry) => entry.id === id);
  if (!item) return;
  dealForm.elements.namedItem("name").value = item.name || "";
  dealOrganizationSelect.value = item.organization_id || "";
  dealForm.elements.namedItem("stage").value = item.stage || "";
  dealForm.elements.namedItem("value_amount").value = item.value_amount || "";
  dealForm.elements.namedItem("value_currency").value = item.value_currency || "";
  dealResult.textContent = `Editing deal ${item.name}.`;
  setEditMode("deal", id);
}

function startEditPerson(id) {
  const item = state.people.find((entry) => entry.id === id);
  if (!item) return;
  personOrganizationSelect.value = item.organization_id || "";
  personForm.elements.namedItem("first_name").value = item.first_name || "";
  personForm.elements.namedItem("last_name").value = item.last_name || "";
  personForm.elements.namedItem("email").value = item.email || "";
  personResult.textContent = `Editing person ${item.first_name} ${item.last_name}.`;
  setEditMode("person", id);
}

function startEditTask(id) {
  const item = state.tasks.find((entry) => entry.id === id);
  if (!item) return;
  taskForm.elements.namedItem("title").value = item.title || "";
  taskOrganizationSelect.value = item.organization_id || "";
  taskDealSelect.value = item.deal_id || "";
  taskForm.elements.namedItem("priority").value = item.priority || "";
  taskForm.elements.namedItem("status").value = item.status || "";
  taskResult.textContent = `Editing task ${item.title}.`;
  setEditMode("task", id);
}

function primeTaskFromDeal(id) {
  const deal = state.deals.find((entry) => entry.id === id);
  if (!deal) return;
  taskOrganizationSelect.value = deal.organization_id || "";
  taskDealSelect.value = deal.id;
  taskForm.elements.namedItem("title").value = `Follow up ${deal.name}`;
  taskForm.elements.namedItem("priority").value = "high";
  taskForm.elements.namedItem("status").value = "open";
  taskResult.textContent = `Task prefilled from deal ${deal.name}.`;
  document.getElementById("tasks").scrollIntoView({ behavior: "smooth", block: "start" });
}

async function deleteEntity(kind, id) {
  const routeMap = {
    organization: "organizations",
    deal: "deals",
    person: "people",
    task: "tasks",
  };
  await api(`/${routeMap[kind]}/${id}`, { method: "DELETE" });
  await loadDashboard();
}

async function loadDashboard() {
  const [outboxStats, deliveryStats, failedOutbox, failedDeliveries, audit, organizations, deals, people, tasks] = await Promise.all([
    api("/outbox-events/stats"),
    api("/webhook-deliveries/stats"),
    api("/outbox-events?status=failed&limit=10"),
    api("/webhook-deliveries?status=failed"),
    api("/audit-log?limit=10"),
    api("/organizations"),
    api("/deals"),
    api("/people"),
    api("/tasks"),
  ]);

  state.organizations = organizations || [];
  state.deals = deals || [];
  state.people = people || [];
  state.tasks = tasks || [];
  state.outboxFailed = failedOutbox || [];
  state.deliveryFailed = failedDeliveries || [];
  state.audit = audit || [];

  if (state.selectedOrganizationId != null && !state.organizations.find((item) => item.id === state.selectedOrganizationId)) {
    state.selectedOrganizationId = null;
  }

  renderStatusCards("outbox-status-cards", outboxStats.by_status || []);
  renderStatusCards("delivery-status-cards", deliveryStats.by_status || []);
  renderRankedList("event-type-list", outboxStats.by_event_type || [], "event_type");
  renderRankedList("endpoint-list", deliveryStats.by_endpoint || [], "endpoint_name");
  renderPipeline(state.deals);
  populateOrganizationSelect();
  populateDealSelect();
  renderOrganizations();
  renderDeals();
  renderPeople();
  renderTasks();
  renderEventList("outbox-failed-list", state.outboxFailed, "outbox");
  renderEventList("delivery-failed-list", state.deliveryFailed, "delivery");
  renderAudit();
  setWorkerStatus((outboxStats.by_status || []).length > 0);
}

function showApp() {
  authPanel.classList.add("hidden");
  appPanel.classList.remove("hidden");
  sessionBadge.textContent = `${state.session.email} @ ${state.session.tenant_slug}`;
}

function showLogin() {
  appPanel.classList.add("hidden");
  authPanel.classList.remove("hidden");
}

async function submitOrganization(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(organizationForm).entries());
  try {
    const isEdit = state.editing.organizationId != null;
    const result = await api(isEdit ? `/organizations/${state.editing.organizationId}` : "/organizations", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    organizationResult.textContent = `${isEdit ? "Updated" : "Created"} organization ${result.name}.`;
    organizationForm.reset();
    clearEditMode("organization");
    await loadDashboard();
  } catch (error) {
    organizationResult.textContent = error.message;
  }
}

async function submitDeal(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(dealForm).entries());
  body.organization_id = Number(body.organization_id);
  body.value_amount = Number(body.value_amount || 0);
  try {
    const isEdit = state.editing.dealId != null;
    const result = await api(isEdit ? `/deals/${state.editing.dealId}` : "/deals", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    dealResult.textContent = `${isEdit ? "Updated" : "Created"} deal ${result.name}.`;
    dealForm.reset();
    clearEditMode("deal");
    if (state.selectedOrganizationId) {
      dealOrganizationSelect.value = String(state.selectedOrganizationId);
    }
    await loadDashboard();
  } catch (error) {
    dealResult.textContent = error.message;
  }
}

async function submitPerson(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(personForm).entries());
  if (body.organization_id) body.organization_id = Number(body.organization_id); else delete body.organization_id;
  try {
    const isEdit = state.editing.personId != null;
    const result = await api(isEdit ? `/people/${state.editing.personId}` : "/people", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    personResult.textContent = `${isEdit ? "Updated" : "Created"} person ${result.first_name} ${result.last_name}.`;
    personForm.reset();
    clearEditMode("person");
    if (state.selectedOrganizationId) {
      personOrganizationSelect.value = String(state.selectedOrganizationId);
    }
    await loadDashboard();
  } catch (error) {
    personResult.textContent = error.message;
  }
}

async function submitTask(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(taskForm).entries());
  if (body.organization_id) body.organization_id = Number(body.organization_id); else delete body.organization_id;
  if (body.deal_id) body.deal_id = Number(body.deal_id); else delete body.deal_id;
  try {
    const isEdit = state.editing.taskId != null;
    const result = await api(isEdit ? `/tasks/${state.editing.taskId}` : "/tasks", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    taskResult.textContent = `${isEdit ? "Updated" : "Created"} task ${result.title}.`;
    taskForm.reset();
    clearEditMode("task");
    if (state.selectedOrganizationId) {
      taskOrganizationSelect.value = String(state.selectedOrganizationId);
    }
    await loadDashboard();
  } catch (error) {
    taskResult.textContent = error.message;
  }
}

function cancelEdit(kind, form, resultEl, resetSelect) {
  form.reset();
  resultEl.textContent = "";
  clearEditMode(kind);
  if (resetSelect && state.selectedOrganizationId) {
    resetSelect.value = String(state.selectedOrganizationId);
  }
}

loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  loginError.classList.add("hidden");
  loginError.textContent = "";
  const body = Object.fromEntries(new FormData(loginForm).entries());

  try {
    const result = await api("/login", {
      method: "POST",
      body: JSON.stringify(body),
      headers: { "Content-Type": "application/json" },
    });
    setSession(result.token, { email: result.email, tenant_slug: result.tenant_slug });
    showApp();
    await loadDashboard();
  } catch (error) {
    loginError.textContent = error.message;
    loginError.classList.remove("hidden");
  }
});

document.getElementById("logout-button").addEventListener("click", () => {
  clearSession();
  showLogin();
});

document.getElementById("refresh-button").addEventListener("click", () => loadDashboard());
document.getElementById("refresh-outbox").addEventListener("click", () => loadDashboard());
document.getElementById("refresh-deliveries").addEventListener("click", () => loadDashboard());

organizationForm.addEventListener("submit", submitOrganization);
dealForm.addEventListener("submit", submitDeal);
personForm.addEventListener("submit", submitPerson);
taskForm.addEventListener("submit", submitTask);

organizationCancel.addEventListener("click", () => cancelEdit("organization", organizationForm, organizationResult));
dealCancel.addEventListener("click", () => cancelEdit("deal", dealForm, dealResult, dealOrganizationSelect));
personCancel.addEventListener("click", () => cancelEdit("person", personForm, personResult, personOrganizationSelect));
taskCancel.addEventListener("click", () => cancelEdit("task", taskForm, taskResult, taskOrganizationSelect));

document.addEventListener("click", async (event) => {
  const editButton = event.target.closest("[data-edit-type]");
  if (editButton) {
    const id = Number(editButton.getAttribute("data-edit-id"));
    const type = editButton.getAttribute("data-edit-type");
    if (type === "organization") startEditOrganization(id);
    if (type === "deal") startEditDeal(id);
    if (type === "person") startEditPerson(id);
    if (type === "task") startEditTask(id);
    if (type === "task-from-deal") primeTaskFromDeal(id);
    return;
  }

  const deleteButton = event.target.closest("[data-delete-type]");
  if (deleteButton) {
    const id = Number(deleteButton.getAttribute("data-delete-id"));
    const type = deleteButton.getAttribute("data-delete-type");
    await deleteEntity(type, id);
    return;
  }

  const organizationCard = event.target.closest("[data-organization-id]");
  if (organizationCard) {
    state.selectedOrganizationId = Number(organizationCard.getAttribute("data-organization-id"));
    populateOrganizationSelect();
    renderOrganizations();
    return;
  }

  const outboxButton = event.target.closest("[data-outbox-id]");
  if (outboxButton) {
    outboxButton.animate(
      [{ transform: "scale(1)" }, { transform: "scale(1.08)" }, { transform: "scale(1)" }],
      { duration: 260, easing: "ease-out" },
    );
    await api(`/outbox-events/${outboxButton.getAttribute("data-outbox-id")}/replay`, { method: "POST" });
    await loadDashboard();
    return;
  }

  const deliveryButton = event.target.closest("[data-delivery-id]");
  if (deliveryButton) {
    deliveryButton.animate(
      [{ transform: "scale(1)" }, { transform: "scale(1.08)" }, { transform: "scale(1)" }],
      { duration: 260, easing: "ease-out" },
    );
    await api(`/webhook-deliveries/${deliveryButton.getAttribute("data-delivery-id")}/replay`, { method: "POST" });
    await loadDashboard();
  }
});

commandSearch.addEventListener("input", () => {
  const query = commandSearch.value.trim().toLowerCase();
  if (!query) {
    renderOrganizations();
    renderDeals();
    renderPeople();
    renderTasks();
    return;
  }

  const matches = (value) => String(value || "").toLowerCase().includes(query);
  renderEntityCards("organization-list", state.organizations.filter((item) => matches(item.name) || matches(item.domain)), (item) => `
    <strong>${item.name}</strong>
    <div class="muted small">${item.domain || "no domain"} • ${item.industry || "no industry"}</div>
    <div class="entity-meta"><span class="pill">id ${item.id}</span></div>
  `);
  renderEntityCards("deal-list", state.deals.filter((item) => matches(item.name) || matches(item.stage)), (item) => `
    <strong>${item.name}</strong>
    <div class="muted small">${item.stage} • ${item.status}</div>
  `);
  renderEntityCards("person-list", state.people.filter((item) => matches(item.first_name) || matches(item.last_name) || matches(item.email)), (item) => `
    <strong>${item.first_name} ${item.last_name}</strong>
    <div class="muted small">${item.email || "no email"}</div>
  `);
  renderEntityCards("task-list", state.tasks.filter((item) => matches(item.title) || matches(item.status) || matches(item.priority)), (item) => `
    <strong>${item.title}</strong>
    <div class="muted small">${item.status} • ${item.priority}</div>
  `);
});

function markActiveNav() {
  const sections = [...document.querySelectorAll(".content-section[id]")];
  const navItems = [...document.querySelectorAll(".nav-item")];
  const current = sections.find((section) => {
    const rect = section.getBoundingClientRect();
    return rect.top <= 180 && rect.bottom >= 180;
  });
  if (!current) return;
  navItems.forEach((item) => item.classList.toggle("active", item.getAttribute("href") === `#${current.id}`));
}

window.addEventListener("scroll", markActiveNav, { passive: true });

async function boot() {
  if (!state.token || !state.session) {
    showLogin();
    return;
  }

  try {
    showApp();
    await loadDashboard();
    markActiveNav();
  } catch (_error) {
    clearSession();
    showLogin();
  }
}

boot();
