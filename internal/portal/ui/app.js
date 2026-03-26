const TARGET_STORAGE_KEY = "penframe.portal.target";
const CHAIN_EDITOR_COLLAPSE_KEY = "penframe.portal.chainEditorCollapsed";

const state = {
  portal: null,
  selectedRunId: null,
  activeTab: "task",
  chainEditorCollapsed: false,
  runInProgress: false,
  runStartedAt: 0,
  chain: {
    run_nmap_quick: true,
    run_nmap_full: false,
    run_fscan_web: true,
    run_nuclei_web: true,
    nmap_quick_args: "",
    nmap_full_args: "",
    fscan_args: "",
    nuclei_args: "",
  },
};

const workflowNameEl = document.getElementById("workflowName");
const runStatusEl = document.getElementById("runStatus");
const reloadButton = document.getElementById("reloadButton");
const targetInput = document.getElementById("targetInput");
const startTaskButton = document.getElementById("startTaskButton");
const taskStatusEl = document.getElementById("taskStatus");
const chainBuilderPaneEl = document.getElementById("chainBuilderPane");
const chainBuilderBodyEl = document.getElementById("chainBuilderBody");
const chainCollapseButtonEl = document.getElementById("chainCollapseButton");
const chainEditorEl = document.getElementById("chainEditor");
const chainPreviewEl = document.getElementById("chainPreview");
const toolCallsEl = document.getElementById("toolCalls");
const rawOutputsEl = document.getElementById("rawOutputs");
const parsedOutputEl = document.getElementById("parsedOutput");
const runListEl = document.getElementById("runList");
const selectedRunJsonEl = document.getElementById("selectedRunJson");
const workflowMetaEl = document.getElementById("workflowMeta");
const toolCatalogEl = document.getElementById("toolCatalog");
const tabButtons = Array.from(document.querySelectorAll("[data-tab]"));
const tabPanels = Array.from(document.querySelectorAll("[data-panel]"));
const chainPresetButtons = Array.from(document.querySelectorAll("[data-chain-preset]"));
let runProgressTimer = null;

const CHAIN_PRESETS = {
  baseline: {
    run_nmap_quick: true,
    run_nmap_full: false,
    run_fscan_web: true,
    run_nuclei_web: true,
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    nmap_full_args: "-Pn -n -sS -p-",
    fscan_args: "-nobr -np -json -f json -log INFO",
    nuclei_args: "-as -severity info,low,medium,high,critical -rate-limit 120 -timeout 10 -retries 2 -duc -ni -sr -nmhe -stats -si 5 -nc",
  },
  full: {
    run_nmap_quick: true,
    run_nmap_full: true,
    run_fscan_web: true,
    run_nuclei_web: true,
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    nmap_full_args: "-Pn -n -sS -p- --min-rate 3000",
    fscan_args: "-nobr -np -json -f json -log INFO -m Web",
    nuclei_args: "-as -severity info,low,medium,high,critical -rate-limit 120 -timeout 10 -retries 2 -duc -ni -sr -nmhe -stats -si 5 -nc",
  },
  "quick-only": {
    run_nmap_quick: true,
    run_nmap_full: false,
    run_fscan_web: false,
    run_nuclei_web: false,
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    nmap_full_args: "-Pn -n -sS -p-",
    fscan_args: "-nobr -np -json -f json -log INFO",
    nuclei_args: "-as -severity info,low,medium,high,critical -rate-limit 120 -timeout 10 -retries 2 -duc -ni -sr -nmhe -stats -si 5 -nc",
  },
};

function restoreTargetInput() {
  const savedTarget = localStorage.getItem(TARGET_STORAGE_KEY) || "";
  if (targetInput && !targetInput.value) {
    targetInput.value = savedTarget;
  }
}

function restoreChainEditorState() {
  state.chainEditorCollapsed = localStorage.getItem(CHAIN_EDITOR_COLLAPSE_KEY) === "1";
}

function hydrateChainDefaults() {
  const vars = state.portal?.workflow?.global_vars || {};
  state.chain.run_nmap_quick = toBool(vars.run_nmap_quick, true);
  state.chain.run_nmap_full = toBool(vars.run_nmap_full, false);
  state.chain.run_fscan_web = toBool(vars.run_fscan_web, true);
  state.chain.run_nuclei_web = toBool(vars.run_nuclei_web, true);
  state.chain.nmap_quick_args = String(vars.nmap_quick_args || "");
  state.chain.nmap_full_args = String(vars.nmap_full_args || "");
  state.chain.fscan_args = String(vars.fscan_args || "");
  state.chain.nuclei_args = String(vars.nuclei_args || "");
}

function toBool(value, fallback) {
  if (typeof value === "boolean") {
    return value;
  }
  if (value === "true") {
    return true;
  }
  if (value === "false") {
    return false;
  }
  return fallback;
}

async function fetchJSON(url, options = {}) {
  const response = await fetch(url, options);
  const payload = await response.json();
  if (!response.ok) {
    throw new Error(payload.error || `请求失败：${response.status}`);
  }
  return payload;
}

async function loadState() {
  const payload = await fetchJSON("/api/state");
  state.portal = payload;
  state.selectedRunId = payload.latest_run ? payload.latest_run.id : null;
  restoreTargetInput();
  restoreChainEditorState();
  hydrateChainDefaults();
  render();
}

function currentRun() {
  if (!state.portal) {
    return null;
  }
  if (!state.selectedRunId) {
    return state.portal.latest_run || null;
  }
  return state.portal.recent_runs.find((run) => run.id === state.selectedRunId) || state.portal.latest_run || null;
}

function render() {
  if (!state.portal) {
    return;
  }
  const run = currentRun();
  workflowNameEl.textContent = state.portal.workflow.name || "未命名";
  renderTopStatus(run);
  renderTaskPanel(run);
  renderRunPanel(run);
  renderSystemPanel();
  renderTabs();
}

function renderTopStatus(run) {
  if (!run) {
    runStatusEl.className = "status-pill";
    runStatusEl.textContent = "空闲";
    return;
  }
  const status = run.summary.status;
  runStatusEl.className = `status-pill ${status === "succeeded" ? "success" : status === "failed" ? "error" : ""}`;
  runStatusEl.textContent = `${statusLabel(status)} · ${formatTime(run.summary.finished_at)}`;
}

function renderTaskPanel(run) {
  renderChainEditor();
  renderChainLayout();
  renderChainPreview(run);
  renderToolCalls(run);
  renderRawOutputs(run);
  renderParsedOutput(run);
}

function renderChainLayout() {
  if (chainBuilderPaneEl) {
    chainBuilderPaneEl.classList.toggle("is-collapsed", state.chainEditorCollapsed);
  }
  if (chainBuilderBodyEl) {
    chainBuilderBodyEl.hidden = state.chainEditorCollapsed;
  }
  if (chainCollapseButtonEl) {
    chainCollapseButtonEl.textContent = state.chainEditorCollapsed ? "展开配置" : "收起配置";
  }
}

function renderChainEditor() {
  chainEditorEl.innerHTML = `
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nmap_quick" ${state.chain.run_nmap_quick ? "checked" : ""}>
        Nmap 快速 Web/TLS 探测
      </label>
      <input class="chain-args" type="text" data-chain-field="nmap_quick_args" value="${escapeHTML(state.chain.nmap_quick_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nmap_full" ${state.chain.run_nmap_full ? "checked" : ""}>
        Nmap 全端口 SYN 扫描
      </label>
      <input class="chain-args" type="text" data-chain-field="nmap_full_args" value="${escapeHTML(state.chain.nmap_full_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_fscan_web" ${state.chain.run_fscan_web ? "checked" : ""}>
        fscan Web 任务
      </label>
      <input class="chain-args" type="text" data-chain-field="fscan_args" value="${escapeHTML(state.chain.fscan_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nuclei_web" ${state.chain.run_nuclei_web ? "checked" : ""}>
        nuclei CVE 模板检测
      </label>
      <input class="chain-args" type="text" data-chain-field="nuclei_args" value="${escapeHTML(state.chain.nuclei_args)}">
    </article>
  `;
}

function renderChainPreview(run) {
  const target = String(targetInput?.value || "").trim();
  const host = extractTargetHost(target);
  const url = normalizeTargetURL(target || host || "target.local");

  const nmapPath = toolBinaryPath("nmap_scan");
  const fscanPath = toolBinaryPath("fscan_scan");
  const nucleiPath = toolBinaryPath("nuclei_scan");
  const outputWindows = `<运行时自动生成>/output/${sanitizeTarget(host || "target")}`;

  const preview = [];
  if (state.chain.run_nmap_quick) {
    preview.push({
      nodeID: "nmap_quick",
      name: "nmap_quick",
      cmd: `"${nmapPath}" ${state.chain.nmap_quick_args} -oN "${outputWindows}\\nmap-quick.txt" ${host || "<target_host>"}`,
    });
  }
  if (state.chain.run_nmap_full) {
    preview.push({
      nodeID: "nmap_full",
      name: "nmap_full",
      cmd: `"${nmapPath}" ${state.chain.nmap_full_args} -oN "${outputWindows}\\nmap-full.txt" ${host || "<target_host>"}`,
    });
  }
  if (state.chain.run_fscan_web) {
    preview.push({
      nodeID: "fscan_web",
      name: "fscan_web",
      cmd: `"${fscanPath}" -u ${url} ${state.chain.fscan_args} -o "${outputWindows}\\fscan-web-live.json"`,
    });
  }
  if (state.chain.run_nuclei_web) {
    preview.push({
      nodeID: "nuclei_web",
      name: "nuclei_web",
      cmd: `"${nucleiPath}" -u ${url} ${state.chain.nuclei_args} -o "${outputWindows}\\nuclei-web-live.txt"`,
    });
  }

  if (preview.length === 0) {
    chainPreviewEl.innerHTML = `<div class="empty">当前没有启用任何工具节点。</div>`;
    return;
  }

  const activeRun = run || currentRun();
  let unresolvedIndex = state.runInProgress
    ? preview.findIndex((item) => !activeRun?.summary?.node_results?.[item.nodeID])
    : -1;
  if (state.runInProgress && unresolvedIndex < 0 && preview.length > 0) {
    unresolvedIndex = 0;
  }

  chainPreviewEl.innerHTML = preview
    .map((item, index) => {
      const stage = chainNodeStage(item.nodeID, activeRun, state.runInProgress, unresolvedIndex, index);
      return `
      <article class="item">
        <div class="item-head">
          <div class="item-title">步骤 ${index + 1} · ${escapeHTML(item.name)}</div>
          <span class="chip ${escapeHTML(stage.className)}">${escapeHTML(stage.label)}</span>
        </div>
        <p class="item-meta">输出目录：${escapeHTML(outputWindows)}</p>
        <p class="item-meta">阶段说明：${escapeHTML(stage.hint)}</p>
        <pre class="cmd">${escapeHTML(item.cmd)}</pre>
      </article>
    `;
    })
    .join("");
}

function chainNodeStage(nodeID, run, runInProgress, unresolvedIndex, currentIndex) {
  const node = run?.summary?.node_results?.[nodeID];
  if (node?.status === "succeeded") {
    return { className: "succeeded", label: "已完成", hint: "节点执行成功，结果已回填。" };
  }
  if (node?.status === "failed") {
    return { className: "failed", label: "失败", hint: node.error || "节点执行失败，请查看命令原始输出。" };
  }
  if (node?.status === "skipped") {
    return { className: "skipped", label: "跳过", hint: node.skip_reason || "未满足条件，节点被跳过。" };
  }

  if (runInProgress && unresolvedIndex >= 0 && unresolvedIndex === currentIndex) {
    return {
      className: "running",
      label: "运行中",
      hint: "任务正在执行中（当前版本在任务结束后统一回填节点结果）。",
    };
  }
  if (runInProgress && unresolvedIndex >= 0 && currentIndex > unresolvedIndex) {
    return { className: "", label: "排队中", hint: "等待前序节点执行完成后开始。" };
  }
  if (run?.summary?.status === "failed") {
    return { className: "not_started", label: "未执行", hint: "本次任务中断前未进入该节点。" };
  }
  return { className: "", label: "待执行", hint: "尚未开始本次任务。" };
}

function toolBinaryPath(toolName) {
  const tool = (state.portal?.tools || []).find((item) => item.name === toolName);
  return String(tool?.metadata?.binary_path || tool?.name || toolName);
}

function renderToolCalls(run) {
  if (!run) {
    toolCallsEl.innerHTML = `<div class="empty">还没有任务运行记录。</div>`;
    return;
  }

  const nodeIds = run.summary.execution_order.length > 0
    ? run.summary.execution_order
    : Object.keys(run.summary.node_results || {});
  if (nodeIds.length === 0) {
    toolCallsEl.innerHTML = `<div class="empty">本次任务没有可展示的节点执行信息。</div>`;
    return;
  }

  toolCallsEl.innerHTML = nodeIds
    .map((nodeID) => {
      const node = run.summary.node_results[nodeID];
      if (!node) {
        return "";
      }
      return `
        <article class="item">
          <div class="item-head">
            <div class="item-title">${escapeHTML(node.node_id)}</div>
            <span class="chip ${escapeHTML(node.status)}">${escapeHTML(statusLabel(node.status))}</span>
          </div>
          <p class="item-meta">工具：${escapeHTML(node.tool)} · 执行器：${escapeHTML(node.executor)}</p>
          <pre class="cmd">${escapeHTML(node.rendered_command || "(无命令)")}</pre>
        </article>
      `;
    })
    .join("");
}

function renderRawOutputs(run) {
  if (!run) {
    rawOutputsEl.innerHTML = `<div class="empty">运行任务后会显示原始输出。</div>`;
    return;
  }

  const nodeIds = run.summary.execution_order.length > 0
    ? run.summary.execution_order
    : Object.keys(run.summary.node_results || {});
  rawOutputsEl.innerHTML = nodeIds
    .map((nodeID) => {
      const node = run.summary.node_results[nodeID];
      if (!node) {
        return "";
      }
      const rawText = mergeRawOutput(node);
      return `
        <article class="item">
          <div class="item-head">
            <div class="item-title">${escapeHTML(node.node_id)}</div>
            <span class="chip">${escapeHTML(node.tool)}</span>
          </div>
          <pre class="code-block">${escapeHTML(rawText)}</pre>
        </article>
      `;
    })
    .join("");
}

function mergeRawOutput(node) {
  const chunks = [];
  const stdout = String(node.stdout || "").trim();
  const stderr = String(node.metadata?.stderr || "").trim();
  if (stdout) {
    chunks.push(stdout);
  }
  if (stderr) {
    chunks.push(`[stderr]\n${stderr}`);
  }

  const outputFiles = Array.isArray(node.metadata?.output_files) ? node.metadata.output_files : [];
  outputFiles.forEach((entry) => {
    const path = String(entry.path || "(unknown)");
    if (entry.error) {
      chunks.push(`[output file] ${path}\n[read error] ${entry.error}`);
      return;
    }
    const content = String(entry.content || "").trim();
    chunks.push(`[output file] ${path}\n${content || "(empty file)"}`);
  });

  if (chunks.length === 0) {
    return "(无输出)";
  }
  return chunks.join("\n\n====================\n\n");
}

function renderParsedOutput(run) {
  if (!run) {
    parsedOutputEl.textContent = "{}";
    return;
  }

  const parsedRecords = {};
  Object.entries(run.summary.node_results || {}).forEach(([nodeID, node]) => {
    if (Array.isArray(node.records) && node.records.length > 0) {
      parsedRecords[nodeID] = node.records;
    }
  });

  const payload = {
    workflow: run.summary.workflow,
    status: run.summary.status,
    finished_at: run.summary.finished_at,
    vars: run.summary.vars,
    stats: run.summary.stats,
    assets: run.summary.assets,
    parsed_records: parsedRecords,
  };
  parsedOutputEl.textContent = JSON.stringify(payload, null, 2);
}

function renderRunPanel(run) {
  renderRunList();
  selectedRunJsonEl.textContent = run ? JSON.stringify(run.summary, null, 2) : "{}";
}

function renderRunList() {
  const runs = state.portal.recent_runs || [];
  if (runs.length === 0) {
    runListEl.innerHTML = `<div class="empty">暂无运行记录。</div>`;
    return;
  }
  runListEl.innerHTML = runs
    .map((run) => {
      const active = run.id === state.selectedRunId ? "active" : "";
      return `
        <article class="item run-item ${active}" data-run-id="${escapeHTML(run.id)}">
          <div class="item-head">
            <div class="item-title">${escapeHTML(run.id)}</div>
            <span class="chip ${escapeHTML(run.summary.status)}">${escapeHTML(statusLabel(run.summary.status))}</span>
          </div>
          <p class="item-meta">${escapeHTML(formatTime(run.summary.finished_at))}</p>
          <p class="item-meta">执行节点：${escapeHTML(String(run.summary.stats.executed_nodes))}</p>
        </article>
      `;
    })
    .join("");
}

function renderSystemPanel() {
  const metaPayload = {
    workflow: state.portal.workflow.name,
    description: state.portal.workflow.description,
    node_count: state.portal.workflow_meta.node_count,
    edge_count: state.portal.workflow_meta.edge_count,
    entry_nodes: state.portal.workflow_meta.entry_nodes,
    paths: state.portal.paths,
    external_root: state.portal.external_root,
  };
  workflowMetaEl.textContent = JSON.stringify(metaPayload, null, 2);

  const tools = state.portal.tools || [];
  if (tools.length === 0) {
    toolCatalogEl.innerHTML = `<div class="empty">工具目录为空。</div>`;
    return;
  }
  toolCatalogEl.innerHTML = tools
    .map((tool) => `
      <article class="item">
        <div class="item-head">
          <div class="item-title">${escapeHTML(tool.name)}</div>
          <span class="chip">${escapeHTML(tool.category || "unknown")}</span>
        </div>
        <p class="item-meta">${escapeHTML(tool.description || "无描述")}</p>
        <pre class="cmd">${escapeHTML(tool.command_template || "(无命令模板)")}</pre>
      </article>
    `)
    .join("");
}

function renderTabs() {
  tabButtons.forEach((button) => {
    button.classList.toggle("is-active", button.dataset.tab === state.activeTab);
  });
  tabPanels.forEach((panel) => {
    panel.classList.toggle("is-active", panel.dataset.panel === state.activeTab);
  });
}

function setActiveTab(tab) {
  state.activeTab = tab;
  renderTabs();
}

function applyChainPreset(presetName) {
  const preset = CHAIN_PRESETS[presetName];
  if (!preset) {
    return;
  }
  state.chain = {
    ...state.chain,
    ...preset,
  };
  taskStatusEl.textContent = `已应用预置：${presetName}，你可以继续手动修改参数。`;
  renderChainEditor();
  renderChainPreview(currentRun());
}

function startRunProgressHint() {
  stopRunProgressHint();
  state.runInProgress = true;
  state.runStartedAt = Date.now();
  runProgressTimer = window.setInterval(() => {
    const elapsed = Math.max(0, Math.floor((Date.now() - state.runStartedAt) / 1000));
    taskStatusEl.textContent = `任务执行中，已等待 ${formatElapsed(elapsed)}。当前阶段为估算展示，节点结果将在任务结束后回填。`;
    renderChainPreview(currentRun());
  }, 1000);
}

function stopRunProgressHint() {
  state.runInProgress = false;
  state.runStartedAt = 0;
  if (runProgressTimer) {
    window.clearInterval(runProgressTimer);
    runProgressTimer = null;
  }
}

async function runTask() {
  const target = String(targetInput?.value || "").trim();
  if (!target) {
    taskStatusEl.textContent = "请先输入靶机地址。";
    runStatusEl.className = "status-pill error";
    runStatusEl.textContent = "输入无效";
    return;
  }

  localStorage.setItem(TARGET_STORAGE_KEY, target);
  startTaskButton.disabled = true;
  reloadButton.disabled = true;
  startRunProgressHint();
  runStatusEl.className = "status-pill";
  runStatusEl.textContent = "任务运行中...";
  taskStatusEl.textContent = "任务已提交，正在执行。";
  renderChainPreview(currentRun());

  try {
    const payload = await fetchJSON("/api/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        target,
        vars: {
          run_nmap_quick: state.chain.run_nmap_quick,
          run_nmap_full: state.chain.run_nmap_full,
          run_fscan_web: state.chain.run_fscan_web,
          run_nuclei_web: state.chain.run_nuclei_web,
          nmap_quick_args: state.chain.nmap_quick_args,
          nmap_full_args: state.chain.nmap_full_args,
          fscan_args: state.chain.fscan_args,
          nuclei_args: state.chain.nuclei_args,
        },
      }),
    });
    mergeRun(payload.run);
    state.selectedRunId = payload.run.id;

    if (payload.run.summary.status === "failed") {
      runStatusEl.className = "status-pill error";
      runStatusEl.textContent = payload.error || payload.run.summary.error || "任务失败";
      taskStatusEl.textContent = runStatusEl.textContent;
    } else {
      runStatusEl.className = "status-pill success";
      runStatusEl.textContent = `任务完成 · ${formatTime(payload.run.summary.finished_at)}`;
      taskStatusEl.textContent = "任务执行完成，结果已更新。";
    }
    render();
  } catch (error) {
    runStatusEl.className = "status-pill error";
    runStatusEl.textContent = error.message;
    taskStatusEl.textContent = error.message;
  } finally {
    stopRunProgressHint();
    startTaskButton.disabled = false;
    reloadButton.disabled = false;
    renderChainPreview(currentRun());
  }
}

function mergeRun(run) {
  if (!state.portal) {
    return;
  }
  state.portal.latest_run = run;
  const existing = state.portal.recent_runs || [];
  state.portal.recent_runs = [run, ...existing.filter((item) => item.id !== run.id)].slice(0, 20);
}

async function reloadConfig() {
  reloadButton.disabled = true;
  startTaskButton.disabled = true;
  runStatusEl.className = "status-pill";
  runStatusEl.textContent = "刷新中...";
  try {
    state.portal = await fetchJSON("/api/reload", { method: "POST" });
    hydrateChainDefaults();
    if (!state.portal.recent_runs.some((run) => run.id === state.selectedRunId)) {
      state.selectedRunId = state.portal.latest_run ? state.portal.latest_run.id : null;
    }
    render();
    runStatusEl.className = "status-pill success";
    runStatusEl.textContent = "配置已刷新";
  } catch (error) {
    runStatusEl.className = "status-pill error";
    runStatusEl.textContent = error.message;
  } finally {
    reloadButton.disabled = false;
    startTaskButton.disabled = false;
  }
}

function statusLabel(status) {
  const mapping = {
    running: "运行中",
    succeeded: "成功",
    failed: "失败",
    skipped: "跳过",
    pending: "等待",
  };
  return mapping[status] || status || "未知";
}

function extractTargetHost(raw) {
  const input = String(raw || "").trim();
  if (!input) {
    return "";
  }
  try {
    const parsed = new URL(input.includes("://") ? input : `https://${input}`);
    return parsed.hostname || input;
  } catch (_error) {
    return input;
  }
}

function normalizeTargetURL(raw) {
  const input = String(raw || "").trim();
  if (!input) {
    return "";
  }
  if (input.includes("://")) {
    return input;
  }
  return `https://${input}`;
}

function sanitizeTarget(raw) {
  const text = String(raw || "target").trim();
  return text.replace(/[^A-Za-z0-9._-]+/g, "_").replace(/^[_\-.]+|[_\-.]+$/g, "") || "target";
}

function formatTime(value) {
  if (!value) {
    return "无";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }
  return date.toLocaleString("zh-CN");
}

function formatElapsed(seconds) {
  if (seconds < 60) {
    return `${seconds}秒`;
  }
  const mins = Math.floor(seconds / 60);
  const secs = seconds % 60;
  return `${mins}分${secs}秒`;
}

function escapeHTML(value) {
  return String(value ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

tabButtons.forEach((button) => {
  button.addEventListener("click", () => {
    setActiveTab(button.dataset.tab);
  });
});
chainPresetButtons.forEach((button) => {
  button.addEventListener("click", () => {
    applyChainPreset(button.dataset.chainPreset);
  });
});

startTaskButton.addEventListener("click", runTask);
targetInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter") {
    event.preventDefault();
    runTask();
  }
});
targetInput.addEventListener("input", () => {
  renderChainPreview(currentRun());
});
reloadButton.addEventListener("click", reloadConfig);
chainCollapseButtonEl?.addEventListener("click", () => {
  state.chainEditorCollapsed = !state.chainEditorCollapsed;
  localStorage.setItem(CHAIN_EDITOR_COLLAPSE_KEY, state.chainEditorCollapsed ? "1" : "0");
  renderChainLayout();
});

document.addEventListener("input", (event) => {
  const field = event.target?.dataset?.chainField;
  if (!field) {
    return;
  }
  if (field === "run_nmap_quick" || field === "run_nmap_full" || field === "run_fscan_web" || field === "run_nuclei_web") {
    state.chain[field] = Boolean(event.target.checked);
  } else {
    state.chain[field] = String(event.target.value || "");
  }
  renderChainPreview(currentRun());
});

runListEl.addEventListener("click", (event) => {
  const item = event.target.closest("[data-run-id]");
  if (!item) {
    return;
  }
  state.selectedRunId = item.dataset.runId || null;
  render();
});

loadState().catch((error) => {
  runStatusEl.className = "status-pill error";
  runStatusEl.textContent = error.message;
  taskStatusEl.textContent = error.message;
});
