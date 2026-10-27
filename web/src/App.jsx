import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  DEFAULT_API_BASE,
  enrollAgent,
  getApiBase,
  getApiToken,
  validateApiBase,
} from "./api.js";
import {
  BrowserRouter,
  NavLink,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";
import {
  FiActivity,
  FiAlertTriangle,
  FiBarChart2,
  FiCalendar,
  FiClock,
  FiLayers,
  FiDownload,
  FiCheck,
  FiChevronRight,
  FiCommand,
  FiCpu,
  FiFile,
  FiFileText,
  FiHardDrive,
  FiLayout,
  FiMoon,
  FiMonitor,
  FiPackage,
  FiPlay,
  FiPlus,
  FiRefreshCw,
  FiSearch,
  FiServer,
  FiSettings,
  FiShield,
  FiSun,
  FiTerminal,
  FiTrash2,
  FiUploadCloud,
  FiWifi,
  FiX,
  FiZap,
} from "react-icons/fi";

import {
  deleteFile,
  getClients,
  getClientDevices,
  getClientStats,
  getFiles,
  getPolicies,
  getSchedules,
  createPolicy,
  createSchedule,
  deletePolicy,
  runPolicy,
  deleteSchedule,
  setScheduleEnabled,
  install,
  uploadFile,
  wakeOnLan,
} from "./api.js";
import { useI18n } from "./i18n/LanguageProvider.jsx";
import LanguageSelector from "./i18n/LanguageSelector.jsx";
import "./index.css";

const NAVIGATION = [
  { to: "/", key: "nav.dashboard", icon: FiLayout, end: true },
  { to: "/clients", key: "nav.clients", icon: FiMonitor },
  { to: "/files", key: "nav.files", icon: FiFileText },
  { to: "/install", key: "nav.software", icon: FiPackage },
  { to: "/devices", key: "nav.devices", icon: FiCpu },
  { to: "/analytics", key: "nav.analytics", icon: FiBarChart2 },
  { to: "/automation", key: "nav.automation", icon: FiLayers },
  { to: "/settings", key: "nav.settings", icon: FiSettings },
];

const TITLES = {
  "/": ["page.dashboard.title", "page.dashboard.description"],
  "/clients": ["page.clients.title", "page.clients.description"],
  "/files": ["page.files.title", "page.files.description"],
  "/install": ["page.install.title", "page.install.description"],
  "/devices": ["page.devices.title", "page.devices.description"],
  "/analytics": ["page.analytics.title", "page.analytics.description"],
  "/automation": ["page.automation.title", "page.automation.description"],
  "/settings": ["page.settings.title", "page.settings.description"],
};

function BrandMark({ size = 38 }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 40 40"
      fill="none"
      role="img"
      aria-label="LANdapter"
    >
      <rect width="40" height="40" rx="12" fill="#0E9F91" />
      <path
        d="M20 20L11 11M20 20L29 11M20 20L11 29M20 20L29 29"
        stroke="#D8FFF7"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <rect x="15" y="15" width="10" height="10" rx="3" fill="white" />
      <circle cx="11" cy="11" r="3" fill="#A7F3D0" />
      <circle cx="29" cy="11" r="3" fill="#A7F3D0" />
      <circle cx="11" cy="29" r="3" fill="#A7F3D0" />
      <circle cx="29" cy="29" r="3" fill="#A7F3D0" />
    </svg>
  );
}

function useTheme() {
  const [theme, setTheme] = useState(
    () => localStorage.getItem("theme") || "dark"
  );

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("theme", theme);
  }, [theme]);

  return {
    theme,
    toggleTheme: () =>
      setTheme((current) => (current === "dark" ? "light" : "dark")),
  };
}

function useResource(loader, dependencies = []) {
  const { t } = useI18n();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [revision, setRevision] = useState(0);

  const refresh = useCallback(() => {
    setRevision((value) => value + 1);
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    let active = true;

    async function run() {
      setLoading(true);
      setError("");

      try {
        const result = await loader(controller.signal);

        if (active) {
          setData(result ?? []);
        }
      } catch (err) {
        if (active && err.name !== "AbortError") {
          setError(err.message || t("common.refreshError"));
        }
      } finally {
        if (active) setLoading(false);
      }
    }

    run();

    return () => {
      active = false;
      controller.abort();
    };
    // Loader identity is deliberately managed by the caller.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [...dependencies, revision]);

  return { data, loading, error, refresh, setData };
}

function formatDate(value) {
  if (!value) return "—";

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";

  const locale = document.documentElement.lang || "en";
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(date);
}

function formatBytes(bytes = 0) {
  if (!Number.isFinite(Number(bytes)) || Number(bytes) < 0) return "—";

  const value = Number(bytes);
  if (value < 1024) return `${value} ${document.documentElement.lang === "zh-CN" ? "字节" : document.documentElement.lang === "ru" ? "Б" : "B"}`;

  const lang = document.documentElement.lang || "en";
  const units = lang === "ru" ? ["КБ", "МБ", "ГБ", "ТБ"] : ["KB", "MB", "GB", "TB"];
  let size = value / 1024;
  let unit = 0;

  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }

  return `${size.toLocaleString(document.documentElement.lang || "en", { maximumFractionDigits: 1 })} ${units[unit]}`;
}

function Status({ online }) {
  const { t } = useI18n();
  return (
    <span className={`status ${online ? "status-online" : "status-offline"}`}>
      <span className="status-dot" />
      {online ? t("common.online") : t("common.offline")}
    </span>
  );
}

function Button({
  children,
  variant = "secondary",
  icon: Icon,
  loading = false,
  disabled = false,
  ...props
}) {
  return (
    <button
      className={`button button-${variant}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <FiRefreshCw className="spin" /> : Icon ? <Icon /> : null}
      {children}
    </button>
  );
}

function EmptyState({ icon: Icon = FiFile, title, description, action }) {
  return (
    <div className="empty-state">
      <div className="empty-icon"><Icon size={25} /></div>
      <h3>{title}</h3>
      <p>{description}</p>
      {action}
    </div>
  );
}

function PageHeading({ eyebrow = "LANDAPTER", title, description, actions }) {
  return (
    <div className="page-heading">
      <div>
        <div className="eyebrow">{eyebrow}</div>
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      {actions && <div className="heading-actions">{actions}</div>}
    </div>
  );
}

function AppShell({ theme, toggleTheme, children }) {
  const location = useLocation();
  const { language, t } = useI18n();
  const [mobileNav, setMobileNav] = useState(false);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const timer = setInterval(() => setNow(new Date()), 30_000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => setMobileNav(false), [location.pathname]);

  const [titleKey, subtitleKey] = TITLES[location.pathname] || [
    "app.title",
    "page.dashboard.description",
  ];
  const title = t(titleKey);
  const subtitle = t(subtitleKey);

  return (
    <div className="app-shell">
      {mobileNav && (
        <button
          className="mobile-overlay"
          aria-label={t("mobile.closeMenu")}
          onClick={() => setMobileNav(false)}
        />
      )}

      <aside className={`sidebar ${mobileNav ? "sidebar-open" : ""}`}>
        <NavLink to="/" className="brand">
          <BrandMark />
          <span className="brand-copy">
            <strong>LANdapter</strong>
            <small>INFRASTRUCTURE CONTROL</small>
          </span>
        </NavLink>

        <div className="nav-label">{t("workspace.label")}</div>
        <nav className="nav-list" aria-label={t("nav.main")}>
          {NAVIGATION.map(({ to, key, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `nav-link ${isActive ? "nav-link-active" : ""}`
              }
            >
              <Icon size={18} />
              <span>{t(key)}</span>
              <FiChevronRight className="nav-chevron" />
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-bottom">
          <div className="sidebar-status">
            <span className="status-dot status-dot-live" />
            <div>
              <strong>{t("sidebar.localConsole")}</strong>
              <small>API {getApiBase()}</small>
            </div>
          </div>
          <button className="theme-button" onClick={toggleTheme}>
            {theme === "dark" ? <FiSun /> : <FiMoon />}
            <span>{theme === "dark" ? t("theme.light") : t("theme.dark")}</span>
          </button>
        </div>
      </aside>

      <main className="main-panel">
        <header className="topbar">
          <button
            className="mobile-menu"
            onClick={() => setMobileNav(true)}
            aria-label={t("mobile.openMenu")}
          >
            <FiLayout />
          </button>
          <div className="breadcrumbs">
            <span>LANdapter</span>
            <FiChevronRight />
            <strong>{title}</strong>
          </div>
          <div className="topbar-right">
            <LanguageSelector className="topbar-language" />
            <span className="topbar-time">
              {now.toLocaleTimeString(language === "ru" ? "ru-RU" : language === "zh-CN" ? "zh-CN" : "en-US", {
                hour: "2-digit",
                minute: "2-digit",
              })}
            </span>
            <span className="environment-chip">
              <span className="status-dot" />
              MASTER
            </span>
          </div>
        </header>

        <div className="page-content">
          <div className="mobile-page-title">
            <h1>{title}</h1>
            <p>{subtitle}</p>
          </div>
          {children}
          <footer className="footer">
            <span>LANdapter · Infrastructure management</span>
            <span>{t("footer.localNetwork")}</span>
          </footer>
        </div>
      </main>
    </div>
  );
}

function ResourceError({ message, onRetry }) {
  const { t } = useI18n();
  if (!message) return null;

  return (
    <div className="error-banner" role="alert">
      <FiShield size={18} />
      <div>
        <strong>{t("common.refreshError")}</strong>
        <p>{message}</p>
      </div>
      <Button icon={FiRefreshCw} onClick={onRetry}>{t("common.retry")}</Button>
    </div>
  );
}

function Dashboard() {
  const { t } = useI18n();
  const clients = useResource((signal) => getClients(signal));
  const files = useResource((signal) => getFiles(signal));

  const clientList = Array.isArray(clients.data) ? clients.data : [];
  const fileList = Array.isArray(files.data) ? files.data : [];
  const online = clientList.filter((client) => client.online).length;
  const offline = clientList.length - online;

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.overview")}
        title={t("page.dashboard.title")}
        description={t("page.dashboard.description")}
        actions={
          <Button
            icon={FiRefreshCw}
            onClick={() => {
              clients.refresh();
              files.refresh();
            }}
          >
            {t("common.refreshData")}
          </Button>
        }
      />

      <ResourceError message={clients.error} onRetry={clients.refresh} />
      <ResourceError message={files.error} onRetry={files.refresh} />

      <div className="metrics-grid">
        <Metric
          label={t("common.totalComputers")}
          value={clients.loading ? "—" : clientList.length}
          icon={FiMonitor}
          detail={t("common.registeredAgents")}
          accent="teal"
        />
        <Metric
          label={t("common.online")}
          value={clients.loading ? "—" : online}
          icon={FiWifi}
          detail={t("common.availableNodes")}
          accent="green"
        />
        <Metric
          label={t("common.offline")}
          value={clients.loading ? "—" : offline}
          icon={FiActivity}
          detail={t("common.noActiveConnection")}
          accent="amber"
        />
        <Metric
          label={t("common.files")}
          value={files.loading ? "—" : fileList.length}
          icon={FiHardDrive}
          detail={t("common.deploymentLibrary")}
          accent="blue"
        />
      </div>

      <div className="content-grid">
        <section className="panel panel-main">
          <div className="panel-header">
            <div>
              <h2>{t("dashboard.computers")}</h2>
              <p>{t("dashboard.recentNodes")}</p>
            </div>
            <NavLink className="text-link" to="/clients">
              {t("dashboard.allComputers")} <FiChevronRight />
            </NavLink>
          </div>

          {clients.loading ? (
            <LoadingRows />
          ) : clientList.length === 0 && !clients.error ? (
            <EmptyState
              icon={FiMonitor}
              title={t("dashboard.noAgents")}
              description={t("dashboard.connectAgent")}
            />
          ) : (
            <ClientTable clients={clientList.slice(0, 6)} />
          )}
        </section>

        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>{t("dashboard.systemStatus")}</h2>
              <p>{t("dashboard.resourceSummary")}</p>
            </div>
            <FiActivity className="panel-heading-icon" />
          </div>

          <div className="health-list">
            <HealthItem
              label="Master API"
              description="REST API"
              status={clients.error ? "error" : clients.loading ? "loading" : "ok"}
            />
            <HealthItem
              label={t("dashboard.agents")}
              description={t("dashboard.registrationConnections")}
              status={clients.error ? "error" : clients.loading ? "loading" : "ok"}
            />
            <HealthItem
              label={t("dashboard.library")}
              description={t("dashboard.packageStorage")}
              status={files.error ? "error" : files.loading ? "loading" : "ok"}
            />
          </div>

          <div className="callout">
            <FiShield size={20} />
            <div>
              <strong>{t("dashboard.securityCheck")}</strong>
              <p>
                {t("dashboard.securityAdvice")}
              </p>
            </div>
          </div>
        </section>
      </div>
    </>
  );
}

function Metric({ label, value, icon: Icon, detail, accent }) {
  return (
    <section className={`metric-card metric-${accent}`}>
      <div className="metric-top">
        <span>{label}</span>
        <span className="metric-icon"><Icon size={19} /></span>
      </div>
      <div className="metric-value">{value}</div>
      <div className="metric-detail">{detail}</div>
    </section>
  );
}

function HealthItem({ label, description, status }) {
  const { t } = useI18n();
  const text = {
    ok: t("health.available"),
    error: t("health.error"),
    loading: t("health.checking"),
  }[status];

  return (
    <div className="health-item">
      <span className={`health-indicator health-${status}`} />
      <div className="health-copy">
        <strong>{label}</strong>
        <small>{description}</small>
      </div>
      <span className={`health-label health-label-${status}`}>{text}</span>
    </div>
  );
}

function LoadingRows() {
  const { t } = useI18n();
  return (
    <div className="loading-rows" aria-label={t("common.loading")}>
      {[1, 2, 3, 4].map((item) => (
        <div className="skeleton-row" key={item}>
          <span />
          <span />
          <span />
        </div>
      ))}
    </div>
  );
}

function ClientTable({ clients, selected = [], onToggle }) {
  const { t } = useI18n();
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {onToggle && <th aria-label={t("clients.select")} />}
            <th>{t("common.computer")}</th>
            <th>{t("common.operatingSystem")}</th>
            <th>{t("common.macAddress")}</th>
            <th>{t("common.status")}</th>
            <th>{t("common.lastActivity")}</th>
          </tr>
        </thead>
        <tbody>
          {clients.map((client) => (
            <tr key={client.id}>
              {onToggle && (
                <td>
                  <input
                    type="checkbox"
                    checked={selected.includes(client.id)}
                    onChange={() => onToggle(client.id)}
                    aria-label={t("clients.selectClient", { hostname: client.hostname })}
                  />
                </td>
              )}
              <td>
                <div className="client-name">
                  <span className="client-avatar"><FiMonitor /></span>
                  <div>
                    <strong>{client.hostname || t("common.noName")}</strong>
                    <small title={client.id}>{String(client.id).slice(0, 12)}</small>
                  </div>
                </div>
              </td>
              <td>{client.os || t("common.unknown")}</td>
              <td><code>{client.mac || "—"}</code></td>
              <td><Status online={Boolean(client.online)} /></td>
              <td>{formatDate(client.last_seen)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Clients() {
  const { t } = useI18n();
  const resource = useResource((signal) => getClients(signal));
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("all");
  const [selected, setSelected] = useState([]);
  const [busy, setBusy] = useState(false);
  const [runningPolicyId, setRunningPolicyId] = useState("");
  const [notice, setNotice] = useState("");

  const clients = Array.isArray(resource.data) ? resource.data : [];
  const filtered = useMemo(() => {
    const query = search.toLowerCase().trim();

    return clients.filter((client) => {
      const matchesQuery = [
        client.hostname,
        client.os,
        client.mac,
        client.id,
      ].some((value) => String(value || "").toLowerCase().includes(query));

      const matchesStatus =
        filter === "all" ||
        (filter === "online" && client.online) ||
        (filter === "offline" && !client.online);

      return matchesQuery && matchesStatus;
    });
  }, [clients, search, filter]);

  function toggle(id) {
    setSelected((current) =>
      current.includes(id)
        ? current.filter((value) => value !== id)
        : [...current, id]
    );
  }

  async function wakeSelected() {
    setBusy(true);
    setNotice("");

    try {
      await wakeOnLan(selected);
      setNotice(t("clients.wolSent"));
    } catch (error) {
      setNotice(t("clients.wolError", { error: error.message }));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.infrastructure")}
        title={t("page.clients.title")}
        description={t("page.clients.description")}
        actions={
          <Button icon={FiRefreshCw} onClick={resource.refresh}>
            {t("common.refresh")}
          </Button>
        }
      />

      <ResourceError message={resource.error} onRetry={resource.refresh} />
      {notice && (
        <div className="notice-banner" role="status">
          <span>{notice}</span>
          <button onClick={() => setNotice("")} aria-label={t("common.close")}>
            <FiX />
          </button>
        </div>
      )}

      <div className="toolbar panel">
        <label className="search-box">
          <FiSearch />
          <input
            value={search}
            onChange={(event) => setSearch(event.target.value)}
            placeholder={t("common.searchClients")}
          />
          {search && (
            <button onClick={() => setSearch("")} aria-label={t("common.clearSearch")}>
              <FiX />
            </button>
          )}
        </label>

        <select value={filter} onChange={(event) => setFilter(event.target.value)}>
          <option value="all">{t("common.allNodes")}</option>
          <option value="online">{t("common.online")}</option>
          <option value="offline">{t("common.offline")}</option>
        </select>

        {selected.length > 0 && (
          <Button icon={FiZap} loading={busy} onClick={wakeSelected}>
            Wake-on-LAN ({selected.length})
          </Button>
        )}
      </div>

      <section className="panel">
        <div className="panel-header">
          <div>
            <h2>{t("clients.registry")}</h2>
            <p>
              {resource.loading
                ? t("common.loadingData")
                : t("clients.shown", { shown: filtered.length, total: clients.length })}
            </p>
          </div>
          <span className="count-chip">{clients.length} {t("clients.count", { count: clients.length })}</span>
        </div>

        {resource.loading ? (
          <LoadingRows />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={FiMonitor}
            title={clients.length ? t("clients.noneFound") : t("clients.noneRegistered")}
            description={
              clients.length
                ? t("clients.changeFilter")
                : t("clients.agentsAppear")
            }
          />
        ) : (
          <ClientTable
            clients={filtered}
            selected={selected}
            onToggle={toggle}
          />
        )}
      </section>
    </>
  );
}

function Files() {
  const { t } = useI18n();
  const resource = useResource((signal) => getFiles(signal));
  const [search, setSearch] = useState("");
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const files = Array.isArray(resource.data) ? resource.data : [];
  const filtered = files.filter((file) =>
    String(file.name || "").toLowerCase().includes(search.toLowerCase().trim())
  );

  async function handleUpload(event) {
    const input = event.currentTarget;
    const file = input.files?.[0];
    if (!file) return;

    setUploading(true);
    setError("");
    setMessage("");

    try {
      await uploadFile(file);
      setMessage(t("files.uploaded", { name: file.name }));
      resource.refresh();
    } catch (err) {
      setError(err.message || t("common.uploadError"));
    } finally {
      setUploading(false);
      input.value = "";
    }
  }

  async function handleDelete(file) {
    if (!window.confirm(t("files.deleteConfirm", { name: file.name }))) return;

    setError("");
    setMessage("");

    try {
      await deleteFile(file.id);
      resource.setData((current) =>
        current.filter((item) => item.id !== file.id)
      );
      setMessage(t("files.deleted"));
    } catch (err) {
      setError(err.message || t("common.deleteError"));
    }
  }

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.artifacts")}
        title={t("page.files.title")}
        description={t("page.files.description")}
        actions={
          <label className={`button button-primary upload-button ${uploading ? "disabled" : ""}`}>
            {uploading ? <FiRefreshCw className="spin" /> : <FiUploadCloud />}
            {uploading ? t("common.uploading") : t("common.upload")}
            <input
              type="file"
              onChange={handleUpload}
              disabled={uploading}
              hidden
            />
          </label>
        }
      />

      <ResourceError message={resource.error} onRetry={resource.refresh} />
      {error && <div className="error-banner" role="alert">{error}</div>}
      {message && (
        <div className="notice-banner" role="status">
          {message}
          <button onClick={() => setMessage("")} aria-label={t("common.close")}>
            <FiX />
          </button>
        </div>
      )}

      <div className="file-summary-grid">
        <div className="file-summary">
          <FiHardDrive />
          <div><span>{t("files.total")}</span><strong>{files.length}</strong></div>
        </div>
        <div className="file-summary">
          <FiPackage />
          <div><span>{t("files.totalSize")}</span><strong>{formatBytes(
            files.reduce((total, file) => total + Number(file.size || 0), 0)
          )}</strong></div>
        </div>
        <div className="file-summary">
          <FiShield />
          <div><span>{t("files.storage")}</span><strong>Master</strong></div>
        </div>
      </div>

      <section className="panel">
        <div className="panel-header file-panel-header">
          <div>
            <h2>{t("files.listTitle")}</h2>
            <p>{t("files.uploadedArtifacts")}</p>
          </div>
          <label className="search-box compact-search">
            <FiSearch />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t("files.search")}
            />
          </label>
        </div>

        {resource.loading ? (
          <LoadingRows />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={FiFile}
            title={files.length ? t("files.notFound") : t("files.empty")}
            description={
              files.length
                ? t("files.changeSearch")
                : t("files.uploadPrompt")
            }
            action={
              <label className="button button-secondary upload-button">
                <FiPlus /> {t("common.addFile")}
                <input type="file" onChange={handleUpload} hidden disabled={uploading} />
              </label>
            }
          />
        ) : (
          <div className="file-list">
            {filtered.map((file) => (
              <div className="file-row" key={file.id}>
                <div className="file-type-icon"><FiFileText size={21} /></div>
                <div className="file-info">
                  <strong>{file.name}</strong>
                  <span>
                    {(file.type || "FILE").toUpperCase()} · {formatBytes(file.size)}
                  </span>
                </div>
                <div className="file-date">{formatDate(file.uploadedAt || file.uploaded_at)}</div>
                <button
                  className="icon-button danger-icon"
                  onClick={() => handleDelete(file)}
                  title={t("files.delete")}
                  aria-label={`${t("files.delete")}: ${file.name}`}
                >
                  <FiTrash2 />
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
    </>
  );
}

function Install() {
  const { t } = useI18n();
  const clientsResource = useResource((signal) => getClients(signal));
  const filesResource = useResource((signal) => getFiles(signal));

  const clients = Array.isArray(clientsResource.data) ? clientsResource.data : [];
  const files = Array.isArray(filesResource.data) ? filesResource.data : [];

  const [clientIds, setClientIds] = useState([]);
  const [fileIds, setFileIds] = useState([]);
  const [mode, setMode] = useState("quiet");
  const [busy, setBusy] = useState(false);
  const [jobId, setJobId] = useState("");
  const [error, setError] = useState("");

  function toggleValue(value, setter) {
    setter((current) =>
      current.includes(value)
        ? current.filter((item) => item !== value)
        : [...current, value]
    );
  }

  async function submit(event) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setJobId("");

    try {
      const result = await install(fileIds, clientIds, mode);
      setJobId(result.job_id || result.jobId || "");
    } catch (err) {
      setError(err.message || t("common.installError"));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.deployment")}
        title={t("page.install.title")}
        description={t("page.install.description")}
      />

      <ResourceError message={clientsResource.error} onRetry={clientsResource.refresh} />
      <ResourceError message={filesResource.error} onRetry={filesResource.refresh} />
      {error && <div className="error-banner" role="alert">{error}</div>}
      {jobId && (
        <div className="notice-banner" role="status">
          <FiCheck />
          <span>{t("install.jobCreated")} <code>{jobId}</code></span>
        </div>
      )}

      <form className="deploy-layout" onSubmit={submit}>
        <div className="deploy-main">
          <section className="panel">
            <div className="step-heading">
              <span className="step-number">01</span>
              <div><h2>{t("install.choosePackages")}</h2><p>{t("install.whatToInstall")}</p></div>
            </div>

            {files.length === 0 ? (
              <EmptyState
                icon={FiPackage}
                title={t("install.noPackages")}
                description={t("install.uploadFirst")}
              />
            ) : (
              <div className="selection-list">
                {files.map((file) => (
                  <label className="selection-row" key={file.id}>
                    <input
                      type="checkbox"
                      checked={fileIds.includes(file.id)}
                      onChange={() => toggleValue(file.id, setFileIds)}
                    />
                    <span className="selection-icon"><FiFileText /></span>
                    <span className="selection-copy">
                      <strong>{file.name}</strong>
                      <small>{formatBytes(file.size)} · {(file.type || "FILE").toUpperCase()}</small>
                    </span>
                    <FiCheck className="selection-check" />
                  </label>
                ))}
              </div>
            )}
          </section>

          <section className="panel">
            <div className="step-heading">
              <span className="step-number">02</span>
              <div><h2>{t("install.targetComputers")}</h2><p>{t("install.selectTargets")}</p></div>
            </div>

            {clients.length === 0 ? (
              <EmptyState
                icon={FiMonitor}
                title={t("install.noComputers")}
                description={t("install.connectAgents")}
              />
            ) : (
              <>
                <div className="selection-toolbar">
                  <span>{t("install.selectedCount", { selected: clientIds.length, total: clients.length })}</span>
                  <button
                    type="button"
                    className="text-button"
                    onClick={() =>
                      setClientIds((current) =>
                        current.length === clients.length
                          ? []
                          : clients.map((client) => client.id)
                      )
                    }
                  >
                    {clientIds.length === clients.length ? t("install.deselectAll") : t("install.selectAll")}
                  </button>
                </div>
                <div className="selection-list">
                  {clients.map((client) => (
                    <label className="selection-row" key={client.id}>
                      <input
                        type="checkbox"
                        checked={clientIds.includes(client.id)}
                        onChange={() => toggleValue(client.id, setClientIds)}
                      />
                      <span className="selection-icon"><FiMonitor /></span>
                      <span className="selection-copy">
                        <strong>{client.hostname || client.id}</strong>
                        <small>{client.os || t("install.unknownOS")}</small>
                      </span>
                      <Status online={Boolean(client.online)} />
                    </label>
                  ))}
                </div>
              </>
            )}
          </section>
        </div>

        <aside className="deploy-aside">
          <section className="panel">
            <div className="panel-header">
              <div><h2>{t("install.options")}</h2><p>{t("install.executionMode")}</p></div>
              <FiSettings className="panel-heading-icon" />
            </div>

            <label className="mode-option">
              <input
                type="radio"
                name="mode"
                value="quiet"
                checked={mode === "quiet"}
                onChange={() => setMode("quiet")}
              />
              <span className="mode-icon"><FiZap /></span>
              <span><strong>{t("install.silent")}</strong><small>{t("install.noInteractive")}</small></span>
            </label>

            <label className="mode-option">
              <input
                type="radio"
                name="mode"
                value="interactive"
                checked={mode === "interactive"}
                onChange={() => setMode("interactive")}
              />
              <span className="mode-icon"><FiMonitor /></span>
              <span><strong>{t("install.interactive")}</strong><small>{t("install.withUI")}</small></span>
            </label>

            <div className="deploy-summary">
              <div><span>{t("install.packages")}</span><strong>{fileIds.length}</strong></div>
              <div><span>{t("install.computers")}</span><strong>{clientIds.length}</strong></div>
              <div><span>{t("install.mode")}</span><strong>{mode === "quiet" ? t("install.silentMode") : t("install.interactiveMode")}</strong></div>
            </div>

            <Button
              type="submit"
              variant="primary"
              icon={FiDownload}
              loading={busy}
              disabled={!fileIds.length || !clientIds.length}
            >
              {t("install.createJob")}
            </Button>

            <p className="form-hint">
              {t("install.osHint")}
            </p>
          </section>
        </aside>
      </form>
    </>
  );
}

function Settings() {
  const { t } = useI18n();
  const [apiUrl, setApiUrl] = useState(
    () => localStorage.getItem("api_url") || DEFAULT_API_BASE
  );
  const [apiToken, setApiToken] = useState(() => getApiToken());
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [testing, setTesting] = useState(false);
  const [enrolling, setEnrolling] = useState(false);
  const [agentId, setAgentId] = useState("");
  const [issuedCredential, setIssuedCredential] = useState(null);

  function saveConnection(event) {
    event.preventDefault();
    setError("");
    setNotice("");

    try {
      const normalizedUrl = validateApiBase(apiUrl);

      localStorage.setItem("api_url", normalizedUrl);

      if (apiToken.trim()) {
        sessionStorage.setItem("api_token", apiToken.trim());
      } else {
        sessionStorage.removeItem("api_token");
      }

      setApiUrl(normalizedUrl);
      setApiToken(apiToken.trim());
      setNotice(t("settings.saved"));
    } catch (err) {
      setError(err.message || t("settings.invalid"));
    }
  }

  async function testConnection() {
    setTesting(true);
    setError("");
    setNotice("");

    try {
      const base = validateApiBase(apiUrl);
      const token = apiToken.trim();

      const response = await fetch(`${base}/clients`, {
        headers: token
          ? { Authorization: `Bearer ${token}` }
          : {},
        credentials: "same-origin",
        redirect: "error",
      });

      if (!response.ok) {
        const responseText = (await response.text()).trim();

        if (response.status === 401) {
          throw new Error(t("settings.invalidToken"));
        }

        throw new Error(responseText || `HTTP ${response.status}`);
      }

      setNotice(t("settings.connected"));
    } catch (err) {
      setError(t("common.connectionCheckFailed", { error: err.message }));
    } finally {
      setTesting(false);
    }
  }

  async function issueAgentCredential() {
    setEnrolling(true);
    setError("");
    setNotice("");
    setIssuedCredential(null);

    try {
      const normalizedUrl = validateApiBase(apiUrl);
      const token = apiToken.trim();

      if (!token) {
        throw new Error(t("settings.tokenRequired"));
      }

      // enrollAgent() использует сохранённые настройки API и token.
      localStorage.setItem("api_url", normalizedUrl);
      sessionStorage.setItem("api_token", token);
      setApiUrl(normalizedUrl);

      const credential = await enrollAgent(agentId);
      setIssuedCredential(credential);
      setNotice(
        t("settings.credentialIssued")
      );
    } catch (err) {
      setError(t("common.enrollFailed", { error: err.message }));
    } finally {
      setEnrolling(false);
    }
  }

  async function copyCredential() {
    if (!issuedCredential) return;

    try {
      await navigator.clipboard.writeText(
        JSON.stringify(issuedCredential, null, 2)
      );
      setNotice(t("settings.credentialCopied"));
    } catch {
      setError(t("settings.copyFailed"));
    }
  }

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.preferences")}
        title={t("page.settings.title")}
        description={t("page.settings.description")}
      />

      {error && (
        <div className="error-banner" role="alert">
          <FiShield />
          <div>
            <strong>{t("common.error")}</strong>
            <p>{error}</p>
          </div>
        </div>
      )}

      {notice && (
        <div className="notice-banner" role="status">
          <span>{notice}</span>
          <button onClick={() => setNotice("")} aria-label={t("common.close")}>
            <FiX />
          </button>
        </div>
      )}

      <div className="settings-layout">
        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>{t("settings.connection")}</h2>
              <p>{t("settings.connectionDescription")}</p>
            </div>
            <FiServer className="panel-heading-icon" />
          </div>

          <form onSubmit={saveConnection} className="settings-form">
            <label className="field-label" htmlFor="api-url">
              {t("settings.apiURL")}
            </label>
            <input
              id="api-url"
              className="text-input"
              value={apiUrl}
              onChange={(event) => setApiUrl(event.target.value)}
              placeholder="/api/v1"
              required
            />

            <label
              className="field-label"
              htmlFor="admin-api-token"
              style={{ marginTop: 18 }}
            >
              {t("settings.adminToken")}
            </label>
            <input
              id="admin-api-token"
              className="text-input"
              type="password"
              autoComplete="current-password"
              value={apiToken}
              onChange={(event) => setApiToken(event.target.value)}
              placeholder={t("settings.tokenPlaceholder")}
            />

            <p className="field-help">
              {t("settings.tokenHelp")}
            </p>

            <div className="form-actions">
              <Button type="submit" variant="primary" icon={FiCheck}>
                {t("settings.save")}
              </Button>
              <Button
                type="button"
                icon={FiActivity}
                loading={testing}
                onClick={testConnection}
              >
                {t("settings.test")}
              </Button>
            </div>
          </form>
        </section>

        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>{t("settings.registerAgent")}</h2>
              <p>{t("settings.registerAgentDescription")}</p>
            </div>
            <FiShield className="panel-heading-icon" />
          </div>

          <div className="settings-form">
            <label className="field-label" htmlFor="agent-id">
              {t("settings.existingUUID")}
            </label>
            <input
              id="agent-id"
              className="text-input"
              value={agentId}
              onChange={(event) => setAgentId(event.target.value)}
              placeholder={t("settings.newUUID")}
            />

            <p className="field-help">
              {t("settings.rotateWarning")}
            </p>

            <div className="form-actions">
              <Button
                type="button"
                variant="primary"
                icon={FiPlus}
                loading={enrolling}
                onClick={issueAgentCredential}
              >
                {t("settings.issueToken")}
              </Button>
            </div>

            {issuedCredential && (
              <div style={{ marginTop: 18 }}>
                <label className="field-label">
                  {t("settings.credentialsLabel")}
                </label>
                <textarea
                  className="text-input"
                  style={{ height: "auto", minHeight: 115, padding: 12 }}
                  rows={5}
                  readOnly
                  value={JSON.stringify(issuedCredential, null, 2)}
                />
                <div className="form-actions">
                  <Button
                    type="button"
                    icon={FiFileText}
                    onClick={copyCredential}
                  >
                    {t("settings.copyJSON")}
                  </Button>
                </div>
              </div>
            )}
          </div>
        </section>
      </div>
    </>
  );
}

function parseMaybeJSON(value) {
  if (typeof value !== "string") return value;
  try { return JSON.parse(value); } catch { return value; }
}

function normalizeDeviceList(value) {
  const parsed = parseMaybeJSON(value);
  if (Array.isArray(parsed)) return parsed;
  if (parsed && Array.isArray(parsed.devices)) return parsed.devices;
  return [];
}

function normalizeStats(value) {
  const parsed = parseMaybeJSON(value);
  return parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed : {};
}

function useDefaultClientSelection(clients, selectedClientId, setSelectedClientId) {
  useEffect(() => {
    if (!clients.length) {
      if (selectedClientId) setSelectedClientId("");
      return;
    }
    if (!clients.some((client) => client.id === selectedClientId)) {
      const preferred = clients.find((client) => client.online) || clients[0];
      setSelectedClientId(preferred.id);
    }
  }, [clients, selectedClientId, setSelectedClientId]);
}

function ClientPicker({ clients, value, onChange, label }) {
  return (
    <label className="field-inline">
      <span>{label}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)} disabled={!clients.length}>
        {clients.length === 0 && <option value="">—</option>}
        {clients.map((client) => (
          <option key={client.id} value={client.id}>
            {client.hostname || client.id}{client.online ? " · online" : " · offline"}
          </option>
        ))}
      </select>
    </label>
  );
}

function Devices() {
  const { t } = useI18n();
  const clientsResource = useResource((signal) => getClients(signal));
  const clients = Array.isArray(clientsResource.data) ? clientsResource.data : [];
  const [selectedClientId, setSelectedClientId] = useState("");
  useDefaultClientSelection(clients, selectedClientId, setSelectedClientId);
  const selectedClient = clients.find((client) => client.id === selectedClientId);
  const selectedClientOnline = Boolean(selectedClient?.online);
  const inventory = useResource(
    (signal) => selectedClientId && selectedClientOnline
      ? getClientDevices(selectedClientId, signal)
      : Promise.resolve([]),
    [selectedClientId, selectedClientOnline]
  );
  const parsedInventory = parseMaybeJSON(inventory.data);
  const devices = normalizeDeviceList(parsedInventory);
  const inventoryWarning = parsedInventory && !Array.isArray(parsedInventory) && typeof parsedInventory.warning === "string" ? parsedInventory.warning : "";
  const issues = devices.filter((device) => Boolean(device.driver_issue) || /driver-missing|error|problem/i.test(String(device.status || "")));

  return (
    <>
      <PageHeading
        eyebrow={t("eyebrow.infrastructure")}
        title={t("page.devices.title")}
        description={t("page.devices.description")}
        actions={<Button icon={FiRefreshCw} onClick={() => { clientsResource.refresh(); inventory.refresh(); }}>{t("common.refresh")}</Button>}
      />
      <ResourceError message={clientsResource.error} onRetry={clientsResource.refresh} />
      <ResourceError message={inventory.error} onRetry={inventory.refresh} />
      {inventoryWarning && <div className="notice-banner" role="status"><FiAlertTriangle /><span>{inventoryWarning}</span></div>}
      <section className="panel inventory-toolbar">
        <ClientPicker clients={clients} value={selectedClientId} onChange={setSelectedClientId} label={t("devices.client")} />
        {selectedClient && <div className="inventory-host"><strong>{selectedClient.hostname}</strong><span>{selectedClient.os || t("common.unknown")} · {selectedClient.online ? t("status.online") : t("status.offline")}</span></div>}
      </section>
      <div className="metrics-grid metrics-grid-three">
        <Metric label={t("devices.detected")} value={inventory.loading || !selectedClientOnline ? "—" : devices.length} icon={FiCpu} detail={t("devices.detectedDetail")} accent="blue" />
        <Metric label={t("devices.driverIssues")} value={inventory.loading || !selectedClientOnline ? "—" : issues.length} icon={FiAlertTriangle} detail={t("devices.driverIssuesDetail")} accent="amber" />
        <Metric label={t("devices.withDriver")} value={inventory.loading || !selectedClientOnline ? "—" : devices.length - issues.length} icon={FiCheck} detail={t("devices.driverStatusHint")} accent="green" />
      </div>
      <section className="panel">
        <div className="panel-header">
          <div><h2>{t("devices.inventory")}</h2><p>{t("devices.inventoryDescription")}</p></div>
          <span className="count-chip">{t("devices.issueCount", { count: issues.length })}</span>
        </div>
        {inventory.loading ? <LoadingRows /> : !selectedClientId ? (
          <EmptyState icon={FiCpu} title={t("devices.noClients")} description={t("devices.noClientsDescription")} />
        ) : !selectedClientOnline ? (
          <EmptyState icon={FiWifi} title={t("devices.clientOffline")} description={t("devices.clientOfflineDescription")} />
        ) : inventory.error ? null : devices.length === 0 ? (
          <EmptyState icon={FiHardDrive} title={t("devices.noDevices")} description={t("devices.noDevicesDescription")} />
        ) : (
          <div className="table-wrap"><table>
            <thead><tr><th>{t("devices.device")}</th><th>{t("devices.class")}</th><th>{t("devices.bus")}</th><th>{t("devices.driver")}</th><th>{t("common.status")}</th><th>{t("devices.diagnostic")}</th></tr></thead>
            <tbody>{devices.map((device, index) => {
              const hasIssue = Boolean(device.driver_issue) || /driver-missing|error|problem/i.test(String(device.status || ""));
              return <tr key={`${device.instance_id || device.name || "device"}-${index}`}>
                <td><div className="device-name"><strong>{device.name || t("common.unknown")}</strong><small>{device.manufacturer || "—"}</small><small>{device.hardware_id || device.instance_id || "—"}</small>{device.instance_id && device.instance_id !== device.hardware_id && <small>{device.instance_id}</small>}</div></td>
                <td>{device.class || "—"}</td><td>{device.bus || "PnP"}</td>
                <td><div className="device-name"><span>{device.driver_name || t("devices.driverUnknown")}</span><small>{device.driver_version ? t("devices.driverVersion", { version: device.driver_version }) : ""}</small></div></td>
                <td><span className={`status ${hasIssue ? "status-offline" : "status-online"}`}><span className="status-dot" />{hasIssue ? t("devices.checkDriver") : (device.status || t("common.unknown"))}</span></td>
                <td className={hasIssue ? "driver-issue-copy" : "text-muted-cell"}>{device.issue || (device.problem_code ? `Code ${device.problem_code}` : "—")}</td>
              </tr>;
            })}</tbody>
          </table></div>
        )}
        <p className="panel-footnote">{t("devices.diagnosticDisclaimer")}</p>
      </section>
    </>
  );
}

function BarChart({ title, subtitle, items, emptyLabel }) {
  const maxValue = Math.max(1, ...items.map((item) => Number(item.value) || 0));
  return (
    <section className="panel chart-panel">
      <div className="panel-header"><div><h2>{title}</h2><p>{subtitle}</p></div><FiBarChart2 className="panel-heading-icon" /></div>
      {items.length === 0 ? <div className="chart-empty">{emptyLabel}</div> : (
        <div className="bar-chart" role="img" aria-label={title}>
          {items.map((item) => <div className="bar-row" key={item.label}>
            <span className="bar-label" title={item.label}>{item.label}</span>
            <div className="bar-track"><span className={`bar-fill ${item.tone || ""}`} style={{ width: `${Math.max(item.value > 0 ? 3 : 0, (Number(item.value) / maxValue) * 100)}%` }} /></div>
            <strong className="bar-value">{item.value}</strong>
          </div>)}
        </div>
      )}
    </section>
  );
}

function Analytics() {
  const { t } = useI18n();
  const clientsResource = useResource((signal) => getClients(signal));
  const clients = Array.isArray(clientsResource.data) ? clientsResource.data : [];
  const [selectedClientId, setSelectedClientId] = useState("");
  useDefaultClientSelection(clients, selectedClientId, setSelectedClientId);
  const selectedClient = clients.find((client) => client.id === selectedClientId);
  const selectedClientOnline = Boolean(selectedClient?.online);
  const statsResource = useResource(
    (signal) => selectedClientId && selectedClientOnline
      ? getClientStats(selectedClientId, signal).then((payload) => ({ clientId: selectedClientId, payload }))
      : Promise.resolve({ clientId: selectedClientId, payload: {} }),
    [selectedClientId, selectedClientOnline]
  );
  // Never show a previous client's CPU/RAM sample while a new selection is loading.
  const statsRecord = statsResource.data;
  const stats = statsRecord && statsRecord.clientId === selectedClientId
    ? normalizeStats(statsRecord.payload)
    : {};
  const online = clients.filter((client) => client.online).length;
  const offline = clients.length - online;
  const osCounts = useMemo(() => {
    const counts = new Map();
    for (const client of clients) {
      const os = String(client.os || t("common.unknown")).trim() || t("common.unknown");
      counts.set(os, (counts.get(os) || 0) + 1);
    }
    return [...counts.entries()].map(([label, value]) => ({ label, value })).sort((a, b) => b.value - a.value);
  }, [clients, t]);
  const cpu = Number(stats.cpu_percent);
  const memoryTotal = Number(stats.mem_total_mb || 0);
  const memoryUsed = Number(stats.mem_used_mb || (memoryTotal - Number(stats.mem_available_mb || 0)) || 0);
  const memoryPercent = memoryTotal > 0 ? Math.min(100, Math.max(0, (memoryUsed / memoryTotal) * 100)) : null;
  const uptime = Number(stats.uptime_seconds);
  const uptimeLabel = Number.isFinite(uptime) && uptime > 0 ? `${Math.floor(uptime / 86400)}d ${Math.floor((uptime % 86400) / 3600)}h` : (stats.uptime_human || "—");

  return (
    <>
      <PageHeading eyebrow={t("eyebrow.overview")} title={t("page.analytics.title")} description={t("page.analytics.description")} actions={<Button icon={FiRefreshCw} onClick={() => { clientsResource.refresh(); statsResource.refresh(); }}>{t("common.refresh")}</Button>} />
      <ResourceError message={clientsResource.error} onRetry={clientsResource.refresh} />
      <div className="metrics-grid">
        <Metric label={t("analytics.registered")} value={clientsResource.loading ? "—" : clients.length} icon={FiMonitor} detail={t("analytics.nodesInRegistry")} accent="blue" />
        <Metric label={t("analytics.onlineRate")} value={clientsResource.loading || !clients.length ? "—" : `${Math.round((online / clients.length) * 100)}%`} icon={FiActivity} detail={t("analytics.liveAvailability")} accent="green" />
        <Metric label={t("common.online")} value={clientsResource.loading ? "—" : online} icon={FiWifi} detail={t("analytics.currentlyConnected")} accent="teal" />
        <Metric label={t("common.offline")} value={clientsResource.loading ? "—" : offline} icon={FiClock} detail={t("analytics.currentlyDisconnected")} accent="amber" />
      </div>
      <div className="content-grid analytics-grid">
        <BarChart title={t("analytics.availabilityChart")} subtitle={t("analytics.currentSnapshot")} emptyLabel={t("analytics.noClientData")} items={[
          { label: t("common.online"), value: online, tone: "bar-positive" },
          { label: t("common.offline"), value: offline, tone: "bar-warning" },
        ]} />
        <BarChart title={t("analytics.osChart")} subtitle={t("analytics.osDistribution")} emptyLabel={t("analytics.noClientData")} items={osCounts} />
      </div>
      <section className="panel agent-stats-panel">
        <div className="panel-header"><div><h2>{t("analytics.agentTelemetry")}</h2><p>{t("analytics.agentTelemetryDescription")}</p></div><FiActivity className="panel-heading-icon" /></div>
        <div className="telemetry-toolbar"><ClientPicker clients={clients} value={selectedClientId} onChange={setSelectedClientId} label={t("devices.client")} /><Button icon={FiRefreshCw} loading={statsResource.loading} disabled={!selectedClientOnline} onClick={statsResource.refresh}>{t("analytics.queryAgent")}</Button></div>
        <ResourceError message={statsResource.error} onRetry={statsResource.refresh} />
        {selectedClientId && !selectedClientOnline && <div className="notice-banner" role="status"><FiWifi /><span>{t("analytics.agentOffline")}</span></div>}
        {stats.error ? <div className="notice-banner">{String(stats.error)}</div> : null}
        <div className="telemetry-grid">
          <Metric label={t("analytics.cpuUsage")} value={Number.isFinite(cpu) ? `${cpu.toFixed(1)}%` : "—"} icon={FiCpu} detail={t("analytics.cpuSnapshot")} accent="blue" />
          <Metric label={t("analytics.memoryUsage")} value={memoryPercent === null ? "—" : `${memoryPercent.toFixed(1)}%`} icon={FiHardDrive} detail={memoryTotal ? `${Math.round(memoryUsed)} / ${Math.round(memoryTotal)} MB` : t("analytics.memoryUnavailable")} accent="teal" />
          <Metric label={t("analytics.uptime")} value={uptimeLabel} icon={FiClock} detail={t("analytics.uptimeDetail")} accent="green" />
        </div>
        {!selectedClientId && <p className="panel-footnote">{t("analytics.selectOnlineClient")}</p>}
        {statsResource.error && <p className="panel-footnote">{t("analytics.onlineAgentRequired")}</p>}
      </section>
      <p className="analytics-note"><FiActivity /> {t("analytics.noHistoricalNote")}</p>
    </>
  );
}

function Automation() {
  const { t } = useI18n();
  const policiesResource = useResource((signal) => getPolicies(signal));
  const schedulesResource = useResource((signal) => getSchedules(signal));
  const filesResource = useResource((signal) => getFiles(signal));
  const policies = Array.isArray(policiesResource.data) ? policiesResource.data : [];
  const schedules = Array.isArray(schedulesResource.data) ? schedulesResource.data : [];
  const files = Array.isArray(filesResource.data) ? filesResource.data : [];
  const [policyName, setPolicyName] = useState("");
  const [policyDescription, setPolicyDescription] = useState("");
  const [osFilter, setOsFilter] = useState("");
  const [mode, setMode] = useState("quiet");
  const [selectedFiles, setSelectedFiles] = useState([]);
  const [scheduleName, setScheduleName] = useState("");
  const [schedulePolicyId, setSchedulePolicyId] = useState("");
  const [intervalMinutes, setIntervalMinutes] = useState("60");
  const [busy, setBusy] = useState(false);
  const [runningPolicyId, setRunningPolicyId] = useState("");
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!schedulePolicyId && policies.length) setSchedulePolicyId(policies[0].id);
    if (schedulePolicyId && !policies.some((policy) => policy.id === schedulePolicyId)) setSchedulePolicyId(policies[0]?.id || "");
  }, [policies, schedulePolicyId]);

  async function submitPolicy(event) {
    event.preventDefault(); setBusy(true); setError(""); setNotice("");
    try {
      await createPolicy({ name: policyName, description: policyDescription, os_filter: osFilter, files: selectedFiles, mode });
      setPolicyName(""); setPolicyDescription(""); setOsFilter(""); setSelectedFiles([]);
      setNotice(t("automation.policyCreated")); policiesResource.refresh();
    } catch (err) { setError(err.message || t("automation.createFailed")); }
    finally { setBusy(false); }
  }

  async function submitSchedule(event) {
    event.preventDefault(); setBusy(true); setError(""); setNotice("");
    try {
      await createSchedule({ name: scheduleName, policy_id: schedulePolicyId, interval_minutes: Number(intervalMinutes), enabled: true });
      setScheduleName(""); setNotice(t("automation.scheduleCreated")); schedulesResource.refresh();
    } catch (err) { setError(err.message || t("automation.createFailed")); }
    finally { setBusy(false); }
  }

  async function runPolicyNow(policy) {
    if (!window.confirm(t("automation.runPolicyConfirm", { name: policy.name }))) return;
    setRunningPolicyId(policy.id); setError(""); setNotice("");
    try {
      const result = await runPolicy(policy.id);
      setNotice(t("automation.policyRunStarted", { name: policy.name, job: result?.job_id || "" }));
    } catch (err) { setError(err.message || t("automation.runFailed")); }
    finally { setRunningPolicyId(""); }
  }

  async function removePolicy(policy) {
    if (!window.confirm(t("automation.deletePolicyConfirm", { name: policy.name }))) return;
    setError("");
    try { await deletePolicy(policy.id); setNotice(t("automation.policyDeleted")); policiesResource.refresh(); schedulesResource.refresh(); }
    catch (err) { setError(err.message || t("automation.deleteFailed")); }
  }

  async function removeSchedule(schedule) {
    if (!window.confirm(t("automation.deleteScheduleConfirm", { name: schedule.name }))) return;
    setError("");
    try { await deleteSchedule(schedule.id); setNotice(t("automation.scheduleDeleted")); schedulesResource.refresh(); }
    catch (err) { setError(err.message || t("automation.deleteFailed")); }
  }

  async function toggleSchedule(schedule) {
    setError("");
    try { await setScheduleEnabled(schedule.id, !schedule.enabled); schedulesResource.refresh(); }
    catch (err) { setError(err.message || t("automation.updateFailed")); }
  }

  return (
    <>
      <PageHeading eyebrow={t("eyebrow.deployment")} title={t("page.automation.title")} description={t("page.automation.description")} actions={<Button icon={FiRefreshCw} onClick={() => { policiesResource.refresh(); schedulesResource.refresh(); filesResource.refresh(); }}>{t("common.refresh")}</Button>} />
      {error && <div className="error-banner" role="alert"><FiAlertTriangle /><div><strong>{t("common.error")}</strong><p>{error}</p></div><button className="icon-button" onClick={() => setError("")} aria-label={t("common.close")}><FiX /></button></div>}
      {notice && <div className="notice-banner" role="status"><span>{notice}</span><button onClick={() => setNotice("")} aria-label={t("common.close")}><FiX /></button></div>}
      <div className="content-grid automation-grid">
        <section className="panel">
          <div className="panel-header"><div><h2>{t("automation.newPolicy")}</h2><p>{t("automation.newPolicyDescription")}</p></div><FiLayers className="panel-heading-icon" /></div>
          <form className="automation-form" onSubmit={submitPolicy}>
            <label className="field-label">{t("automation.policyName")}<input className="text-input" value={policyName} onChange={(e) => setPolicyName(e.target.value)} maxLength={120} required /></label>
            <label className="field-label">{t("automation.descriptionLabel")}<textarea className="text-input" rows={2} value={policyDescription} onChange={(e) => setPolicyDescription(e.target.value)} maxLength={500} /></label>
            <label className="field-label">{t("automation.osFilter")}<input className="text-input" value={osFilter} onChange={(e) => setOsFilter(e.target.value)} maxLength={100} placeholder={t("automation.osFilterPlaceholder")} /><small className="field-help">{t("automation.osFilterHelp")}</small></label>
            <label className="field-label">{t("install.executionMode")}<select className="text-input" value={mode} onChange={(e) => setMode(e.target.value)}><option value="quiet">{t("install.silent")}</option><option value="normal">{t("install.normalMode")}</option><option value="interactive">{t("install.interactive")}</option></select></label>
            <div className="field-label">{t("automation.packages")}<div className="file-choice-list">{files.map((file) => <label className="file-choice" key={file.id}><input type="checkbox" checked={selectedFiles.includes(file.id)} onChange={() => setSelectedFiles((current) => current.includes(file.id) ? current.filter((id) => id !== file.id) : [...current, file.id])} /><span><strong>{file.name}</strong><small>{formatBytes(file.size)}</small></span></label>)}{files.length === 0 && <p className="field-help">{t("automation.uploadPackageFirst")}</p>}</div></div>
            <Button type="submit" variant="primary" icon={FiPlus} loading={busy} disabled={!files.length || !selectedFiles.length}>{t("automation.createPolicy")}</Button>
          </form>
        </section>
        <section className="panel">
          <div className="panel-header"><div><h2>{t("automation.newSchedule")}</h2><p>{t("automation.newScheduleDescription")}</p></div><FiCalendar className="panel-heading-icon" /></div>
          <form className="automation-form" onSubmit={submitSchedule}>
            <label className="field-label">{t("automation.scheduleName")}<input className="text-input" value={scheduleName} onChange={(e) => setScheduleName(e.target.value)} maxLength={120} required /></label>
            <label className="field-label">{t("automation.policyLabel")}<select className="text-input" value={schedulePolicyId} onChange={(e) => setSchedulePolicyId(e.target.value)} required disabled={!policies.length}><option value="">{t("automation.choosePolicy")}</option>{policies.map((policy) => <option key={policy.id} value={policy.id}>{policy.name}</option>)}</select></label>
            <label className="field-label">{t("automation.intervalMinutes")}<input className="text-input" type="number" min="5" max="43200" step="1" value={intervalMinutes} onChange={(e) => setIntervalMinutes(e.target.value)} required /><small className="field-help">{t("automation.intervalHelp")}</small></label>
            <div className="schedule-preview"><FiClock /><span>{t("automation.schedulePreview", { interval: intervalMinutes })}</span></div>
            <Button type="submit" variant="primary" icon={FiPlus} loading={busy} disabled={!policies.length || Number(intervalMinutes) < 5 || Number(intervalMinutes) > 43200 || !scheduleName.trim()}>{t("automation.createSchedule")}</Button>
          </form>
        </section>
      </div>
      <section className="panel automation-list-panel">
        <div className="panel-header"><div><h2>{t("automation.policiesTitle")}</h2><p>{t("automation.policiesDescription")}</p></div><span className="count-chip">{policies.length}</span></div>
        {policiesResource.loading ? <LoadingRows /> : policies.length === 0 ? <EmptyState icon={FiLayers} title={t("automation.noPolicies")} description={t("automation.noPoliciesDescription")} /> : <div className="automation-list">{policies.map((policy) => <article className="automation-card" key={policy.id}><div className="automation-card-icon"><FiLayers /></div><div className="automation-card-copy"><div className="automation-title-row"><strong>{policy.name}</strong><span className={`state-chip ${policy.enabled ? "state-enabled" : "state-disabled"}`}>{policy.enabled ? t("common.enabled") : t("common.disabled")}</span></div><p>{policy.description || t("automation.noDescription")}</p><small>{t("automation.policyMetadata", { count: (policy.files || []).length, os: policy.os_filter || t("automation.allOperatingSystems"), mode: policy.mode })}</small></div><div className="automation-card-actions"><Button variant="primary" icon={FiPlay} loading={runningPolicyId === policy.id} onClick={() => runPolicyNow(policy)}>{t("automation.runNow")}</Button><Button variant="secondary" icon={FiTrash2} onClick={() => removePolicy(policy)}>{t("common.delete")}</Button></div></article>)}</div>}
      </section>
      <section className="panel automation-list-panel">
        <div className="panel-header"><div><h2>{t("automation.schedulesTitle")}</h2><p>{t("automation.schedulesDescription")}</p></div><span className="count-chip">{schedules.length}</span></div>
        {schedulesResource.loading ? <LoadingRows /> : schedules.length === 0 ? <EmptyState icon={FiCalendar} title={t("automation.noSchedules")} description={t("automation.noSchedulesDescription")} /> : <div className="automation-list">{schedules.map((schedule) => { const policy = policies.find((item) => item.id === schedule.policy_id); return <article className="automation-card" key={schedule.id}><div className="automation-card-icon"><FiCalendar /></div><div className="automation-card-copy"><div className="automation-title-row"><strong>{schedule.name}</strong><span className={`state-chip ${schedule.enabled ? "state-enabled" : "state-disabled"}`}>{schedule.enabled ? t("common.enabled") : t("common.disabled")}</span></div><p>{policy?.name || t("automation.policyMissing")} · {t("automation.everyMinutes", { count: schedule.interval_minutes })}</p><small>{t("automation.nextRun", { date: formatDate(schedule.next_run) })}{schedule.last_run ? ` · ${t("automation.lastRun", { date: formatDate(schedule.last_run) })}` : ""}</small></div><div className="automation-card-actions"><Button variant="secondary" onClick={() => toggleSchedule(schedule)}>{schedule.enabled ? t("common.disable") : t("common.enable")}</Button><Button variant="secondary" icon={FiTrash2} onClick={() => removeSchedule(schedule)}>{t("common.delete")}</Button></div></article>; })}</div>}
      </section>
      <p className="analytics-note"><FiShield /> {t("automation.safetyNote")}</p>
    </>
  );
}

function NotFound() {
  const { t } = useI18n();
  return (
    <EmptyState
      icon={FiSearch}
      title={t("notFound.title")}
      description={t("notFound.description")}
      action={<NavLink className="button button-primary" to="/">{t("common.home")}</NavLink>}
    />
  );
}

function App() {
  const { theme, toggleTheme } = useTheme();

  return (
    <BrowserRouter>
      <AppShell theme={theme} toggleTheme={toggleTheme}>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/clients" element={<Clients />} />
          <Route path="/files" element={<Files />} />
          <Route path="/install" element={<Install />} />
          <Route path="/devices" element={<Devices />} />
          <Route path="/analytics" element={<Analytics />} />
          <Route path="/automation" element={<Automation />} />
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}

export default App;