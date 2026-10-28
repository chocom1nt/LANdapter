export const DEFAULT_API_BASE = "/api/v1";

export function getApiToken() {
  return sessionStorage.getItem("api_token") || "";
}

export function validateApiBase(value) {
  const raw = String(value ?? "").trim();
  if (!raw || raw.includes("\\") || raw.includes("?") || raw.includes("#")) {
    throw new Error("Укажите корректный URL API без query-параметров и фрагмента.");
  }

  if (raw.startsWith("/") && !raw.startsWith("//")) {
    const segments = raw.split("/");
    if (segments.some((segment) => segment === "." || segment === "..")) {
      throw new Error("URL API не должен содержать сегменты . или ..");
    }
    const normalized = raw.replace(/\/+$/, "") || "/";
    if (!normalized.endsWith("/api/v1")) {
      throw new Error("Адрес API должен оканчиваться на /api/v1.");
    }
    return normalized;
  }

  let parsed;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error("Введите относительный путь или полный URL API.");
  }
  if (!["http:", "https:"].includes(parsed.protocol)) {
    throw new Error("Разрешены только схемы HTTP и HTTPS.");
  }
  if (parsed.username || parsed.password || parsed.search || parsed.hash) {
    throw new Error("Не помещайте учётные данные в URL API.");
  }
  if (parsed.pathname.split("/").some((part) => part === "." || part === "..")) {
    throw new Error("URL API не должен содержать сегменты . или ..");
  }
  if (!parsed.pathname.replace(/\/+$/, "").endsWith("/api/v1")) {
    throw new Error("Адрес API должен оканчиваться на /api/v1.");
  }
  const localHosts = new Set(["localhost", "127.0.0.1", "[::1]"]);
  if (parsed.protocol === "http:" && !localHosts.has(parsed.hostname.toLowerCase())) {
    throw new Error("Для удалённого Master используйте HTTPS.");
  }
  return `${parsed.origin}${parsed.pathname.replace(/\/+$/, "")}`;
}

export function getApiBase() {
  const configured = localStorage.getItem("api_url")?.trim();
  if (!configured) return DEFAULT_API_BASE;
  try {
    return validateApiBase(configured);
  } catch {
    return DEFAULT_API_BASE;
  }
}

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  if (options.body && !isFormData && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const token = getApiToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const response = await fetch(`${getApiBase()}${path}`, {
    ...options,
    headers,
    redirect: "error",
  });

  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(
      message.trim() || `HTTP ${response.status}: ${response.statusText}`
    );
  }

  if (response.status === 204) return null;

  const contentType = response.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    return response.json();
  }

  return response.text();
}

export const getClients = (signal) => request("/clients", { signal });
export const getFiles = (signal) => request("/files", { signal });

export const getClientDevices = (clientId, signal) =>
  request(`/clients/${encodeURIComponent(clientId)}/devices`, { signal });

export const getClientStats = (clientId, signal) =>
  request(`/clients/${encodeURIComponent(clientId)}/stats`, { signal });

export function uploadFile(file, onProgress) {
  // fetch does not expose upload progress; keep this argument for caller compatibility.
  void onProgress;
  const form = new FormData();
  form.append("file", file);
  return request("/upload", { method: "POST", body: form });
}

export function deleteFile(fileId) {
  return request(`/files/${encodeURIComponent(fileId)}`, { method: "DELETE" });
}

export function install(fileIds, clientIds, mode = "quiet") {
  if (!fileIds.length) throw new Error("Выберите хотя бы один файл");
  if (!clientIds.length) throw new Error("Выберите хотя бы один компьютер");
  return request("/install", {
    method: "POST",
    body: JSON.stringify({ file_ids: fileIds, client_ids: clientIds, mode }),
  });
}

export function wakeOnLan(clientIds) {
  if (!clientIds.length) throw new Error("Выберите хотя бы один компьютер");
  return request("/wol", {
    method: "POST",
    body: JSON.stringify({ client_ids: clientIds }),
  });
}

export function parseDriver(url) {
  return request("/parse-driver", {
    method: "POST",
    body: JSON.stringify({ url }),
  });
}

export function enrollAgent(agentId = "") {
  const body = agentId.trim() ? { agent_id: agentId.trim() } : {};
  return request("/agents/enroll", {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export const getPolicies = (signal) => request("/policies", { signal });

export function createPolicy(policy) {
  return request("/policies", {
    method: "POST",
    body: JSON.stringify(policy),
  });
}

export function deletePolicy(policyId) {
  return request(`/policies/${encodeURIComponent(policyId)}`, { method: "DELETE" });
}

export function runPolicy(policyId) {
  return request(`/policies/${encodeURIComponent(policyId)}/run`, { method: "POST" });
}

export const getSchedules = (signal) => request("/schedules", { signal });

export function createSchedule(schedule) {
  return request("/schedules", {
    method: "POST",
    body: JSON.stringify(schedule),
  });
}

export function setScheduleEnabled(scheduleId, enabled) {
  return request(`/schedules/${encodeURIComponent(scheduleId)}`, {
    method: "PATCH",
    body: JSON.stringify({ enabled: Boolean(enabled) }),
  });
}

export function deleteSchedule(scheduleId) {
  return request(`/schedules/${encodeURIComponent(scheduleId)}`, { method: "DELETE" });
}
