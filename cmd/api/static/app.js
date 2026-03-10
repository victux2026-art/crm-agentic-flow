const state = {
  token: localStorage.getItem("crmflow_token") || "",
  session: JSON.parse(localStorage.getItem("crmflow_session") || "null"),
  workspaceMode: localStorage.getItem("crmflow_workspace_mode") || "auto",
  organizations: [],
  deals: [],
  people: [],
  tasks: [],
  notes: [],
  adminTenant: null,
  adminUsers: [],
  webhookEndpoints: [],
  webhookSubscriptions: [],
  outboxFailed: [],
  deliveryFailed: [],
  audit: [],
  selectedOrganizationId: null,
  editing: {
    organizationId: null,
    dealId: null,
    personId: null,
    taskId: null,
    noteId: null,
    adminUserId: null,
    endpointId: null,
    subscriptionId: null,
  },
  filters: {
    dealStage: localStorage.getItem("crmflow_filter_deal_stage") || "",
    taskPriority: localStorage.getItem("crmflow_filter_task_priority") || "",
  },
  savedViews: JSON.parse(localStorage.getItem("crmflow_saved_views") || "[]"),
  homeWidgets: JSON.parse(localStorage.getItem("crmflow_home_widgets") || "{\"summary\":true,\"platform\":true,\"control\":true,\"activity\":true,\"snapshot\":true,\"posture\":true}"),
};

const authPanel = document.getElementById("auth-panel");
const appPanel = document.getElementById("app-panel");
const loginForm = document.getElementById("login-form");
const loginError = document.getElementById("login-error");
const sessionBadge = document.getElementById("session-badge");
const roleBadge = document.getElementById("role-badge");
const workerLed = document.getElementById("worker-led");
const workerStatusText = document.getElementById("worker-status-text");
const commandSearch = document.getElementById("command-search");
const workspaceSwitcher = document.getElementById("workspace-switcher");
const viewAdmin = document.getElementById("view-admin");
const viewTeam = document.getElementById("view-team");

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
const noteForm = document.getElementById("note-form");
const noteResult = document.getElementById("note-result");
const noteSubmit = document.getElementById("note-submit");
const noteCancel = document.getElementById("note-cancel");
const noteOrganizationSelect = document.getElementById("note-organization-id");
const noteDealSelect = document.getElementById("note-deal-id");
const dealStageFilter = document.getElementById("deal-stage-filter");
const taskPriorityFilter = document.getElementById("task-priority-filter");
const savedViewSelect = document.getElementById("saved-view-select");
const saveViewButton = document.getElementById("save-view-button");
const deleteViewButton = document.getElementById("delete-view-button");
const savedViewResult = document.getElementById("saved-view-result");
const tenantForm = document.getElementById("tenant-form");
const tenantResult = document.getElementById("tenant-result");
const adminUserForm = document.getElementById("admin-user-form");
const adminUserResult = document.getElementById("admin-user-result");
const adminUserSubmit = document.getElementById("admin-user-submit");
const adminUserCancel = document.getElementById("admin-user-cancel");

const endpointForm = document.getElementById("endpoint-form");
const endpointResult = document.getElementById("endpoint-result");
const endpointSubmit = document.getElementById("endpoint-submit");
const endpointCancel = document.getElementById("endpoint-cancel");

const subscriptionForm = document.getElementById("subscription-form");
const subscriptionResult = document.getElementById("subscription-result");
const subscriptionSubmit = document.getElementById("subscription-submit");
const subscriptionCancel = document.getElementById("subscription-cancel");
const subscriptionEndpointSelect = document.getElementById("subscription-endpoint-id");
const homeLayoutForm = document.getElementById("home-layout-form");

const HOME_SECTION_IDS = ["command-center", "dashboard"];
const PAGE_ALIASES = {
  "": "home",
  home: "home",
  "command-center": "home",
  dashboard: "home",
};

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

function isAdminSession() {
  return ["admin", "owner"].includes(String(state.session?.role || "").toLowerCase());
}

function currentWorkspaceMode() {
  if (!isAdminSession()) {
    return "team";
  }
  return state.workspaceMode === "team" ? "team" : "admin";
}

function setWorkspaceMode(mode) {
  state.workspaceMode = mode;
  localStorage.setItem("crmflow_workspace_mode", mode);
  applyWorkspaceMode();
}

function applyWorkspaceMode() {
  const mode = currentWorkspaceMode();
  const root = document.body;
  root.dataset.workspace = mode;

  if (isAdminSession()) {
    workspaceSwitcher.classList.remove("hidden");
    viewAdmin.classList.toggle("active", mode === "admin");
    viewTeam.classList.toggle("active", mode === "team");
  } else {
    workspaceSwitcher.classList.add("hidden");
  }

  if (state.token) {
    renderWorkspaceFocus();
    renderBridgeRail();
  }
}

function persistFilters() {
  localStorage.setItem("crmflow_filter_deal_stage", state.filters.dealStage);
  localStorage.setItem("crmflow_filter_task_priority", state.filters.taskPriority);
}

function persistHomeWidgets() {
  localStorage.setItem("crmflow_home_widgets", JSON.stringify(state.homeWidgets));
}

function applyHomeLayout() {
  Object.entries(state.homeWidgets).forEach(([key, enabled]) => {
    const element = document.querySelector(`[data-home-widget="${key}"]`);
    if (element) {
      element.classList.toggle("home-widget-hidden", !enabled);
    }
  });
}

function syncHomeLayoutControls() {
  if (!homeLayoutForm) return;
  Object.entries(state.homeWidgets).forEach(([key, enabled]) => {
    const input = homeLayoutForm.elements.namedItem(key);
    if (input) input.checked = Boolean(enabled);
  });
}

function currentPageID() {
  const raw = String(window.location.hash || "").replace(/^#/, "");
  const candidate = PAGE_ALIASES[raw] || raw;
  if (candidate === "home") return "home";
  return document.getElementById(candidate) ? candidate : "home";
}

function applyCurrentPage() {
  const activePage = currentPageID();
  const sections = [...document.querySelectorAll(".content-section[id]")];
  sections.forEach((section) => {
    const shouldShow = activePage === "home"
      ? HOME_SECTION_IDS.includes(section.id)
      : section.id === activePage;
    section.classList.toggle("page-hidden", !shouldShow);
  });

  const navItems = [...document.querySelectorAll(".nav-item[href^=\"#\"]")];
  navItems.forEach((item) => {
    const href = String(item.getAttribute("href") || "").replace(/^#/, "");
    const target = PAGE_ALIASES[href] || href;
    item.classList.toggle("active", activePage === target);
  });
}

function renderSavedViews() {
  savedViewSelect.innerHTML = '<option value="">Saved views</option>';
  state.savedViews.forEach((view) => {
    const option = document.createElement("option");
    option.value = view.id;
    option.textContent = view.name;
    savedViewSelect.appendChild(option);
  });
}

function applyFiltersToControls() {
  dealStageFilter.value = state.filters.dealStage || "";
  taskPriorityFilter.value = state.filters.taskPriority || "";
}

function saveCurrentView() {
  const name = window.prompt("View name");
  if (!name) return;
  const view = {
    id: `view_${Date.now()}`,
    name,
    filters: { ...state.filters },
  };
  state.savedViews = [...state.savedViews, view];
  localStorage.setItem("crmflow_saved_views", JSON.stringify(state.savedViews));
  renderSavedViews();
  savedViewSelect.value = view.id;
  savedViewResult.textContent = `Saved view ${name}.`;
}

function applySavedView(id) {
  const view = state.savedViews.find((item) => item.id === id);
  if (!view) return;
  state.filters = { ...state.filters, ...view.filters };
  persistFilters();
  applyFiltersToControls();
  renderPipeline(state.deals);
  renderTasks();
  savedViewResult.textContent = `Loaded view ${view.name}.`;
}

function deleteCurrentView() {
  const id = savedViewSelect.value;
  if (!id) {
    savedViewResult.textContent = "Select a saved view first.";
    return;
  }
  const view = state.savedViews.find((item) => item.id === id);
  state.savedViews = state.savedViews.filter((item) => item.id !== id);
  localStorage.setItem("crmflow_saved_views", JSON.stringify(state.savedViews));
  renderSavedViews();
  savedViewResult.textContent = view ? `Deleted view ${view.name}.` : "Deleted view.";
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

function renderWorkspaceFocus() {
  const target = document.getElementById("workspace-focus");
  const dashboardTarget = document.getElementById("dashboard-admin-posture");
  const mode = currentWorkspaceMode();
  const controlEntries = mode === "admin"
    ? [
      { label: "Workspace", value: "Administrator" },
      { label: "Primary focus", value: "Tenant governance, event health, revenue oversight" },
      { label: "Controls", value: "Admin, outbox, webhooks, audit, sales workspace" },
    ]
    : [
      { label: "Workspace", value: "Team Member" },
      { label: "Primary focus", value: "Accounts, deals, people, tasks" },
      { label: "Controls", value: "Sales workspace only" },
    ];

  target.innerHTML = controlEntries.map((item) => `
    <div class="list-row">
      <span>${item.label}</span>
      <strong>${item.value}</strong>
    </div>
  `).join("");

  if (dashboardTarget) {
    const adminPostureEntries = mode === "admin"
      ? [
        { label: "Current mode", value: "Admin command view" },
        { label: "Role scope", value: state.session?.role || "admin" },
        { label: "Tenant plan", value: state.adminTenant?.plan || "starter" },
        { label: "Ops pressure", value: `${state.deliveryFailed.length + state.outboxFailed.length} active issues` },
      ]
      : [
        { label: "Current mode", value: "Team sales view" },
        { label: "Role scope", value: state.session?.role || "member" },
        { label: "Focus", value: "Execution and follow-up" },
        { label: "Ops pressure", value: "Hidden from team workspace" },
      ];

    dashboardTarget.innerHTML = adminPostureEntries.map((item) => `
      <div class="list-row">
        <span>${item.label}</span>
        <strong>${item.value}</strong>
      </div>
    `).join("");
  }
}

function renderBridgeRail() {
  const target = document.getElementById("bridge-glance");
  const mode = currentWorkspaceMode();
  const openDeals = state.deals.filter((item) => !["won", "lost", "closed"].includes(String(item.status || "").toLowerCase()));
  const pipelineValue = openDeals.reduce((sum, item) => sum + Number(item.value_amount || 0), 0);
  const failedDeliveries = state.deliveryFailed.length;
  const failedOutbox = state.outboxFailed.length;
  const overdue = overdueTasks();
  const targetAccount = chooseTargetAccount();

  const cards = mode === "admin"
    ? [
      {
        label: "Tenant",
        value: state.adminTenant?.name || state.session?.tenant_slug || "n/a",
        meta: `${state.adminTenant?.plan || "starter"} plan`,
      },
      {
        label: "Open Pipeline",
        value: `USD ${Math.round(pipelineValue).toLocaleString()}`,
        meta: `${openDeals.length} live deals`,
      },
      {
        label: "Delivery Pressure",
        value: String(failedDeliveries),
        meta: failedDeliveries ? "failed deliveries" : "transport healthy",
        tone: failedDeliveries >= 3 ? "critical" : failedDeliveries ? "warn" : "ok",
      },
      {
        label: "Event Backlog",
        value: String(failedOutbox),
        meta: failedOutbox ? "failed outbox events" : "outbox stable",
        tone: failedOutbox >= 3 ? "critical" : failedOutbox ? "warn" : "ok",
      },
    ]
    : [
      {
        label: "My Open Work",
        value: String(openTasksForWorkspace().length),
        meta: "tasks in queue",
      },
      {
        label: "Overdue",
        value: String(overdue.length),
        meta: overdue.length ? "needs recovery" : "under control",
        tone: overdue.length ? "warn" : "ok",
      },
      {
        label: "Active Deals",
        value: String(activeDealsForWorkspace().length),
        meta: "revenue in motion",
      },
      {
        label: "Target Account",
        value: targetAccount?.name || "None",
        meta: "next best place to work",
      },
    ];

  target.innerHTML = cards.map((item) => `
    <article class="glance-card${item.tone ? ` tone-${item.tone}` : ""}">
      <span class="glance-label">${item.label}</span>
      <strong class="glance-value">${item.value}</strong>
      <span class="glance-meta">${item.meta}</span>
    </article>
  `).join("");

  const commandSummary = document.getElementById("command-summary");
  if (commandSummary) {
    const summaryEntries = mode === "admin"
      ? [
        { label: "Tenant slug", value: state.session?.tenant_slug || "n/a" },
        { label: "Users", value: String(state.adminUsers.length) },
        { label: "Webhook endpoints", value: String(state.webhookEndpoints.length) },
        { label: "Open deals", value: String(openDeals.length) },
      ]
      : [
        { label: "Open tasks", value: String(openTasksForWorkspace().length) },
        { label: "Accounts in motion", value: String(activeOrganizationsForWorkspace().length) },
        { label: "Target account", value: targetAccount?.name || "None" },
        { label: "Overdue", value: String(overdue.length) },
      ];

    commandSummary.innerHTML = summaryEntries.map((item) => `
      <div class="list-row">
        <span>${item.label}</span>
        <strong>${item.value}</strong>
      </div>
    `).join("");
  }
}

function renderAdminUsers() {
  renderEntityCards("admin-users-list", state.adminUsers, (item) => `
    <strong>${item.full_name}</strong>
    <div class="muted small">${item.email}</div>
    <div class="entity-meta">
      <span class="pill">${item.role}</span>
      <span class="pill">${item.user_status}</span>
      <span class="pill">${item.membership_status}</span>
    </div>
    <div class="entity-card-actions">
      <button type="button" class="ghost small-button" data-edit-type="adminUser" data-edit-id="${item.id}">Edit</button>
    </div>
  `);
}

function renderSalesSnapshot() {
  const target = document.getElementById("sales-snapshot");
  const activeDeals = state.deals.filter((item) => !["won", "lost", "closed"].includes(String(item.status || "").toLowerCase()));
  const lateStageDeals = activeDeals.filter((item) => ["proposal", "negotiation"].includes(String(item.stage || "").toLowerCase()));
  const openTasks = state.tasks.filter((item) => !["done", "completed", "closed"].includes(String(item.status || "").toLowerCase()));
  const pipelineValue = activeDeals.reduce((sum, item) => sum + Number(item.value_amount || 0), 0);
  const lateStageValue = lateStageDeals.reduce((sum, item) => sum + Number(item.value_amount || 0), 0);

  target.innerHTML = `
    <div class="list-row"><span>Pipeline value</span><strong>USD ${pipelineValue.toLocaleString()}</strong></div>
    <div class="list-row"><span>Late-stage value</span><strong>USD ${lateStageValue.toLocaleString()}</strong></div>
    <div class="list-row"><span>Open tasks</span><strong>${openTasks.length}</strong></div>
    <div class="list-row"><span>Notes captured</span><strong>${state.notes.length}</strong></div>
  `;
}

function renderCommercialReporting() {
  const stageTarget = document.getElementById("report-stage-funnel");
  const conversionTarget = document.getElementById("report-conversion");
  const forecastTarget = document.getElementById("report-forecast");
  const taskLoadTarget = document.getElementById("report-task-load");
  const forecastNotesTarget = document.getElementById("forecast-notes");

  const stageOrder = ["lead", "proposal", "negotiation", "won"];
  const stageWeights = {
    lead: 0.1,
    proposal: 0.4,
    negotiation: 0.7,
    won: 1,
  };

  const stageRows = stageOrder.map((stage) => {
    const deals = state.deals.filter((item) => (item.stage || "lead") === stage);
    const value = deals.reduce((sum, item) => sum + Number(item.value_amount || 0), 0);
    return { stage, count: deals.length, value };
  });

  stageTarget.innerHTML = stageRows.map((item) => `
    <div class="list-row">
      <span>${item.stage}</span>
      <strong>${item.count} deals / USD ${item.value.toLocaleString()}</strong>
    </div>
  `).join("");

  const wonDeals = state.deals.filter((item) => String(item.status || "").toLowerCase() === "won");
  const lostDeals = state.deals.filter((item) => String(item.status || "").toLowerCase() === "lost");
  const openDeals = state.deals.filter((item) => !["won", "lost", "closed"].includes(String(item.status || "").toLowerCase()));
  const closedCount = wonDeals.length + lostDeals.length;
  const winRate = closedCount ? Math.round((wonDeals.length / closedCount) * 100) : 0;

  conversionTarget.innerHTML = `
    <div class="list-row"><span>Won deals</span><strong>${wonDeals.length}</strong></div>
    <div class="list-row"><span>Lost deals</span><strong>${lostDeals.length}</strong></div>
    <div class="list-row"><span>Open deals</span><strong>${openDeals.length}</strong></div>
    <div class="list-row"><span>Win rate</span><strong>${winRate}%</strong></div>
  `;

  const weightedForecast = openDeals.reduce((sum, item) => {
    const weight = stageWeights[String(item.stage || "lead").toLowerCase()] ?? 0.1;
    return sum + Number(item.value_amount || 0) * weight;
  }, 0);
  const bestCase = openDeals.reduce((sum, item) => sum + Number(item.value_amount || 0), 0);
  const lateStageOpen = openDeals.filter((item) => ["proposal", "negotiation"].includes(String(item.stage || "").toLowerCase()));

  forecastTarget.innerHTML = `
    <div class="list-row"><span>Weighted forecast</span><strong>USD ${Math.round(weightedForecast).toLocaleString()}</strong></div>
    <div class="list-row"><span>Best case pipeline</span><strong>USD ${bestCase.toLocaleString()}</strong></div>
    <div class="list-row"><span>Late-stage open</span><strong>${lateStageOpen.length}</strong></div>
  `;

  if (forecastNotesTarget) {
    const coverage = bestCase > 0 ? Math.round((weightedForecast / bestCase) * 100) : 0;
    const dominantStage = stageRows.slice().sort((a, b) => b.value - a.value)[0];
    forecastNotesTarget.innerHTML = `
      <div class="list-row"><span>Coverage ratio</span><strong>${coverage}% weighted vs best case</strong></div>
      <div class="list-row"><span>Dominant stage</span><strong>${dominantStage?.stage || "lead"}</strong></div>
      <div class="list-row"><span>Late-stage value</span><strong>USD ${lateStageOpen.reduce((sum, item) => sum + Number(item.value_amount || 0), 0).toLocaleString()}</strong></div>
      <div class="list-row"><span>Closing posture</span><strong>${lateStageOpen.length ? "watch negotiation quality" : "build late-stage pipeline"}</strong></div>
    `;
  }

  const openTasks = state.tasks.filter((item) => !["done", "completed", "closed"].includes(String(item.status || "").toLowerCase()));
  const highTasks = openTasks.filter((item) => String(item.priority || "").toLowerCase() === "high");
  const overdue = overdueTasks();
  const today = dueTodayTasks();

  taskLoadTarget.innerHTML = `
    <div class="list-row"><span>Open tasks</span><strong>${openTasks.length}</strong></div>
    <div class="list-row"><span>High priority</span><strong>${highTasks.length}</strong></div>
    <div class="list-row"><span>Due today</span><strong>${today.length}</strong></div>
    <div class="list-row"><span>Overdue</span><strong>${overdue.length}</strong></div>
  `;
}

function renderSystemActivityStream() {
  const target = document.getElementById("system-activity-stream");
  if (!target) return;

  const activity = [
    ...state.audit.slice(0, 8).map((item) => ({
      ts: item.created_at,
      title: `${item.entity_type} ${item.action}`,
      detail: `entity ${item.entity_id}`,
      kind: "audit",
      tone: "neutral",
    })),
    ...state.outboxFailed.slice(0, 5).map((item) => ({
      ts: item.created_at,
      title: `outbox ${item.event_type}`,
      detail: item.last_error || "failed event requires attention",
      kind: "event",
      tone: "warn",
    })),
    ...state.deliveryFailed.slice(0, 5).map((item) => ({
      ts: item.created_at,
      title: `delivery #${item.id} ${item.event_type || "webhook"}`,
      detail: item.last_error || "delivery failure detected",
      kind: "delivery",
      tone: "warn",
    })),
  ]
    .filter((item) => item.ts)
    .sort((a, b) => new Date(b.ts) - new Date(a.ts))
    .slice(0, 10);

  if (!activity.length) {
    target.innerHTML = `<div class="event-card"><span class="muted">No system activity yet.</span></div>`;
    return;
  }

  target.innerHTML = activity.map((item) => `
    <article class="event-card">
      <header>
        <div>
          <strong>${item.title}</strong>
          <div class="muted small">${item.kind} • ${item.detail}</div>
        </div>
        <span class="status ${item.tone === "warn" ? "status-failed" : "status-processed"}">${item.kind}</span>
      </header>
      <div class="muted small">${new Date(item.ts).toLocaleString()}</div>
    </article>
  `).join("");
}

function renderPlatformConsole() {
  const outboxSummary = document.getElementById("outbox-console-summary");
  const webhookSummary = document.getElementById("webhook-console-summary");
  const auditStream = document.getElementById("audit-monitor-stream");

  if (outboxSummary) {
    outboxSummary.innerHTML = `
      <div class="list-row"><span>Failed events</span><strong>${state.outboxFailed.length}</strong></div>
      <div class="list-row"><span>Replay available</span><strong>${state.outboxFailed.length ? "yes" : "idle"}</strong></div>
      <div class="list-row"><span>Operator stance</span><strong>${state.outboxFailed.length >= 3 ? "critical" : state.outboxFailed.length ? "watch" : "healthy"}</strong></div>
    `;
  }

  if (webhookSummary) {
    webhookSummary.innerHTML = `
      <div class="list-row"><span>Failed deliveries</span><strong>${state.deliveryFailed.length}</strong></div>
      <div class="list-row"><span>Configured endpoints</span><strong>${state.webhookEndpoints.length}</strong></div>
      <div class="list-row"><span>Transport stance</span><strong>${state.deliveryFailed.length >= 3 ? "critical" : state.deliveryFailed.length ? "degraded" : "healthy"}</strong></div>
    `;
  }

  if (auditStream) {
    if (!state.audit.length) {
      auditStream.innerHTML = `<div class="event-card"><span class="muted">No audit entries.</span></div>`;
      return;
    }
    auditStream.innerHTML = state.audit.slice(0, 12).map((item) => `
      <article class="event-card">
        <header>
          <div>
            <strong>${item.entity_type}</strong>
            <div class="muted small">${item.action} • entity ${item.entity_id}</div>
          </div>
          <span class="muted small">${new Date(item.created_at).toLocaleString()}</span>
        </header>
      </article>
    `).join("");
  }
}

function renderLeadModule() {
  const inboxTarget = document.getElementById("lead-inbox-summary");
  const blueprintTarget = document.getElementById("lead-blueprint");
  if (!inboxTarget || !blueprintTarget) return;

  const unlinkedPeople = state.people.filter((item) => !item.organization_id);
  const orgsWithoutDeals = state.organizations.filter((org) => !state.deals.some((deal) => deal.organization_id === org.id));
  const recentNotes = state.notes.slice(0, 5);

  inboxTarget.innerHTML = `
    <div class="list-row"><span>Potential lead contacts</span><strong>${unlinkedPeople.length}</strong></div>
    <div class="list-row"><span>Accounts without opportunity</span><strong>${orgsWithoutDeals.length}</strong></div>
    <div class="list-row"><span>Recent notes to classify</span><strong>${recentNotes.length}</strong></div>
    <div class="detail-card"><span class="muted">Until a dedicated lead object exists, this page highlights where inbound or early-stage qualification pressure is building inside current CRM data.</span></div>
  `;

  blueprintTarget.innerHTML = `
    <div class="list-row"><span>Status</span><strong>navigation live, model pending</strong></div>
    <div class="list-row"><span>Expected output</span><strong>promote qualified leads into accounts and opportunities</strong></div>
    <div class="list-row"><span>Priority signals</span><strong>source, fit, urgency, owner</strong></div>
    <div class="list-row"><span>Best next implementation</span><strong>create leads + conversion flow</strong></div>
  `;
}

function renderReportsOverview() {
  const summaryTarget = document.getElementById("reports-overview-summary");
  const usageTarget = document.getElementById("reports-overview-usage");
  if (!summaryTarget || !usageTarget) return;

  const openDeals = state.deals.filter((item) => !["won", "lost", "closed"].includes(String(item.status || "").toLowerCase()));
  const wonDeals = state.deals.filter((item) => String(item.status || "").toLowerCase() === "won");
  const openTasks = state.tasks.filter((item) => !["done", "completed", "closed"].includes(String(item.status || "").toLowerCase()));

  summaryTarget.innerHTML = `
    <div class="list-row"><span>Dashboards</span><strong>pipeline, conversion, workload</strong></div>
    <div class="list-row"><span>Forecasts</span><strong>weighted and best-case posture</strong></div>
    <div class="list-row"><span>Open pipeline</span><strong>${openDeals.length} deals</strong></div>
    <div class="list-row"><span>Closed won</span><strong>${wonDeals.length}</strong></div>
  `;

  usageTarget.innerHTML = `
    <div class="list-row"><span>Leadership cadence</span><strong>use dashboards for weekly operating review</strong></div>
    <div class="list-row"><span>Revenue posture</span><strong>use forecasts for closing confidence</strong></div>
    <div class="list-row"><span>Execution pressure</span><strong>${openTasks.length} open tasks inform delivery risk</strong></div>
    <div class="list-row"><span>Next step</span><strong>saved reports and exportable views</strong></div>
  `;
}

function renderImportsModule() {
  const summaryTarget = document.getElementById("imports-summary");
  const flowTarget = document.getElementById("imports-flow");
  if (!summaryTarget || !flowTarget) return;

  const totalEntities = state.organizations.length + state.people.length + state.deals.length + state.tasks.length;
  const accountsWithoutDomain = state.organizations.filter((item) => !item.domain).length;
  const contactsWithoutEmail = state.people.filter((item) => !item.email).length;

  summaryTarget.innerHTML = `
    <div class="list-row"><span>Current CRM records</span><strong>${totalEntities}</strong></div>
    <div class="list-row"><span>Accounts missing domain</span><strong>${accountsWithoutDomain}</strong></div>
    <div class="list-row"><span>Contacts missing email</span><strong>${contactsWithoutEmail}</strong></div>
    <div class="detail-card"><span class="muted">This import page is now useful as a readiness checkpoint: it shows where current data quality will matter once CSV and provider import jobs are connected.</span></div>
  `;

  flowTarget.innerHTML = `
    <div class="list-row"><span>1. Upload</span><strong>CSV or provider extract</strong></div>
    <div class="list-row"><span>2. Preview</span><strong>mapping and validation</strong></div>
    <div class="list-row"><span>3. Execute</span><strong>tracked import jobs</strong></div>
    <div class="list-row"><span>4. Review</span><strong>errors, duplicates, ownership fixes</strong></div>
  `;
}

function userOwnsItem(item) {
  const userID = Number(state.session?.user_id || 0);
  if (!userID) return false;
  return Number(item.owner_user_id || item.created_by_user_id || 0) === userID;
}

function openTasksForWorkspace() {
  const open = state.tasks.filter((item) => !["done", "completed", "closed"].includes(String(item.status || "").toLowerCase()));
  const mine = open.filter((item) => userOwnsItem(item));
  const source = mine.length ? mine : open;
  return source.sort((a, b) => {
    const priorityWeight = { high: 0, medium: 1, low: 2 };
    const aPriority = priorityWeight[String(a.priority || "").toLowerCase()] ?? 3;
    const bPriority = priorityWeight[String(b.priority || "").toLowerCase()] ?? 3;
    return aPriority - bPriority;
  });
}

function activeDealsForWorkspace() {
  const active = state.deals.filter((item) => !["won", "lost", "closed"].includes(String(item.status || "").toLowerCase()));
  const mine = active.filter((item) => userOwnsItem(item));
  const source = mine.length ? mine : active;
  return source.sort((a, b) => Number(b.value_amount || 0) - Number(a.value_amount || 0));
}

function activeOrganizationsForWorkspace() {
  const ids = new Set();
  openTasksForWorkspace().slice(0, 6).forEach((item) => {
    if (item.organization_id) ids.add(Number(item.organization_id));
  });
  activeDealsForWorkspace().slice(0, 6).forEach((item) => {
    if (item.organization_id) ids.add(Number(item.organization_id));
  });
  return state.organizations.filter((item) => ids.has(Number(item.id))).slice(0, 6);
}

function nextDealStage(stage) {
  const flow = ["lead", "proposal", "negotiation", "won"];
  const index = flow.indexOf(String(stage || "").toLowerCase());
  if (index === -1 || index === flow.length - 1) return null;
  return flow[index + 1];
}

function isTaskOpen(item) {
  return !["done", "completed", "closed"].includes(String(item.status || "").toLowerCase());
}

function dueDateParts(item) {
  if (!item?.due_at) return null;
  const due = new Date(item.due_at);
  if (Number.isNaN(due.getTime())) return null;
  const dueKey = due.toISOString().slice(0, 10);
  const today = new Date();
  const todayKey = new Date(Date.UTC(today.getFullYear(), today.getMonth(), today.getDate())).toISOString().slice(0, 10);
  return { due, dueKey, todayKey };
}

function dueTodayTasks() {
  return openTasksForWorkspace().filter((item) => {
    const parts = dueDateParts(item);
    return parts && parts.dueKey === parts.todayKey;
  });
}

function overdueTasks() {
  return openTasksForWorkspace().filter((item) => {
    const parts = dueDateParts(item);
    return parts && parts.dueKey < parts.todayKey;
  });
}

function teamPipelineCounts() {
  const counts = new Map([
    ["lead", 0],
    ["proposal", 0],
    ["negotiation", 0],
    ["won", 0],
  ]);
  activeDealsForWorkspace().forEach((item) => {
    const stage = counts.has(item.stage) ? item.stage : "lead";
    counts.set(stage, counts.get(stage) + 1);
  });
  return [...counts.entries()].map(([stage, count]) => ({ stage, count }));
}

function chooseTargetAccount() {
  const organizationScores = new Map();

  overdueTasks().forEach((task) => {
    if (!task.organization_id) return;
    const id = Number(task.organization_id);
    organizationScores.set(id, (organizationScores.get(id) || 0) + 4);
  });

  dueTodayTasks().forEach((task) => {
    if (!task.organization_id) return;
    const id = Number(task.organization_id);
    organizationScores.set(id, (organizationScores.get(id) || 0) + 2);
  });

  activeDealsForWorkspace().forEach((deal) => {
    if (!deal.organization_id) return;
    const id = Number(deal.organization_id);
    const boost = ["negotiation", "proposal"].includes(String(deal.stage || "").toLowerCase()) ? 3 : 1;
    organizationScores.set(id, (organizationScores.get(id) || 0) + boost);
  });

  const [bestID] = [...organizationScores.entries()].sort((a, b) => b[1] - a[1])[0] || [];
  return state.organizations.find((item) => Number(item.id) === Number(bestID)) || activeOrganizationsForWorkspace()[0] || null;
}

function renderTeamHub() {
  const focusSummary = document.getElementById("team-focus-summary");
  const teamTaskList = document.getElementById("team-task-list");
  const teamDealList = document.getElementById("team-deal-list");
  const teamOrganizationList = document.getElementById("team-organization-list");

  const openTasks = openTasksForWorkspace();
  const activeDeals = activeDealsForWorkspace();
  const activeOrganizations = activeOrganizationsForWorkspace();
  const proposalDeals = activeDeals.filter((item) => ["proposal", "negotiation"].includes(String(item.stage || "").toLowerCase()));
  const todayTasks = dueTodayTasks();
  const overdue = overdueTasks();
  const targetAccount = chooseTargetAccount();
  const teamTodayList = document.getElementById("team-today-list");
  const teamOverdueList = document.getElementById("team-overdue-list");
  const teamPipeline = document.getElementById("team-pipeline");
  const teamTargetAccount = document.getElementById("team-target-account");
  const teamAccountsList = document.getElementById("team-accounts-list");
  const teamActivityList = document.getElementById("team-activity-list");

  focusSummary.innerHTML = `
    <div class="list-row"><span>Open tasks</span><strong>${openTasks.length}</strong></div>
    <div class="list-row"><span>Due today</span><strong>${todayTasks.length}</strong></div>
    <div class="list-row"><span>Overdue</span><strong>${overdue.length}</strong></div>
    <div class="list-row"><span>Active deals</span><strong>${activeDeals.length}</strong></div>
    <div class="list-row"><span>Late-stage deals</span><strong>${proposalDeals.length}</strong></div>
    <div class="list-row"><span>Active accounts</span><strong>${activeOrganizations.length}</strong></div>
  `;

  teamTaskList.innerHTML = openTasks.slice(0, 6).map((item) => {
    const organization = state.organizations.find((entry) => entry.id === item.organization_id);
    return `
      <div class="compact-row">
        <div>
          <strong>${item.title}</strong>
          <div class="muted small">${organization ? organization.name : "No organization"}</div>
        </div>
        <span class="pill">${item.priority || item.status}</span>
      </div>
    `;
  }).join("") || `<span class="muted small">No open tasks.</span>`;

  teamDealList.innerHTML = activeDeals.slice(0, 6).map((item) => {
    const organization = state.organizations.find((entry) => entry.id === item.organization_id);
    return `
      <div class="compact-row">
        <div>
          <strong>${item.name}</strong>
          <div class="muted small">${organization ? organization.name : "No organization"} • ${item.stage}</div>
        </div>
        <span class="pill">${item.value_currency} ${Number(item.value_amount || 0).toLocaleString()}</span>
      </div>
    `;
  }).join("") || `<span class="muted small">No active deals.</span>`;

  teamOrganizationList.innerHTML = activeOrganizations.map((item) => `
    <div class="compact-row">
      <div>
        <strong>${item.name}</strong>
        <div class="muted small">${item.domain || "no domain"} • ${item.industry || "no industry"}</div>
      </div>
      <span class="pill">account</span>
    </div>
  `).join("") || `<span class="muted small">No active accounts yet.</span>`;

  const renderTaskMini = (item) => {
    const organization = state.organizations.find((entry) => entry.id === item.organization_id);
    const due = item.due_at ? new Date(item.due_at).toLocaleDateString() : "no due date";
    return `
      <div class="compact-row">
        <div>
          <strong>${item.title}</strong>
          <div class="muted small">${organization ? organization.name : "No organization"} • ${due}</div>
        </div>
        <span class="pill">${item.priority}</span>
      </div>
    `;
  };

  teamTodayList.innerHTML = todayTasks.slice(0, 4).map(renderTaskMini).join("") || `<span class="muted small">No tasks due today.</span>`;
  teamOverdueList.innerHTML = overdue.slice(0, 4).map(renderTaskMini).join("") || `<span class="muted small">No overdue tasks.</span>`;

  teamPipeline.innerHTML = teamPipelineCounts().map((item) => `
    <div class="list-row">
      <span>${item.stage}</span>
      <strong>${item.count}</strong>
    </div>
  `).join("");

  teamAccountsList.innerHTML = activeOrganizations.map((item) => {
    const orgDeals = activeDeals.filter((deal) => deal.organization_id === item.id);
    const orgTasks = openTasks.filter((task) => task.organization_id === item.id);
    return `
      <button type="button" class="compact-row account-focus-row" data-focus-organization="${item.id}">
        <div>
          <strong>${item.name}</strong>
          <div class="muted small">${item.domain || "no domain"} • ${item.industry || "no industry"}</div>
        </div>
        <span class="pill">${orgDeals.length} deals / ${orgTasks.length} tasks</span>
      </button>
    `;
  }).join("") || `<span class="muted small">No accounts in focus.</span>`;

  const recentActivity = [
    ...activeDeals.slice(0, 4).map((item) => ({
      ts: item.updated_at || item.created_at,
      title: item.name,
      detail: `${item.stage} • ${item.status}`,
      kind: "deal",
    })),
    ...openTasks.slice(0, 4).map((item) => ({
      ts: item.updated_at || item.created_at,
      title: item.title,
      detail: `${item.priority} • ${item.status}`,
      kind: "task",
    })),
  ].sort((a, b) => new Date(b.ts) - new Date(a.ts)).slice(0, 6);

  teamActivityList.innerHTML = recentActivity.map((item) => `
    <div class="compact-row">
      <div>
        <strong>${item.title}</strong>
        <div class="muted small">${item.detail}</div>
      </div>
      <span class="pill">${item.kind}</span>
    </div>
  `).join("") || `<span class="muted small">No recent activity.</span>`;

  if (!targetAccount) {
    teamTargetAccount.innerHTML = `<span class="muted">No target account yet.</span>`;
    return;
  }

  const targetDeals = activeDeals.filter((item) => item.organization_id === targetAccount.id).slice(0, 3);
  const targetTasks = openTasks.filter((item) => item.organization_id === targetAccount.id).slice(0, 3);

  teamTargetAccount.innerHTML = `
    <div class="detail-grid">
      <div>
        <p class="eyebrow">Target Account</p>
        <h3>${targetAccount.name}</h3>
        <p class="muted">${targetAccount.domain || "No domain"} • ${targetAccount.industry || "No industry"}</p>
        <div class="entity-meta">
          <span class="pill">deals ${targetDeals.length}</span>
          <span class="pill">tasks ${targetTasks.length}</span>
        </div>
      </div>
      <div class="detail-section">
        <h4>Active Deals</h4>
        <div class="compact-list">
          ${targetDeals.length ? targetDeals.map((item) => `
            <div class="compact-row">
              <span>${item.name}</span>
              <span class="muted small">${item.stage} • ${item.value_currency} ${Number(item.value_amount || 0).toLocaleString()}</span>
            </div>
          `).join("") : `<span class="muted small">No active deals.</span>`}
        </div>
      </div>
      <div class="detail-section">
        <h4>Open Tasks</h4>
        <div class="compact-list">
          ${targetTasks.length ? targetTasks.map((item) => `
            <div class="compact-row">
              <span>${item.title}</span>
              <span class="muted small">${item.priority} • ${item.status}</span>
            </div>
          `).join("") : `<span class="muted small">No open tasks.</span>`}
        </div>
      </div>
    </div>
  `;
}

function setWorkerStatus(isLive) {
  workerLed.style.background = isLive ? "var(--ok)" : "var(--bad)";
  workerStatusText.textContent = isLive ? "Worker Live" : "Worker Idle";
}

function renderPipeline(deals) {
  const board = document.getElementById("pipeline-board");
  const filteredDeals = deals.filter((deal) => !state.filters.dealStage || deal.stage === state.filters.dealStage);
  const columns = [
    { id: "lead", label: "Lead" },
    { id: "proposal", label: "Proposal" },
    { id: "negotiation", label: "Negotiation" },
    { id: "won", label: "Won" },
  ];

  board.innerHTML = "";

  columns.forEach((column) => {
    const columnDeals = filteredDeals.filter((deal) => deal.stage === column.id || (column.id === "lead" && !["proposal", "negotiation", "won"].includes(deal.stage)));
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
          const nextStage = nextDealStage(deal.stage);
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
                ${nextStage ? `<button type="button" class="ghost compact-action" data-stage-deal-id="${deal.id}" data-stage-next="${nextStage}">Move to ${nextStage}</button>` : `<button type="button" class="ghost compact-action" data-edit-type="deal" data-edit-id="${deal.id}">Review</button>`}
                <button type="button" class="ghost compact-action" data-edit-type="task-from-deal" data-edit-id="${deal.id}">Log Activity</button>
              </div>
            </article>`;
        }).join("") || `<div class="mini-card"><span class="muted small">No deals in this stage.</span></div>`}
      </div>
    `;
    board.appendChild(node);
  });
}

async function moveDealToNextStage(id, nextStage) {
  const deal = state.deals.find((entry) => entry.id === id);
  if (!deal) return;

  const body = {
    organization_id: deal.organization_id,
    primary_person_id: deal.primary_person_id,
    name: deal.name,
    stage: nextStage,
    status: nextStage === "won" ? "won" : deal.status,
    value_amount: Number(deal.value_amount || 0),
    value_currency: deal.value_currency || "USD",
    close_date_expected: deal.close_date_expected,
    close_date_actual: deal.close_date_actual,
    owner_user_id: deal.owner_user_id,
    health_score: deal.health_score,
    source: deal.source,
    metadata: deal.metadata || {},
  };

  await api(`/deals/${id}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
  await loadDashboard();
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
  const filteredTasks = state.tasks.filter((item) => !state.filters.taskPriority || String(item.priority || "").toLowerCase() === state.filters.taskPriority);
  renderEntityCards("task-list", filteredTasks, (item) => {
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

function renderNotes() {
  const target = document.getElementById("notes-list");
  target.innerHTML = "";
  if (!state.notes.length) {
    target.innerHTML = `<div class="event-card"><span class="muted">No notes yet.</span></div>`;
    return;
  }

  state.notes.slice(0, 10).forEach((item) => {
    const organization = state.organizations.find((entry) => entry.id === item.organization_id);
    const deal = state.deals.find((entry) => entry.id === item.deal_id);
    target.innerHTML += `
      <article class="event-card">
        <header>
          <div>
            <strong>${organization ? organization.name : "General note"}</strong>
            <div class="muted small">${deal ? deal.name + " • " : ""}${new Date(item.created_at).toLocaleString()}</div>
          </div>
          <span class="pill">${item.source}</span>
        </header>
        <p class="note-body">${item.body}</p>
      </article>
    `;
  });
}

function renderWebhookEndpoints() {
  renderEntityCards("admin-endpoint-list", state.webhookEndpoints, (item) => `
    <strong>${item.name}</strong>
    <div class="muted small endpoint-url">${item.target_url}</div>
    <div class="entity-meta">
      <span class="pill">${item.status}</span>
      <span class="pill">id ${item.id}</span>
    </div>
    <div class="entity-card-actions">
      <button type="button" class="ghost small-button" data-edit-type="endpoint" data-edit-id="${item.id}">Edit</button>
      <button type="button" class="ghost small-button" data-delete-type="endpoint" data-delete-id="${item.id}">Delete</button>
    </div>
  `);
}

function renderWebhookSubscriptions() {
  renderEntityCards("admin-subscription-list", state.webhookSubscriptions, (item) => {
    const endpoint = state.webhookEndpoints.find((entry) => entry.id === item.webhook_endpoint_id);
    return `
      <strong>${item.event_type}</strong>
      <div class="muted small">${endpoint ? endpoint.name : "Unknown endpoint"}</div>
      <div class="entity-meta">
        <span class="pill">${item.is_active ? "active" : "inactive"}</span>
        <span class="pill">id ${item.id}</span>
      </div>
      <div class="entity-card-actions">
        <button type="button" class="ghost small-button" data-edit-type="subscription" data-edit-id="${item.id}">Edit</button>
        <button type="button" class="ghost small-button" data-delete-type="subscription" data-delete-id="${item.id}">Delete</button>
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

function renderAdminCenter() {
  const tenantSummary = document.getElementById("tenant-summary");
  const workspacePolicy = document.getElementById("workspace-policy");

  tenantSummary.innerHTML = `
    <div class="list-row"><span>Tenant</span><strong>${state.session?.tenant_slug || "n/a"}</strong></div>
    <div class="list-row"><span>User</span><strong>${state.session?.email || "n/a"}</strong></div>
    <div class="list-row"><span>Role</span><strong>${state.session?.role || "n/a"}</strong></div>
    <div class="list-row"><span>Organizations</span><strong>${state.organizations.length}</strong></div>
    <div class="list-row"><span>Deals</span><strong>${state.deals.length}</strong></div>
    <div class="list-row"><span>Endpoints</span><strong>${state.webhookEndpoints.length}</strong></div>
  `;

  workspacePolicy.innerHTML = `
    <div class="list-row"><span>Admin view</span><strong>sales + ops + settings</strong></div>
    <div class="list-row"><span>Team member view</span><strong>sales-first workspace</strong></div>
    <div class="list-row"><span>Current mode</span><strong>${currentWorkspaceMode()}</strong></div>
  `;

  if (state.adminTenant) {
    tenantForm.elements.namedItem("name").value = state.adminTenant.name || "";
    tenantForm.elements.namedItem("slug").value = state.adminTenant.slug || "";
    tenantForm.elements.namedItem("plan").value = state.adminTenant.plan || "starter";
    tenantForm.elements.namedItem("status").value = state.adminTenant.status || "active";
  }

  renderAdminUsers();
  renderWebhookEndpoints();
  renderWebhookSubscriptions();
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
  const selects = [dealOrganizationSelect, personOrganizationSelect, taskOrganizationSelect, noteOrganizationSelect];
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
  [taskDealSelect, noteDealSelect].forEach((select) => {
    select.innerHTML = '<option value="">Optional deal</option>';
    state.deals.forEach((item) => {
      const option = document.createElement("option");
      option.value = item.id;
      option.textContent = item.name;
      select.appendChild(option);
    });
  });
}

function populateSubscriptionEndpointSelect() {
  subscriptionEndpointSelect.innerHTML = '<option value="">Select endpoint</option>';
  state.webhookEndpoints.forEach((item) => {
    const option = document.createElement("option");
    option.value = item.id;
    option.textContent = item.name;
    subscriptionEndpointSelect.appendChild(option);
  });
}

function setEditMode(kind, id) {
  state.editing[`${kind}Id`] = id;
  const submitMap = {
    organization: organizationSubmit,
    deal: dealSubmit,
    person: personSubmit,
    task: taskSubmit,
    note: noteSubmit,
    endpoint: endpointSubmit,
    subscription: subscriptionSubmit,
    adminUser: adminUserSubmit,
  };
  const cancelMap = {
    organization: organizationCancel,
    deal: dealCancel,
    person: personCancel,
    task: taskCancel,
    note: noteCancel,
    endpoint: endpointCancel,
    subscription: subscriptionCancel,
    adminUser: adminUserCancel,
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
    note: "Create Note",
    endpoint: "Create Endpoint",
    subscription: "Create Subscription",
    adminUser: "Create User",
  };
  const submitMap = {
    organization: organizationSubmit,
    deal: dealSubmit,
    person: personSubmit,
    task: taskSubmit,
    note: noteSubmit,
    endpoint: endpointSubmit,
    subscription: subscriptionSubmit,
    adminUser: adminUserSubmit,
  };
  const cancelMap = {
    organization: organizationCancel,
    deal: dealCancel,
    person: personCancel,
    task: taskCancel,
    note: noteCancel,
    endpoint: endpointCancel,
    subscription: subscriptionCancel,
    adminUser: adminUserCancel,
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

function startEditEndpoint(id) {
  const item = state.webhookEndpoints.find((entry) => entry.id === id);
  if (!item) return;
  endpointForm.elements.namedItem("name").value = item.name || "";
  endpointForm.elements.namedItem("target_url").value = item.target_url || "";
  endpointForm.elements.namedItem("signing_secret").value = item.signing_secret || "";
  endpointForm.elements.namedItem("status").value = item.status || "active";
  endpointResult.textContent = `Editing endpoint ${item.name}.`;
  setEditMode("endpoint", id);
}

function startEditSubscription(id) {
  const item = state.webhookSubscriptions.find((entry) => entry.id === id);
  if (!item) return;
  subscriptionEndpointSelect.value = item.webhook_endpoint_id || "";
  subscriptionForm.elements.namedItem("event_type").value = item.event_type || "";
  subscriptionForm.elements.namedItem("is_active").value = String(Boolean(item.is_active));
  subscriptionResult.textContent = `Editing subscription ${item.event_type}.`;
  setEditMode("subscription", id);
}

function startEditAdminUser(id) {
  const item = state.adminUsers.find((entry) => entry.id === id);
  if (!item) return;
  adminUserForm.elements.namedItem("full_name").value = item.full_name || "";
  adminUserForm.elements.namedItem("email").value = item.email || "";
  adminUserForm.elements.namedItem("password").value = "";
  adminUserForm.elements.namedItem("role").value = item.role || "member";
  adminUserForm.elements.namedItem("user_status").value = item.user_status || "active";
  adminUserForm.elements.namedItem("membership_status").value = item.membership_status || "active";
  adminUserForm.elements.namedItem("email").disabled = true;
  adminUserForm.elements.namedItem("password").disabled = true;
  adminUserResult.textContent = `Editing user ${item.full_name}.`;
  setEditMode("adminUser", id);
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
    endpoint: "webhook-endpoints",
    subscription: "webhook-subscriptions",
  };
  await api(`/${routeMap[kind]}/${id}`, { method: "DELETE" });
  await loadDashboard();
}

async function loadDashboard() {
  const adminRequests = isAdminSession()
    ? [api("/admin/tenant"), api("/admin/users")]
    : [Promise.resolve(null), Promise.resolve([])];

  const [outboxStats, deliveryStats, failedOutbox, failedDeliveries, audit, organizations, deals, people, tasks, notes, webhookEndpoints, webhookSubscriptions, adminTenant, adminUsers] = await Promise.all([
    api("/outbox-events/stats"),
    api("/webhook-deliveries/stats"),
    api("/outbox-events?status=failed&limit=10"),
    api("/webhook-deliveries?status=failed"),
    api("/audit-log?limit=10"),
    api("/organizations"),
    api("/deals"),
    api("/people"),
    api("/tasks"),
    api("/notes"),
    api("/webhook-endpoints"),
    api("/webhook-subscriptions"),
    ...adminRequests,
  ]);

  state.organizations = organizations || [];
  state.deals = deals || [];
  state.people = people || [];
  state.tasks = tasks || [];
  state.notes = notes || [];
  state.adminTenant = adminTenant;
  state.adminUsers = adminUsers || [];
  state.webhookEndpoints = webhookEndpoints || [];
  state.webhookSubscriptions = webhookSubscriptions || [];
  state.outboxFailed = failedOutbox || [];
  state.deliveryFailed = failedDeliveries || [];
  state.audit = audit || [];

  if (state.selectedOrganizationId != null && !state.organizations.find((item) => item.id === state.selectedOrganizationId)) {
    state.selectedOrganizationId = null;
  }

  renderStatusCards("outbox-status-cards", outboxStats.by_status || []);
  renderStatusCards("delivery-status-cards", deliveryStats.by_status || []);
  renderRankedList("event-type-list", outboxStats.by_event_type || [], "event_type");
  renderRankedList("endpoint-traffic-list", deliveryStats.by_endpoint || [], "endpoint_name");
  renderSalesSnapshot();
  renderCommercialReporting();
  renderSystemActivityStream();
  renderPlatformConsole();
  renderLeadModule();
  renderReportsOverview();
  renderImportsModule();
  renderWorkspaceFocus();
  renderBridgeRail();
  renderTeamHub();
  renderPipeline(state.deals);
  populateOrganizationSelect();
  populateDealSelect();
  populateSubscriptionEndpointSelect();
  renderOrganizations();
  renderDeals();
  renderPeople();
  renderTasks();
  renderNotes();
  renderEventList("outbox-failed-list", state.outboxFailed, "outbox");
  renderEventList("delivery-failed-list", state.deliveryFailed, "delivery");
  renderAudit();
  renderAdminCenter();
  setWorkerStatus((outboxStats.by_status || []).length > 0);
  syncHomeLayoutControls();
  applyHomeLayout();
  applyCurrentPage();
}

function showApp() {
  authPanel.classList.add("hidden");
  appPanel.classList.remove("hidden");
  sessionBadge.textContent = `${state.session.email} @ ${state.session.tenant_slug}`;
  roleBadge.textContent = state.session.role || "member";
  applyWorkspaceMode();
  applyFiltersToControls();
  renderSavedViews();
  syncHomeLayoutControls();
  applyHomeLayout();
  applyCurrentPage();
}

function showLogin() {
  appPanel.classList.add("hidden");
  authPanel.classList.remove("hidden");
}

async function submitOrganization(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(organizationForm).entries());
  if (state.session?.user_id) {
    body.owner_user_id = Number(state.session.user_id);
  }
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
  if (state.session?.user_id) {
    body.owner_user_id = Number(state.session.user_id);
  }
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
  if (state.session?.user_id) {
    body.owner_user_id = Number(state.session.user_id);
  }
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
  if (state.session?.user_id) {
    body.owner_user_id = Number(state.session.user_id);
    body.created_by_user_id = Number(state.session.user_id);
  }
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

async function submitNote(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(noteForm).entries());
  if (body.organization_id) body.organization_id = Number(body.organization_id); else delete body.organization_id;
  if (body.deal_id) body.deal_id = Number(body.deal_id); else delete body.deal_id;
  try {
    const result = await api("/notes", {
      method: "POST",
      body: JSON.stringify(body),
    });
    noteResult.textContent = `Created note ${result.id}.`;
    noteForm.reset();
    if (state.selectedOrganizationId) {
      noteOrganizationSelect.value = String(state.selectedOrganizationId);
    }
    await loadDashboard();
  } catch (error) {
    noteResult.textContent = error.message;
  }
}

async function submitTenant(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(tenantForm).entries());
  try {
    const result = await api("/admin/tenant", {
      method: "PUT",
      body: JSON.stringify(body),
    });
    tenantResult.textContent = `Updated tenant ${result.name}.`;
    await loadDashboard();
  } catch (error) {
    tenantResult.textContent = error.message;
  }
}

async function submitAdminUser(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(adminUserForm).entries());
  const isEdit = state.editing.adminUserId != null;
  if (isEdit) {
    delete body.email;
    delete body.password;
  }

  try {
    const result = await api(isEdit ? `/admin/users/${state.editing.adminUserId}` : "/admin/users", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    adminUserResult.textContent = `${isEdit ? "Updated" : "Created"} user ${result.full_name}.`;
    adminUserForm.reset();
    adminUserForm.elements.namedItem("email").disabled = false;
    adminUserForm.elements.namedItem("password").disabled = false;
    clearEditMode("adminUser");
    await loadDashboard();
  } catch (error) {
    adminUserResult.textContent = error.message;
  }
}

async function submitEndpoint(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(endpointForm).entries());
  try {
    const isEdit = state.editing.endpointId != null;
    const result = await api(isEdit ? `/webhook-endpoints/${state.editing.endpointId}` : "/webhook-endpoints", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    endpointResult.textContent = `${isEdit ? "Updated" : "Created"} endpoint ${result.name}.`;
    endpointForm.reset();
    endpointForm.elements.namedItem("status").value = "active";
    clearEditMode("endpoint");
    await loadDashboard();
  } catch (error) {
    endpointResult.textContent = error.message;
  }
}

async function submitSubscription(event) {
  event.preventDefault();
  const body = Object.fromEntries(new FormData(subscriptionForm).entries());
  body.webhook_endpoint_id = Number(body.webhook_endpoint_id);
  body.is_active = body.is_active === "true";
  try {
    const isEdit = state.editing.subscriptionId != null;
    const result = await api(isEdit ? `/webhook-subscriptions/${state.editing.subscriptionId}` : "/webhook-subscriptions", {
      method: isEdit ? "PUT" : "POST",
      body: JSON.stringify(body),
    });
    subscriptionResult.textContent = `${isEdit ? "Updated" : "Created"} subscription ${result.event_type}.`;
    subscriptionForm.reset();
    subscriptionForm.elements.namedItem("is_active").value = "true";
    clearEditMode("subscription");
    await loadDashboard();
  } catch (error) {
    subscriptionResult.textContent = error.message;
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
    setSession(result.token, {
      email: result.email,
      tenant_slug: result.tenant_slug,
      role: result.role,
      user_id: result.user_id,
    });
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
noteForm.addEventListener("submit", submitNote);
tenantForm.addEventListener("submit", submitTenant);
adminUserForm.addEventListener("submit", submitAdminUser);
endpointForm.addEventListener("submit", submitEndpoint);
subscriptionForm.addEventListener("submit", submitSubscription);

organizationCancel.addEventListener("click", () => cancelEdit("organization", organizationForm, organizationResult));
dealCancel.addEventListener("click", () => cancelEdit("deal", dealForm, dealResult, dealOrganizationSelect));
personCancel.addEventListener("click", () => cancelEdit("person", personForm, personResult, personOrganizationSelect));
taskCancel.addEventListener("click", () => cancelEdit("task", taskForm, taskResult, taskOrganizationSelect));
noteCancel.addEventListener("click", () => cancelEdit("note", noteForm, noteResult, noteOrganizationSelect));
adminUserCancel.addEventListener("click", () => {
  cancelEdit("adminUser", adminUserForm, adminUserResult);
  adminUserForm.elements.namedItem("email").disabled = false;
  adminUserForm.elements.namedItem("password").disabled = false;
});
endpointCancel.addEventListener("click", () => cancelEdit("endpoint", endpointForm, endpointResult));
subscriptionCancel.addEventListener("click", () => cancelEdit("subscription", subscriptionForm, subscriptionResult, subscriptionEndpointSelect));
dealStageFilter.addEventListener("change", () => {
  state.filters.dealStage = dealStageFilter.value;
  persistFilters();
  renderPipeline(state.deals);
});
taskPriorityFilter.addEventListener("change", () => {
  state.filters.taskPriority = taskPriorityFilter.value;
  persistFilters();
  renderTasks();
});
savedViewSelect.addEventListener("change", () => applySavedView(savedViewSelect.value));
saveViewButton.addEventListener("click", saveCurrentView);
deleteViewButton.addEventListener("click", deleteCurrentView);

viewAdmin.addEventListener("click", () => setWorkspaceMode("admin"));
viewTeam.addEventListener("click", () => setWorkspaceMode("team"));

document.addEventListener("click", async (event) => {
  const editButton = event.target.closest("[data-edit-type]");
  if (editButton) {
    const id = Number(editButton.getAttribute("data-edit-id"));
    const type = editButton.getAttribute("data-edit-type");
    if (type === "organization") startEditOrganization(id);
    if (type === "deal") startEditDeal(id);
    if (type === "person") startEditPerson(id);
    if (type === "task") startEditTask(id);
    if (type === "adminUser") startEditAdminUser(id);
    if (type === "endpoint") startEditEndpoint(id);
    if (type === "subscription") startEditSubscription(id);
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

  const focusOrganization = event.target.closest("[data-focus-organization]");
  if (focusOrganization) {
    state.selectedOrganizationId = Number(focusOrganization.getAttribute("data-focus-organization"));
    populateOrganizationSelect();
    renderOrganizations();
    document.getElementById("organizations").scrollIntoView({ behavior: "smooth", block: "start" });
    return;
  }

  const stageButton = event.target.closest("[data-stage-deal-id]");
  if (stageButton) {
    await moveDealToNextStage(
      Number(stageButton.getAttribute("data-stage-deal-id")),
      stageButton.getAttribute("data-stage-next"),
    );
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

if (homeLayoutForm) {
  homeLayoutForm.addEventListener("change", () => {
    state.homeWidgets = {
      summary: Boolean(homeLayoutForm.elements.namedItem("summary")?.checked),
      platform: Boolean(homeLayoutForm.elements.namedItem("platform")?.checked),
      control: Boolean(homeLayoutForm.elements.namedItem("control")?.checked),
      activity: Boolean(homeLayoutForm.elements.namedItem("activity")?.checked),
      snapshot: Boolean(homeLayoutForm.elements.namedItem("snapshot")?.checked),
      posture: Boolean(homeLayoutForm.elements.namedItem("posture")?.checked),
    };
    persistHomeWidgets();
    applyHomeLayout();
  });
}

window.addEventListener("hashchange", applyCurrentPage);

async function boot() {
  if (!state.token || !state.session) {
    showLogin();
    return;
  }

  try {
    if (!window.location.hash) {
      window.location.hash = "#home";
    }
    showApp();
    await loadDashboard();
  } catch (_error) {
    clearSession();
    showLogin();
  }
}

boot();
