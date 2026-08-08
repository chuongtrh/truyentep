"use strict";

const ui = {
  selfStatus: document.querySelector("#self-status"),
  peerList: document.querySelector("#peer-list"),
  emptyPeers: document.querySelector("#empty-peers"),
  selectedLabel: document.querySelector("#selected-label"),
  actionHint: document.querySelector("#action-hint"),
  clipboardButton: document.querySelector("#clipboard-button"),
  dropZone: document.querySelector("#drop-zone"),
  fileInput: document.querySelector("#file-input"),
  progressBox: document.querySelector("#progress-box"),
  progressTitle: document.querySelector("#progress-title"),
  progressPercent: document.querySelector("#progress-percent"),
  progressBar: document.querySelector("#progress-bar"),
  progressDetail: document.querySelector("#progress-detail"),
  eventList: document.querySelector("#event-list"),
  historySection: document.querySelector("#history-section"),
  manualForm: document.querySelector("#manual-form"),
  manualAddress: document.querySelector("#manual-address"),
  ownAddress: document.querySelector("#own-address"),
  refreshButton: document.querySelector("#refresh-button"),
  appVersion: document.querySelector("#app-version"),
  downloadsButton: document.querySelector("#downloads-button"),
  quitButton: document.querySelector("#quit-button"),
  quitDialog: document.querySelector("#quit-dialog"),
  confirmQuit: document.querySelector("#confirm-quit"),
  toast: document.querySelector("#toast"),
};

let state = null;
let selectedPeerId = localStorage.getItem("truyen-tep-selected-peer") || "";
let sending = false;
let toastTimer = null;
let pollTimer = null;
let stateLoadsInFlight = 0;
let stateLoadSequence = 0;
let activeStateLoadController = null;

const SVG_NAMESPACE = "http://www.w3.org/2000/svg";
const STATE_LOAD_TIMEOUT_MS = 8000;

async function api(path, options = {}) {
  const method = options.method || "GET";
  const headers = new Headers(options.headers || {});
  if (method !== "GET" && method !== "HEAD") headers.set("X-Truyen-Tep-Local", "1");
  if (typeof options.body === "string" && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(path, { ...options, method, headers });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(payload.error || "Không thực hiện được yêu cầu");
  return payload;
}

async function loadState(showFailure = false) {
  const loadId = ++stateLoadSequence;
  activeStateLoadController?.abort();
  const controller = new AbortController();
  activeStateLoadController = controller;
  stateLoadsInFlight += 1;
  setRefreshLoading(true);
  let timedOut = false;
  const timeoutTimer = window.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, STATE_LOAD_TIMEOUT_MS);
  try {
    const nextState = await api("/api/state", { signal: controller.signal });
    if (loadId !== stateLoadSequence) return;
    state = nextState;
    render();
  } catch (error) {
    if (loadId !== stateLoadSequence) return;
    const message = timedOut ? "Không thể cập nhật trạng thái. Vui lòng thử lại." : error.message;
    if (showFailure) showToast(message, true);
    ui.selfStatus.innerHTML = '<span class="status-dot offline"></span>Mất kết nối với ứng dụng';
  } finally {
    window.clearTimeout(timeoutTimer);
    if (activeStateLoadController === controller) activeStateLoadController = null;
    stateLoadsInFlight = Math.max(0, stateLoadsInFlight - 1);
    setRefreshLoading(stateLoadsInFlight > 0);
  }
}

function setRefreshLoading(loading) {
  ui.refreshButton.classList.toggle("loading", loading);
  ui.refreshButton.setAttribute("aria-busy", loading ? "true" : "false");
}

function formatAppVersion(value) {
  if (typeof value !== "string") return "dev";
  const version = value.trim();
  return version || "dev";
}

function render() {
  if (!state) return;
  ui.selfStatus.replaceChildren();
  const dot = document.createElement("span");
  dot.className = "status-dot";
  ui.selfStatus.append(dot, document.createTextNode(`Đang chạy trên ${state.self.name}`));
  ui.appVersion.textContent = `Phiên bản ${formatAppVersion(state.version)}`;

  const peers = state.peers || [];
  if (!peers.some((peer) => peer.id === selectedPeerId)) {
    selectedPeerId = peers.length === 1 ? peers[0].id : "";
    saveSelectedPeer();
  }
  renderPeers(peers);
  renderActions(peers);
  renderEvents(state.events || []);
  renderOwnAddress();
}

function renderPeers(peers) {
  ui.peerList.replaceChildren();
  ui.emptyPeers.hidden = peers.length > 0;
  if (peers.length === 0) {
    const strong = ui.emptyPeers.querySelector("strong");
    const paragraph = ui.emptyPeers.querySelector("p");
    if (state.discoveryError) {
      strong.textContent = "Chưa tìm thấy máy nhận";
      paragraph.textContent = "Hãy thêm bằng địa chỉ IP ở bên dưới.";
    } else {
      strong.textContent = "Đang tìm máy cùng Wi‑Fi…";
      paragraph.textContent = "Hãy mở Truyền Tệp trên máy Mac còn lại.";
    }
  }

  peers.forEach((peer) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = `peer-card${peer.id === selectedPeerId ? " selected" : ""}`;
    button.setAttribute("aria-pressed", peer.id === selectedPeerId ? "true" : "false");

    const avatar = document.createElement("span");
    avatar.className = "peer-avatar";
    avatar.textContent = initials(peer.name);
    if (!peer.manual) {
      const presence = document.createElement("span");
      presence.className = "peer-presence";
      presence.setAttribute("aria-hidden", "true");
      avatar.append(presence);
    }
    const copy = document.createElement("span");
    copy.className = "peer-copy";
    const name = document.createElement("strong");
    name.textContent = peer.name;
    const address = document.createElement("small");
    address.textContent = `${peer.ip}:${peer.port}${peer.manual ? " · thủ công" : " · trực tuyến"}`;
    copy.append(name, address);
    const check = document.createElement("span");
    check.className = "peer-check";
    check.setAttribute("aria-hidden", "true");
    button.append(avatar, copy, check);
    button.addEventListener("click", () => {
      selectedPeerId = peer.id;
      saveSelectedPeer();
      render();
    });
    ui.peerList.append(button);
  });
}

function renderActions(peers) {
  const selected = peers.find((peer) => peer.id === selectedPeerId);
  const enabled = Boolean(selected) && !sending;
  ui.selectedLabel.textContent = selected ? `Đến ${selected.name}` : "Chưa chọn máy";
  if (!selected) {
    ui.actionHint.textContent = "Chọn một máy nhận để bật các lựa chọn gửi.";
  } else if (sending) {
    ui.actionHint.textContent = `Đang gửi đến ${selected.name}…`;
  } else {
    ui.actionHint.textContent = `Sẵn sàng gửi đến ${selected.name}.`;
  }
  ui.clipboardButton.disabled = !enabled || !state.clipboardAvailable;
  ui.fileInput.disabled = !enabled;
  ui.dropZone.classList.toggle("disabled", !enabled);
  ui.dropZone.setAttribute("aria-disabled", enabled ? "false" : "true");
}

function renderEvents(events) {
  ui.historySection.hidden = events.length === 0;
  ui.eventList.replaceChildren();
  events.slice(0, 8).forEach((event) => {
    const item = document.createElement("li");
    item.className = "event-item";
    const icon = document.createElement("span");
    icon.className = `event-icon${event.status === "failed" ? " failed" : ""}`;
    icon.append(createEventIcon(event));
    const copy = document.createElement("span");
    copy.className = "event-copy";
    const title = document.createElement("strong");
    title.textContent = event.title;
    const detail = document.createElement("small");
    detail.textContent = event.detail;
    copy.append(title, detail);
    const time = document.createElement("time");
    time.className = "event-time";
    time.dateTime = event.at;
    time.textContent = relativeTime(event.at);
    item.append(icon, copy, time);
    ui.eventList.append(item);
  });
}

function createEventIcon(event) {
  const svg = document.createElementNS(SVG_NAMESPACE, "svg");
  svg.setAttribute("viewBox", "0 0 20 20");
  svg.setAttribute("aria-hidden", "true");

  let paths;
  if (event.status === "failed") {
    paths = ["M10 3.25a6.75 6.75 0 1 1 0 13.5 6.75 6.75 0 0 1 0-13.5Z", "M10 6.5v4.25", "M10 13.5h.01"];
  } else if (event.kind === "clipboard") {
    paths = ["M7.25 5.25h-1A1.25 1.25 0 0 0 5 6.5v9.25h10V6.5a1.25 1.25 0 0 0-1.25-1.25h-1", "M7.5 3.75h5v3h-5z"];
  } else if (event.direction === "received") {
    paths = ["M10 3.5v8", "m6.75 8.25 3.25 3.25 3.25-3.25", "M4.5 13v2.5h11V13"];
  } else {
    paths = ["M10 11.5v-8", "m6.75 6.75 3.25-3.25 3.25 3.25", "M4.5 13v2.5h11V13"];
  }

  paths.forEach((data) => {
    const path = document.createElementNS(SVG_NAMESPACE, "path");
    path.setAttribute("d", data);
    svg.append(path);
  });
  return svg;
}

function renderOwnAddress() {
  const addresses = state.self.addresses || [];
  ui.ownAddress.replaceChildren();
  if (addresses.length === 0) {
    ui.ownAddress.textContent = "Không xác định được địa chỉ mạng của máy này.";
    return;
  }
  ui.ownAddress.append(document.createTextNode("Địa chỉ máy này: "));
  addresses.forEach((address, index) => {
    if (index > 0) ui.ownAddress.append(document.createTextNode(" · "));
    const full = `${address}:${state.self.port}`;
    const button = document.createElement("button");
    button.type = "button";
    button.textContent = full;
    button.title = "Bấm để sao chép";
    button.addEventListener("click", async () => {
      await navigator.clipboard.writeText(full);
      showToast("Đã sao chép địa chỉ");
    });
    ui.ownAddress.append(button);
  });
}

function saveSelectedPeer() {
  if (selectedPeerId) localStorage.setItem("truyen-tep-selected-peer", selectedPeerId);
  else localStorage.removeItem("truyen-tep-selected-peer");
}

function initials(name) {
  return String(name || "M")
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => Array.from(word)[0])
    .join("")
    .toUpperCase();
}

function selectedPeer() {
  return state?.peers?.find((peer) => peer.id === selectedPeerId) || null;
}

ui.clipboardButton.addEventListener("click", async () => {
  const peer = selectedPeer();
  if (!peer || sending) return;
  setSending(true);
  try {
    const result = await api("/api/send/clipboard", {
      method: "POST",
      body: JSON.stringify({ peerId: peer.id }),
    });
    showToast(`Đã gửi ${result.characters} ký tự đến ${peer.name}`);
  } catch (error) {
    showToast(error.message, true);
  } finally {
    setSending(false);
    loadState();
  }
});

ui.fileInput.addEventListener("change", () => {
  const files = Array.from(ui.fileInput.files || []);
  ui.fileInput.value = "";
  sendFiles(files);
});

["dragenter", "dragover"].forEach((type) => {
  ui.dropZone.addEventListener(type, (event) => {
    event.preventDefault();
    if (!ui.dropZone.classList.contains("disabled")) ui.dropZone.classList.add("dragging");
  });
});

["dragleave", "drop"].forEach((type) => {
  ui.dropZone.addEventListener(type, (event) => {
    event.preventDefault();
    ui.dropZone.classList.remove("dragging");
  });
});

ui.dropZone.addEventListener("drop", (event) => {
  if (ui.dropZone.classList.contains("disabled")) return;
  sendFiles(Array.from(event.dataTransfer?.files || []));
});

async function sendFiles(files) {
  const peer = selectedPeer();
  if (!peer) {
    showToast("Hãy chọn máy nhận trước", true);
    return;
  }
  if (sending || files.length === 0) return;
  const tooLarge = files.find((file) => file.size > 4 * 1024 * 1024 * 1024);
  if (tooLarge) {
    showToast(`${tooLarge.name} lớn quá 4 GB`, true);
    return;
  }

  setSending(true);
  ui.progressBox.hidden = false;
  const total = files.reduce((sum, file) => sum + file.size, 0) || 1;
  let completed = 0;
  let sentCount = 0;
  try {
    for (let index = 0; index < files.length; index += 1) {
      const file = files[index];
      ui.progressTitle.textContent = files.length === 1 ? "Đang gửi tệp…" : `Đang gửi ${index + 1}/${files.length} tệp…`;
      ui.progressDetail.textContent = `${file.name} · ${formatBytes(file.size)}`;
      await sendFile(file, peer.id, (loaded) => updateProgress((completed + loaded) / total));
      completed += file.size;
      sentCount += 1;
      updateProgress(completed / total);
    }
    showToast(`Đã gửi ${sentCount} tệp đến ${peer.name}`);
  } catch (error) {
    showToast(`${sentCount > 0 ? `Đã gửi ${sentCount}/${files.length} tệp. ` : ""}${error.message}`, true);
  } finally {
    setSending(false);
    window.setTimeout(() => { ui.progressBox.hidden = true; updateProgress(0); }, 600);
    loadState();
  }
}

function sendFile(file, peerId, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    const query = new URLSearchParams({ peer: peerId, name: file.name });
    xhr.open("POST", `/api/send/file?${query}`);
    xhr.setRequestHeader("X-Truyen-Tep-Local", "1");
    xhr.setRequestHeader("Content-Type", "application/octet-stream");
    xhr.upload.addEventListener("progress", (event) => {
      if (event.lengthComputable) onProgress(event.loaded);
    });
    xhr.addEventListener("load", () => {
      if (xhr.status >= 200 && xhr.status < 300) {
        resolve();
        return;
      }
      try {
        reject(new Error(JSON.parse(xhr.responseText).error || "Không gửi được tệp"));
      } catch (_) {
        reject(new Error("Không gửi được tệp"));
      }
    });
    xhr.addEventListener("error", () => reject(new Error("Mất kết nối khi gửi tệp")));
    xhr.addEventListener("abort", () => reject(new Error("Đã hủy gửi tệp")));
    xhr.send(file);
  });
}

function updateProgress(ratio) {
  const percent = Math.max(0, Math.min(100, Math.round(ratio * 100)));
  ui.progressPercent.textContent = `${percent}%`;
  ui.progressBar.style.width = `${percent}%`;
}

function setSending(value) {
  sending = value;
  renderActions(state?.peers || []);
}

ui.manualForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  const address = ui.manualAddress.value.trim();
  if (!address) return;
  const button = ui.manualForm.querySelector("button[type=submit]");
  button.disabled = true;
  button.textContent = "Đang nối…";
  try {
    const peer = await api("/api/peers/manual", { method: "POST", body: JSON.stringify({ address }) });
    selectedPeerId = peer.id;
    saveSelectedPeer();
    ui.manualAddress.value = "";
    showToast(`Đã kết nối ${peer.name}`);
    await loadState();
  } catch (error) {
    showToast(error.message, true);
  } finally {
    button.disabled = false;
    button.textContent = "Kết nối";
  }
});

ui.refreshButton.addEventListener("click", () => loadState(true));

ui.downloadsButton.addEventListener("click", async () => {
  try {
    await api("/api/open-downloads", { method: "POST" });
  } catch (error) {
    showToast(error.message, true);
  }
});

ui.quitButton.addEventListener("click", () => ui.quitDialog.showModal());
ui.confirmQuit.addEventListener("click", async (event) => {
  event.preventDefault();
  try {
    await api("/api/quit", { method: "POST" });
    ui.quitDialog.close();
    window.clearInterval(pollTimer);
    document.body.innerHTML = '<main class="shell"><section class="panel empty-state"><strong>Truyền Tệp đã dừng</strong><p>Bạn có thể đóng thẻ trình duyệt này.</p></section></main>';
  } catch (error) {
    showToast(error.message, true);
  }
});

function formatBytes(bytes) {
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = units[0];
  for (let index = 1; value >= 1024 && index < units.length; index += 1) {
    value /= 1024;
    unit = units[index];
  }
  return `${value < 10 ? value.toFixed(1) : Math.round(value)} ${unit}`;
}

function relativeTime(timestamp) {
  const date = new Date(timestamp);
  const seconds = Math.max(0, Math.round((Date.now() - date.getTime()) / 1000));
  if (seconds < 45) return "vừa xong";
  if (seconds < 3600) return `${Math.round(seconds / 60)} phút`;
  if (seconds < 86400) return `${Math.round(seconds / 3600)} giờ`;
  return new Intl.DateTimeFormat("vi-VN", { day: "2-digit", month: "2-digit" }).format(date);
}

function showToast(message, error = false) {
  window.clearTimeout(toastTimer);
  ui.toast.textContent = message;
  ui.toast.classList.toggle("error", error);
  ui.toast.classList.add("show");
  toastTimer = window.setTimeout(() => ui.toast.classList.remove("show"), 3200);
}

loadState(true);
pollTimer = window.setInterval(() => {
  if (stateLoadsInFlight > 0) return;
  loadState();
}, 2000);
