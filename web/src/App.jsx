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
  getFiles,
  install,
  uploadFile,
  wakeOnLan,
} from "./api";
import "./index.css";

const NAVIGATION = [
  { to: "/", label: "Обзор", icon: FiLayout, end: true },
  { to: "/clients", label: "Компьютеры", icon: FiMonitor },
  { to: "/files", label: "Библиотека файлов", icon: FiFileText },
  { to: "/install", label: "Установка", icon: FiPackage },
  { to: "/settings", label: "Настройки", icon: FiSettings },
];

const TITLES = {
  "/": ["Панель управления", "Общее состояние инфраструктуры"],
  "/clients": ["Компьютеры", "Подключённые агенты и состояние узлов"],
  "/files": ["Библиотека файлов", "Пакеты, драйверы и установщики"],
  "/install": ["Развёртывание", "Запуск установки на выбранных узлах"],
  "/settings": ["Настройки", "Подключение и параметры интерфейса"],
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
          setError(err.message || "Не удалось загрузить данные");
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

  return new Intl.DateTimeFormat("ru-RU", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

function formatBytes(bytes = 0) {
  if (!Number.isFinite(Number(bytes)) || Number(bytes) < 0) return "—";

  const value = Number(bytes);
  if (value < 1024) return `${value} Б`;

  const units = ["КБ", "МБ", "ГБ", "ТБ"];
  let size = value / 1024;
  let unit = 0;

  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit++;
  }

  return `${size.toLocaleString("ru-RU", {
    maximumFractionDigits: 1,
  })} ${units[unit]}`;
}

function Status({ online }) {
  return (
    <span className={`status ${online ? "status-online" : "status-offline"}`}>
      <span className="status-dot" />
      {online ? "В сети" : "Не в сети"}
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
  const [mobileNav, setMobileNav] = useState(false);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const timer = setInterval(() => setNow(new Date()), 30_000);
    return () => clearInterval(timer);
  }, []);

  useEffect(() => setMobileNav(false), [location.pathname]);

  const [title, subtitle] = TITLES[location.pathname] || [
    "LANdapter",
    "Управление инфраструктурой",
  ];

  return (
    <div className="app-shell">
      {mobileNav && (
        <button
          className="mobile-overlay"
          aria-label="Закрыть меню"
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

        <div className="nav-label">РАБОЧЕЕ ПРОСТРАНСТВО</div>
        <nav className="nav-list" aria-label="Основная навигация">
          {NAVIGATION.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to}
              to={to}
              end={end}
              className={({ isActive }) =>
                `nav-link ${isActive ? "nav-link-active" : ""}`
              }
            >
              <Icon size={18} />
              <span>{label}</span>
              <FiChevronRight className="nav-chevron" />
            </NavLink>
          ))}
        </nav>

        <div className="sidebar-bottom">
          <div className="sidebar-status">
            <span className="status-dot status-dot-live" />
            <div>
              <strong>Локальная консоль</strong>
              <small>API {getApiBase()}</small>
            </div>
          </div>
          <button className="theme-button" onClick={toggleTheme}>
            {theme === "dark" ? <FiSun /> : <FiMoon />}
            <span>{theme === "dark" ? "Светлая тема" : "Тёмная тема"}</span>
          </button>
        </div>
      </aside>

      <main className="main-panel">
        <header className="topbar">
          <button
            className="mobile-menu"
            onClick={() => setMobileNav(true)}
            aria-label="Открыть меню"
          >
            <FiLayout />
          </button>
          <div className="breadcrumbs">
            <span>LANdapter</span>
            <FiChevronRight />
            <strong>{title}</strong>
          </div>
          <div className="topbar-right">
            <span className="topbar-time">
              {now.toLocaleTimeString("ru-RU", {
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
            <span>Локальное управление сетью</span>
          </footer>
        </div>
      </main>
    </div>
  );
}

function ResourceError({ message, onRetry }) {
  if (!message) return null;

  return (
    <div className="error-banner" role="alert">
      <FiShield size={18} />
      <div>
        <strong>Не удалось получить данные</strong>
        <p>{message}</p>
      </div>
      <Button icon={FiRefreshCw} onClick={onRetry}>Повторить</Button>
    </div>
  );
}

function Dashboard() {
  const clients = useResource((signal) => getClients(signal));
  const files = useResource((signal) => getFiles(signal));

  const clientList = Array.isArray(clients.data) ? clients.data : [];
  const fileList = Array.isArray(files.data) ? files.data : [];
  const online = clientList.filter((client) => client.online).length;
  const offline = clientList.length - online;

  return (
    <>
      <PageHeading
        eyebrow="OVERVIEW / 01"
        title="Панель управления"
        description="Состояние агентов, доступность инфраструктуры и пакеты развёртывания."
        actions={
          <Button
            icon={FiRefreshCw}
            onClick={() => {
              clients.refresh();
              files.refresh();
            }}
          >
            Обновить данные
          </Button>
        }
      />

      <ResourceError message={clients.error} onRetry={clients.refresh} />
      <ResourceError message={files.error} onRetry={files.refresh} />

      <div className="metrics-grid">
        <Metric
          label="Всего компьютеров"
          value={clients.loading ? "—" : clientList.length}
          icon={FiMonitor}
          detail="Зарегистрированные агенты"
          accent="teal"
        />
        <Metric
          label="В сети"
          value={clients.loading ? "—" : online}
          icon={FiWifi}
          detail="Доступные узлы"
          accent="green"
        />
        <Metric
          label="Не в сети"
          value={clients.loading ? "—" : offline}
          icon={FiActivity}
          detail="Нет активного соединения"
          accent="amber"
        />
        <Metric
          label="Файлы"
          value={files.loading ? "—" : fileList.length}
          icon={FiHardDrive}
          detail="В библиотеке развёртывания"
          accent="blue"
        />
      </div>

      <div className="content-grid">
        <section className="panel panel-main">
          <div className="panel-header">
            <div>
              <h2>Компьютеры</h2>
              <p>Последние зарегистрированные узлы</p>
            </div>
            <NavLink className="text-link" to="/clients">
              Все компьютеры <FiChevronRight />
            </NavLink>
          </div>

          {clients.loading ? (
            <LoadingRows />
          ) : clientList.length === 0 && !clients.error ? (
            <EmptyState
              icon={FiMonitor}
              title="Агентов пока нет"
              description="Подключите агент к Master-серверу, чтобы увидеть его здесь."
            />
          ) : (
            <ClientTable clients={clientList.slice(0, 6)} />
          )}
        </section>

        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>Состояние системы</h2>
              <p>Сводка доступных ресурсов</p>
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
              label="Агенты"
              description="Регистрация и соединения"
              status={clients.error ? "error" : clients.loading ? "loading" : "ok"}
            />
            <HealthItem
              label="Библиотека"
              description="Хранилище пакетов"
              status={files.error ? "error" : files.loading ? "loading" : "ok"}
            />
          </div>

          <div className="callout">
            <FiShield size={20} />
            <div>
              <strong>Проверьте безопасность</strong>
              <p>
                Перед публикацией ограничьте доступ к административному API
                и WebSocket.
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
  const text = {
    ok: "Доступно",
    error: "Ошибка",
    loading: "Проверка",
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
  return (
    <div className="loading-rows" aria-label="Загрузка">
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
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {onToggle && <th aria-label="Выбрать" />}
            <th>Компьютер</th>
            <th>Операционная система</th>
            <th>MAC-адрес</th>
            <th>Статус</th>
            <th>Последняя активность</th>
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
                    aria-label={`Выбрать ${client.hostname}`}
                  />
                </td>
              )}
              <td>
                <div className="client-name">
                  <span className="client-avatar"><FiMonitor /></span>
                  <div>
                    <strong>{client.hostname || "Без имени"}</strong>
                    <small title={client.id}>{String(client.id).slice(0, 12)}</small>
                  </div>
                </div>
              </td>
              <td>{client.os || "Неизвестно"}</td>
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
  const resource = useResource((signal) => getClients(signal));
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState("all");
  const [selected, setSelected] = useState([]);
  const [busy, setBusy] = useState(false);
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
      setNotice("Запрос Wake-on-LAN отправлен.");
    } catch (error) {
      setNotice(`Не удалось отправить Wake-on-LAN: ${error.message}`);
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeading
        eyebrow="INFRASTRUCTURE / 02"
        title="Компьютеры"
        description="Управление агентами, сетевыми узлами и их доступностью."
        actions={
          <Button icon={FiRefreshCw} onClick={resource.refresh}>
            Обновить
          </Button>
        }
      />

      <ResourceError message={resource.error} onRetry={resource.refresh} />
      {notice && (
        <div className="notice-banner" role="status">
          <span>{notice}</span>
          <button onClick={() => setNotice("")} aria-label="Закрыть">
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
            placeholder="Имя, ОС, MAC или ID..."
          />
          {search && (
            <button onClick={() => setSearch("")} aria-label="Очистить поиск">
              <FiX />
            </button>
          )}
        </label>

        <select value={filter} onChange={(event) => setFilter(event.target.value)}>
          <option value="all">Все узлы</option>
          <option value="online">В сети</option>
          <option value="offline">Не в сети</option>
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
            <h2>Реестр узлов</h2>
            <p>
              {resource.loading
                ? "Загрузка данных..."
                : `Показано ${filtered.length} из ${clients.length}`}
            </p>
          </div>
          <span className="count-chip">{clients.length} узл.</span>
        </div>

        {resource.loading ? (
          <LoadingRows />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={FiMonitor}
            title={clients.length ? "Ничего не найдено" : "Нет зарегистрированных агентов"}
            description={
              clients.length
                ? "Измените поисковый запрос или фильтр."
                : "После подключения агенты появятся в этом списке."
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
      setMessage(`Файл «${file.name}» загружен.`);
      resource.refresh();
    } catch (err) {
      setError(err.message || "Ошибка загрузки");
    } finally {
      setUploading(false);
      input.value = "";
    }
  }

  async function handleDelete(file) {
    if (!window.confirm(`Удалить файл «${file.name}»?`)) return;

    setError("");
    setMessage("");

    try {
      await deleteFile(file.id);
      resource.setData((current) =>
        current.filter((item) => item.id !== file.id)
      );
      setMessage("Файл удалён.");
    } catch (err) {
      setError(err.message || "Не удалось удалить файл");
    }
  }

  return (
    <>
      <PageHeading
        eyebrow="ARTIFACTS / 03"
        title="Библиотека файлов"
        description="Единое хранилище установщиков и пакетов для развёртывания."
        actions={
          <label className={`button button-primary upload-button ${uploading ? "disabled" : ""}`}>
            {uploading ? <FiRefreshCw className="spin" /> : <FiUploadCloud />}
            {uploading ? "Загрузка..." : "Загрузить файл"}
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
          <button onClick={() => setMessage("")} aria-label="Закрыть">
            <FiX />
          </button>
        </div>
      )}

      <div className="file-summary-grid">
        <div className="file-summary">
          <FiHardDrive />
          <div><span>Всего файлов</span><strong>{files.length}</strong></div>
        </div>
        <div className="file-summary">
          <FiPackage />
          <div><span>Общий размер</span><strong>{formatBytes(
            files.reduce((total, file) => total + Number(file.size || 0), 0)
          )}</strong></div>
        </div>
        <div className="file-summary">
          <FiShield />
          <div><span>Хранилище</span><strong>Master</strong></div>
        </div>
      </div>

      <section className="panel">
        <div className="panel-header file-panel-header">
          <div>
            <h2>Файлы и пакеты</h2>
            <p>Загруженные артефакты</p>
          </div>
          <label className="search-box compact-search">
            <FiSearch />
            <input
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Поиск файлов..."
            />
          </label>
        </div>

        {resource.loading ? (
          <LoadingRows />
        ) : filtered.length === 0 ? (
          <EmptyState
            icon={FiFile}
            title={files.length ? "Файлы не найдены" : "Библиотека пока пуста"}
            description={
              files.length
                ? "Попробуйте изменить поисковый запрос."
                : "Загрузите установщик или драйвер, чтобы использовать его в заданиях."
            }
            action={
              <label className="button button-secondary upload-button">
                <FiPlus /> Добавить файл
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
                  title="Удалить файл"
                  aria-label={`Удалить ${file.name}`}
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
      setError(err.message || "Не удалось запустить установку");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <PageHeading
        eyebrow="DEPLOYMENT / 04"
        title="Развёртывание"
        description="Создание задания установки на одном или нескольких компьютерах."
      />

      <ResourceError message={clientsResource.error} onRetry={clientsResource.refresh} />
      <ResourceError message={filesResource.error} onRetry={filesResource.refresh} />
      {error && <div className="error-banner" role="alert">{error}</div>}
      {jobId && (
        <div className="notice-banner" role="status">
          <FiCheck />
          <span>Задание создано. ID: <code>{jobId}</code></span>
        </div>
      )}

      <form className="deploy-layout" onSubmit={submit}>
        <div className="deploy-main">
          <section className="panel">
            <div className="step-heading">
              <span className="step-number">01</span>
              <div><h2>Выберите пакеты</h2><p>Что требуется установить</p></div>
            </div>

            {files.length === 0 ? (
              <EmptyState
                icon={FiPackage}
                title="Нет доступных пакетов"
                description="Сначала загрузите файл в библиотеку."
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
              <div><h2>Целевые компьютеры</h2><p>Укажите узлы для выполнения задания</p></div>
            </div>

            {clients.length === 0 ? (
              <EmptyState
                icon={FiMonitor}
                title="Нет зарегистрированных компьютеров"
                description="Подключите агенты перед запуском установки."
              />
            ) : (
              <>
                <div className="selection-toolbar">
                  <span>{clientIds.length} из {clients.length} выбрано</span>
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
                    {clientIds.length === clients.length ? "Снять выбор" : "Выбрать все"}
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
                        <small>{client.os || "ОС не определена"}</small>
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
              <div><h2>Параметры</h2><p>Режим выполнения</p></div>
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
              <span><strong>Тихая установка</strong><small>Без интерактивного интерфейса</small></span>
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
              <span><strong>Интерактивная</strong><small>С пользовательским интерфейсом</small></span>
            </label>

            <div className="deploy-summary">
              <div><span>Пакеты</span><strong>{fileIds.length}</strong></div>
              <div><span>Компьютеры</span><strong>{clientIds.length}</strong></div>
              <div><span>Режим</span><strong>{mode === "quiet" ? "Тихий" : "Интерактивный"}</strong></div>
            </div>

            <Button
              type="submit"
              variant="primary"
              icon={FiDownload}
              loading={busy}
              disabled={!fileIds.length || !clientIds.length}
            >
              Создать задание
            </Button>

            <p className="form-hint">
              Перед запуском убедитесь, что выбранные пакеты подходят целевым ОС.
            </p>
          </section>
        </aside>
      </form>
    </>
  );
}

function Settings() {
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
      setNotice("Настройки подключения сохранены.");
    } catch (err) {
      setError(err.message || "Некорректные настройки.");
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
          throw new Error("Неверный или отсутствующий административный токен.");
        }

        throw new Error(responseText || `HTTP ${response.status}`);
      }

      setNotice("Соединение с API установлено.");
    } catch (err) {
      setError(`Проверка соединения не пройдена: ${err.message}`);
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
        throw new Error("Сначала укажите административный API-токен.");
      }

      // enrollAgent() использует сохранённые настройки API и token.
      localStorage.setItem("api_url", normalizedUrl);
      sessionStorage.setItem("api_token", token);
      setApiUrl(normalizedUrl);

      const credential = await enrollAgent(agentId);
      setIssuedCredential(credential);
      setNotice(
        "Учётные данные выданы. Сохраните token: повторно сервер его не покажет."
      );
    } catch (err) {
      setError(`Не удалось зарегистрировать агента: ${err.message}`);
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
      setNotice("Учётные данные скопированы. Храните token как секрет.");
    } catch {
      setError("Не удалось скопировать данные. Скопируйте их вручную.");
    }
  }

  return (
    <>
      <PageHeading
        eyebrow="PREFERENCES / 05"
        title="Настройки"
        description="Безопасное подключение к Master и регистрация агентов."
      />

      {error && (
        <div className="error-banner" role="alert">
          <FiShield />
          <div>
            <strong>Ошибка</strong>
            <p>{error}</p>
          </div>
        </div>
      )}

      {notice && (
        <div className="notice-banner" role="status">
          <span>{notice}</span>
          <button onClick={() => setNotice("")} aria-label="Закрыть">
            <FiX />
          </button>
        </div>
      )}

      <div className="settings-layout">
        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>Подключение к Master</h2>
              <p>Базовый URL REST API и административный секрет</p>
            </div>
            <FiServer className="panel-heading-icon" />
          </div>

          <form onSubmit={saveConnection} className="settings-form">
            <label className="field-label" htmlFor="api-url">
              API base URL
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
              Административный API-токен
            </label>
            <input
              id="admin-api-token"
              className="text-input"
              type="password"
              autoComplete="current-password"
              value={apiToken}
              onChange={(event) => setApiToken(event.target.value)}
              placeholder="Вставьте LANDAPTER_ADMIN_TOKEN"
            />

            <p className="field-help">
              Токен хранится в sessionStorage и удаляется при завершении
              браузерного сеанса. Для удалённого сервера нужен HTTPS.
            </p>

            <div className="form-actions">
              <Button type="submit" variant="primary" icon={FiCheck}>
                Сохранить
              </Button>
              <Button
                type="button"
                icon={FiActivity}
                loading={testing}
                onClick={testConnection}
              >
                Проверить соединение
              </Button>
            </div>
          </form>
        </section>

        <section className="panel">
          <div className="panel-header">
            <div>
              <h2>Регистрация агента</h2>
              <p>Выдача персонального токена для одного узла</p>
            </div>
            <FiShield className="panel-heading-icon" />
          </div>

          <div className="settings-form">
            <label className="field-label" htmlFor="agent-id">
              Существующий UUID (необязательно)
            </label>
            <input
              id="agent-id"
              className="text-input"
              value={agentId}
              onChange={(event) => setAgentId(event.target.value)}
              placeholder="Пусто — создать новый ID"
            />

            <p className="field-help">
              Если UUID указан, токен существующего агента будет заменён,
              а старый перестанет работать. Не выполняйте ротацию, пока
              не готовы обновить секрет на машине агента.
            </p>

            <div className="form-actions">
              <Button
                type="button"
                variant="primary"
                icon={FiPlus}
                loading={enrolling}
                onClick={issueAgentCredential}
              >
                Выпустить токен агента
              </Button>
            </div>

            {issuedCredential && (
              <div style={{ marginTop: 18 }}>
                <label className="field-label">
                  Новые учётные данные — сохраните сейчас
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
                    Копировать JSON
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

function NotFound() {
  return (
    <EmptyState
      icon={FiSearch}
      title="Страница не найдена"
      description="Такого раздела нет. Вернитесь к панели управления."
      action={<NavLink className="button button-primary" to="/">На главную</NavLink>}
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
          <Route path="/settings" element={<Settings />} />
          <Route path="*" element={<NotFound />} />
        </Routes>
      </AppShell>
    </BrowserRouter>
  );
}

export default App;