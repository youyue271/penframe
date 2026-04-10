const TARGET_STORAGE_KEY = "penframe.portal.target";
const CHAIN_EDITOR_COLLAPSE_KEY = "penframe.portal.chainEditorCollapsed";

const state = {
  portal: null,
  selectedRunId: null,
  activeTab: "task",
  chainEditorCollapsed: false,
  runInProgress: false,
  runStartedAt: 0,
  eventSource: null,
  chain: {
    run_asset_seed: true,
    run_host_discovery: true,
    run_entry_discovery: true,
    run_nmap_quick: true,
    run_fscan_web: true,
    run_nuclei_fingerprint: true,
    run_nuclei_cve: true,
    run_executor_placeholder: true,
    fscan_binary_path: "",
    nuclei_binary_path: "",
    nmap_quick_args: "",
    fscan_args: "",
    nuclei_fingerprint_templates: "cve/nuclei/fingerprint",
    nuclei_cve_templates: "cve/nuclei/cves",
    nuclei_fingerprint_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    nuclei_cve_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    executor_name: "custom-executor",
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
const reconSummaryEl = document.getElementById("reconSummary");
const httpFingerprintSummaryEl = null;
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
const HTTP_PROBE_NODES = [];
let runProgressTimer = null;

const CHAIN_PRESETS = {
  baseline: {
    run_asset_seed: true,
    run_host_discovery: true,
    run_entry_discovery: true,
    run_nmap_quick: true,
    run_fscan_web: true,
    run_nuclei_fingerprint: true,
    run_nuclei_cve: true,
    run_executor_placeholder: true,
    fscan_binary_path: "/mnt/h/tools/Penetration/tools/01 scan/fscan/fscan.exe",
    nuclei_binary_path: "/mnt/h/tools/Penetration/tools/00 Assemble tool/nuclei/nuclei.exe",
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    fscan_args: "-nobr -np -log INFO -m Web",
    nuclei_fingerprint_templates: "cve/nuclei/fingerprint",
    nuclei_cve_templates: "cve/nuclei/cves",
    nuclei_fingerprint_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    nuclei_cve_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    executor_name: "custom-executor",
  },
  "discovery-only": {
    run_asset_seed: true,
    run_host_discovery: true,
    run_entry_discovery: true,
    run_nmap_quick: true,
    run_fscan_web: true,
    run_nuclei_fingerprint: true,
    run_nuclei_cve: false,
    run_executor_placeholder: false,
    fscan_binary_path: "/mnt/h/tools/Penetration/tools/01 scan/fscan/fscan.exe",
    nuclei_binary_path: "/mnt/h/tools/Penetration/tools/00 Assemble tool/nuclei/nuclei.exe",
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    fscan_args: "-nobr -np -log INFO -m Web",
    nuclei_fingerprint_templates: "cve/nuclei/fingerprint",
    nuclei_cve_templates: "cve/nuclei/cves",
    nuclei_fingerprint_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    nuclei_cve_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    executor_name: "custom-executor",
  },
  "vuln-focused": {
    run_asset_seed: true,
    run_host_discovery: false,
    run_entry_discovery: true,
    run_nmap_quick: true,
    run_fscan_web: true,
    run_nuclei_fingerprint: true,
    run_nuclei_cve: true,
    run_executor_placeholder: true,
    fscan_binary_path: "/mnt/h/tools/Penetration/tools/01 scan/fscan/fscan.exe",
    nuclei_binary_path: "/mnt/h/tools/Penetration/tools/00 Assemble tool/nuclei/nuclei.exe",
    nmap_quick_args: "-Pn -n -p 80,443,3000 -sV --script http-methods,http-security-headers,ssl-enum-ciphers,http-title",
    fscan_args: "-nobr -np -log INFO -m Web",
    nuclei_fingerprint_templates: "cve/nuclei/fingerprint",
    nuclei_cve_templates: "cve/nuclei/cves",
    nuclei_fingerprint_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    nuclei_cve_args: "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1",
    executor_name: "custom-executor",
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
  state.chain.run_asset_seed = toBool(vars.run_asset_seed, true);
  state.chain.run_host_discovery = toBool(vars.run_host_discovery, true);
  state.chain.run_entry_discovery = toBool(vars.run_entry_discovery, true);
  state.chain.run_nmap_quick = toBool(vars.run_nmap_quick, true);
  state.chain.run_fscan_web = toBool(vars.run_fscan_web, true);
  state.chain.run_nuclei_fingerprint = toBool(vars.run_nuclei_fingerprint, true);
  state.chain.run_nuclei_cve = toBool(vars.run_nuclei_cve, true);
  state.chain.run_executor_placeholder = toBool(vars.run_executor_placeholder, true);
  state.chain.fscan_binary_path = String(vars.fscan_binary_path || toolBinaryPath("fscan_scan", vars));
  state.chain.nuclei_binary_path = String(vars.nuclei_binary_path || toolBinaryPath("nuclei_scan", vars));
  state.chain.nmap_quick_args = String(vars.nmap_quick_args || "");
  state.chain.fscan_args = String(vars.fscan_args || "");
  state.chain.nuclei_fingerprint_templates = String(vars.nuclei_fingerprint_templates || "cve/nuclei/fingerprint");
  state.chain.nuclei_cve_templates = String(vars.nuclei_cve_templates || "cve/nuclei/cves");
  state.chain.nuclei_fingerprint_args = String(vars.nuclei_fingerprint_args || "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1");
  state.chain.nuclei_cve_args = String(vars.nuclei_cve_args || "-jsonl -duc -ni -nc -rl 80 -timeout 10 -retries 1");
  state.chain.executor_name = String(vars.executor_name || "custom-executor");
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

function connectEventStream() {
  if (state.eventSource) {
    return;
  }

  const source = new EventSource("/api/events");
  state.eventSource = source;

  const handleEvent = (event) => {
    let payload;
    try {
      payload = JSON.parse(event.data);
    } catch (_error) {
      return;
    }
    mergeStreamEvent(payload);
  };

  ["run_started", "node_started", "node_finished", "run_finished"].forEach((eventName) => {
    source.addEventListener(eventName, handleEvent);
  });

  source.addEventListener("error", () => {
    if (state.eventSource !== source) {
      return;
    }
    if (source.readyState === EventSource.CLOSED) {
      state.eventSource = null;
      window.setTimeout(connectEventStream, 2000);
    }
  });
}

function mergeStreamEvent(payload) {
  if (!state.portal || !payload?.type || !payload?.run_id) {
    return;
  }

  switch (payload.type) {
    case "run_started": {
      const run = ensureRun(payload.run_id, payload.summary);
      if (shouldAutoSelectRun()) {
        state.selectedRunId = payload.run_id;
      }
      run.summary.status = "running";
      state.portal.latest_run = run;
      if (state.selectedRunId === payload.run_id) {
        taskStatusEl.textContent = "任务已开始，正在等待节点执行结果回传。";
      }
      render();
      return;
    }
    case "node_started": {
      const run = ensureRun(payload.run_id);
      if (shouldAutoSelectRun()) {
        state.selectedRunId = payload.run_id;
      }
      mergeNodeIntoRun(run, payload.node);
      run.summary.status = "running";
      if (!run.summary.started_at && payload.node?.started_at) {
        run.summary.started_at = payload.node.started_at;
      }
      if (state.selectedRunId === payload.run_id && payload.node?.node_id) {
        taskStatusEl.textContent = `节点 ${payload.node.node_id} 执行中...`;
      }
      state.portal.latest_run = run;
      render();
      return;
    }
    case "node_finished": {
      const run = ensureRun(payload.run_id, payload.summary);
      if (payload.summary) {
        run.summary = payload.summary;
      }
      mergeNodeIntoRun(run, payload.node);
      if (shouldAutoSelectRun()) {
        state.selectedRunId = payload.run_id;
      }
      state.portal.latest_run = run;
      if (state.selectedRunId === payload.run_id && payload.node) {
        taskStatusEl.textContent = nodeStatusMessage(payload.node);
      }
      render();
      return;
    }
    case "run_finished": {
      const run = mergeRun({
        id: payload.run_id,
        summary: payload.summary || createPlaceholderSummary(),
      });
      if (shouldAutoSelectRun()) {
        state.selectedRunId = payload.run_id;
      }
      if (state.runInProgress && state.selectedRunId === payload.run_id) {
        stopRunProgressHint();
      }
      if (state.selectedRunId === payload.run_id) {
        taskStatusEl.textContent = run?.summary?.status === "failed"
          ? (run.summary.error || "任务失败")
          : "任务执行完成，结果已更新。";
      }
      render();
    }
  }
}

function shouldAutoSelectRun() {
  return state.runInProgress || !state.selectedRunId;
}

function createPlaceholderSummary() {
  return {
    workflow: state.portal?.workflow?.name || "",
    status: "running",
    error: "",
    started_at: "",
    finished_at: "",
    vars: {},
    assets: {},
    node_results: {},
    execution_order: [],
    stats: {
      total_nodes: Number(state.portal?.workflow_meta?.node_count || 0),
      executed_nodes: 0,
      succeeded_nodes: 0,
      failed_nodes: 0,
      skipped_nodes: 0,
    },
  };
}

function ensureRun(runID, summary = null) {
  const runs = state.portal?.recent_runs || [];
  const existing = runs.find((run) => run.id === runID);
  if (existing) {
    if (summary) {
      existing.summary = summary;
    }
    return existing;
  }

  const run = {
    id: runID,
    summary: summary || createPlaceholderSummary(),
  };
  return mergeRun(run);
}

function mergeNodeIntoRun(run, node) {
  if (!run || !node) {
    return;
  }
  if (!run.summary) {
    run.summary = createPlaceholderSummary();
  }
  if (!run.summary.node_results) {
    run.summary.node_results = {};
  }
  if (!Array.isArray(run.summary.execution_order)) {
    run.summary.execution_order = [];
  }
  run.summary.node_results[node.node_id] = node;
  if (!run.summary.execution_order.includes(node.node_id)) {
    run.summary.execution_order.push(node.node_id);
  }
}

function nodeStatusMessage(node) {
  if (!node?.node_id) {
    return "节点状态已更新。";
  }
  if (node.status === "failed") {
    return `节点 ${node.node_id} 失败：${node.error || "未知错误"}`;
  }
  if (node.status === "skipped") {
    return `节点 ${node.node_id} 已跳过：${node.skip_reason || "条件未满足"}`;
  }
  if (node.status === "succeeded") {
    return `节点 ${node.node_id} 已完成，继续执行后续步骤。`;
  }
  return `节点 ${node.node_id} 状态已更新。`;
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
  if (status === "running") {
    runStatusEl.className = "status-pill";
    runStatusEl.textContent = `运行中 · ${formatTime(run.summary.started_at)}`;
    return;
  }
  runStatusEl.className = `status-pill ${status === "succeeded" ? "success" : status === "failed" ? "error" : ""}`;
  runStatusEl.textContent = `${statusLabel(status)} · ${formatTime(run.summary.finished_at || run.summary.started_at)}`;
}

function renderTaskPanel(run) {
  renderChainEditor();
  renderChainLayout();
  renderChainPreview(run);
  renderReconSummary(run);
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
        <input type="checkbox" data-chain-field="run_asset_seed" ${state.chain.run_asset_seed ? "checked" : ""}>
        资产种子
      </label>
      <p class="item-meta">固定当前任务 scope、host 和初始入口，给资产图谱和后续执行器留统一入口。</p>
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_host_discovery" ${state.chain.run_host_discovery ? "checked" : ""}>
        主机发现阶段
      </label>
      <p class="item-meta">控制是否进入主机发现阶段。当前默认使用 nmap 快速扫描记录端口和服务。</p>
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nmap_quick" ${state.chain.run_nmap_quick ? "checked" : ""}>
        Nmap 主机发现
      </label>
      <input class="chain-args" type="text" data-chain-field="nmap_quick_args" value="${escapeHTML(state.chain.nmap_quick_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_entry_discovery" ${state.chain.run_entry_discovery ? "checked" : ""}>
        入口发现阶段
      </label>
      <p class="item-meta">控制是否进入入口发现阶段。当前用 fscan 提炼可疑路径、标题和跳转。</p>
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_fscan_web" ${state.chain.run_fscan_web ? "checked" : ""}>
        fscan 入口发现
      </label>
      <input class="chain-args" type="text" data-chain-field="fscan_args" value="${escapeHTML(state.chain.fscan_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">fscan 路径</label>
      <input class="chain-args" type="text" data-chain-field="fscan_binary_path" value="${escapeHTML(state.chain.fscan_binary_path)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nuclei_fingerprint" ${state.chain.run_nuclei_fingerprint ? "checked" : ""}>
        nuclei 指纹发现
      </label>
      <input class="chain-args" type="text" data-chain-field="nuclei_fingerprint_templates" value="${escapeHTML(state.chain.nuclei_fingerprint_templates)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">nuclei 指纹参数</label>
      <input class="chain-args" type="text" data-chain-field="nuclei_fingerprint_args" value="${escapeHTML(state.chain.nuclei_fingerprint_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">nuclei 路径</label>
      <input class="chain-args" type="text" data-chain-field="nuclei_binary_path" value="${escapeHTML(state.chain.nuclei_binary_path)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_nuclei_cve" ${state.chain.run_nuclei_cve ? "checked" : ""}>
        nuclei CVE 模板
      </label>
      <input class="chain-args" type="text" data-chain-field="nuclei_cve_templates" value="${escapeHTML(state.chain.nuclei_cve_templates)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">nuclei CVE 参数</label>
      <input class="chain-args" type="text" data-chain-field="nuclei_cve_args" value="${escapeHTML(state.chain.nuclei_cve_args)}">
    </article>
    <article class="item chain-row">
      <label class="chain-toggle">
        <input type="checkbox" data-chain-field="run_executor_placeholder" ${state.chain.run_executor_placeholder ? "checked" : ""}>
        执行器占坑
      </label>
      <input class="chain-args" type="text" data-chain-field="executor_name" value="${escapeHTML(state.chain.executor_name)}">
    </article>
  `;
}

function renderChainPreview(run) {
  const target = String(targetInput?.value || "").trim();
  const targetDetails = parseTargetDetails(target || state.portal?.workflow?.global_vars?.target || "target.local");
  const url = targetDetails.url;
  const host = targetDetails.host;
  const hostport = targetDetails.hostport || host || url;
  const origin = targetDetails.origin || url;
  const nmapPath = toolBinaryPath("nmap_scan");
  const fscanPath = resolveChainBinaryPath("fscan_binary_path", "fscan_scan");
  const nucleiPath = resolveChainBinaryPath("nuclei_binary_path", "nuclei_scan");
  const outputSlug = sanitizeTarget(hostport);
  const outputUnix = `<运行时自动生成>/output/${outputSlug}`;
  const outputWindows = `<运行时自动生成>\\output\\${outputSlug}`;

  const preview = [];
  if (state.chain.run_asset_seed) {
    preview.push({
      nodeID: "asset_seed",
      name: "asset_seed",
      executor: "local",
      output: "(stdout only)",
      cmd: [
        "初始化资产种子",
        `scope -> ${url}`,
        `host -> ${host || "<target_host>"}`,
        `origin -> ${origin}`,
        `entry -> ${url}`,
      ].join("\n"),
    });
  }
  if (state.chain.run_host_discovery && state.chain.run_nmap_quick) {
    preview.push({
      nodeID: "host_discovery",
      name: "host_discovery",
      executor: "local",
      output: `${outputWindows}\\host-discovery.txt`,
      cmd: `"${nmapPath}" ${state.chain.nmap_quick_args} -oN "${outputWindows}\\host-discovery.txt" ${host || "<target_host>"}`,
    });
  }
  if (state.chain.run_entry_discovery && state.chain.run_fscan_web) {
    preview.push({
      nodeID: "entry_discovery",
      name: "entry_discovery",
      executor: "local",
      output: `${outputWindows}\\entry-discovery.txt`,
      cmd: `"${fscanPath}" -u ${url} ${state.chain.fscan_args} -o "${outputWindows}\\entry-discovery.txt"`,
    });
  }
  if (state.chain.run_nuclei_fingerprint) {
    preview.push({
      nodeID: "nuclei_fingerprint",
      name: "nuclei_fingerprint",
      executor: "local",
      output: `${outputWindows}\\nuclei-fingerprint.jsonl`,
      cmd: `"${nucleiPath}" -u ${url} -t "${state.chain.nuclei_fingerprint_templates}" ${state.chain.nuclei_fingerprint_args} -o "${outputWindows}\\nuclei-fingerprint.jsonl"`,
    });
  }
  if (state.chain.run_nuclei_cve) {
    preview.push({
      nodeID: "nuclei_cve",
      name: "nuclei_cve",
      executor: "local",
      output: `${outputWindows}\\nuclei-cve.jsonl`,
      cmd: `"${nucleiPath}" -u ${url} -t "${state.chain.nuclei_cve_templates}" ${state.chain.nuclei_cve_args} -o "${outputWindows}\\nuclei-cve.jsonl"`,
    });
  }
  if (state.chain.run_executor_placeholder) {
    preview.push({
      nodeID: "executor_placeholder",
      name: "executor_placeholder",
      executor: "local",
      output: "(stdout only)",
      cmd: [
        "预留后续自定义执行器接入",
        `executor -> ${state.chain.executor_name || "custom-executor"}`,
        `target -> ${url}`,
        `entry -> ${url}`,
      ].join("\n"),
    });
  }

  if (preview.length === 0) {
    chainPreviewEl.innerHTML = `<div class="empty">当前没有启用任何工具节点。</div>`;
    return;
  }

  const activeRun = run || currentRun();
  const hasLiveRunningNode = preview.some((item) => activeRun?.summary?.node_results?.[item.nodeID]?.status === "running");
  let unresolvedIndex = state.runInProgress
    && !hasLiveRunningNode
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
        <p class="item-meta">执行器：${escapeHTML(item.executor || "local")}</p>
        <p class="item-meta">输出文件：${escapeHTML(item.output || outputUnix)}</p>
        <p class="item-meta">阶段说明：${escapeHTML(stage.hint)}</p>
        <pre class="cmd">${escapeHTML(previewCommand(item))}</pre>
      </article>
    `;
    })
    .join("");
}

function chainNodeStage(nodeID, run, runInProgress, unresolvedIndex, currentIndex) {
  const node = run?.summary?.node_results?.[nodeID];
  if (node?.status === "running") {
    return { className: "running", label: "运行中", hint: "节点正在执行，等待原始输出与解析结果回填。" };
  }
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
      hint: "任务正在执行中；如果首个实时事件尚未到达，这里会先展示估算阶段。",
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

function toolBinaryPath(toolName, vars = state.portal?.workflow?.global_vars || {}) {
  const tool = (state.portal?.tools || []).find((item) => item.name === toolName);
  const binaryVar = String(tool?.metadata?.binary_var || "");
  const varsValue = binaryVar ? String(vars?.[binaryVar] || "").trim() : "";
  return varsValue || String(tool?.metadata?.binary_path || tool?.name || toolName);
}

function resolveChainBinaryPath(field, toolName) {
  const override = String(state.chain?.[field] || "").trim();
  return override || toolBinaryPath(toolName);
}

function orderedNodeIds(run) {
  if (!run?.summary) {
    return [];
  }
  const executionOrder = Array.isArray(run.summary.execution_order) ? run.summary.execution_order : [];
  const nodeResults = run.summary.node_results || {};
  const extras = Object.keys(nodeResults).filter((nodeID) => !executionOrder.includes(nodeID));
  return [...executionOrder, ...extras];
}

function renderToolCalls(run) {
  if (!run) {
    toolCallsEl.innerHTML = `<div class="empty">还没有任务运行记录。</div>`;
    return;
  }

  const nodeIds = orderedNodeIds(run);
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
          <pre class="cmd">${escapeHTML(describeNodeExecution(node))}</pre>
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

  const nodeIds = orderedNodeIds(run);
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

function renderHTTPFingerprintSummary(run) {
  if (!httpFingerprintSummaryEl) {
    return;
  }
  if (!run) {
    httpFingerprintSummaryEl.innerHTML = `<div class="empty">运行 HTTP Probe 后，这里会汇总产品、框架、跳转和关键页面状态。</div>`;
    return;
  }

  const web = asObject(run.summary?.assets?.web);
  const fingerprints = asObject(web.fingerprints);
  const metadata = asObject(web.metadata);
  const branding = asObject(web.branding);

  const products = extractRecordFieldValues(fingerprints.products, "product");
  const frameworks = extractRecordFieldValues(fingerprints.frameworks, "framework");
  const editions = extractRecordFieldValues(fingerprints.dify_edition, "edition");
  const poweredBy = extractRecordFieldValues(web.powered_by, "value");
  const redirects = extractRecordFieldValues(web.redirects, "location");
  const statuses = extractRecordFieldValues(web.http_statuses, "status_code");
  const titles = extractRecordFieldValues(web.page_titles, "title");
  const consoleAPI = extractRecordFieldValues(fingerprints.dify_console_api, "api_prefix");
  const publicAPI = extractRecordFieldValues(fingerprints.dify_public_api, "api_prefix");
  const marketplaceAPI = extractRecordFieldValues(fingerprints.dify_marketplace_api, "api_prefix");
  const marketplaceURL = extractRecordFieldValues(fingerprints.dify_marketplace_url, "url_prefix");
  const robots = extractRecordFieldValues(metadata.robots, "directive");
  const preloadAssets = extractRecordFieldValues(branding.preloaded_assets, "asset");
  const staticAssets = extractRecordFieldValues(web.static_assets, "asset");
  const reactRSC = extractRecordFieldValues(fingerprints.react_rsc, "feature");
  const appRouterAssets = uniqueStrings(staticAssets.filter((asset) => asset.includes("/_next/static/chunks/app/")));
  const nucleiFindings = extractNucleiFindingRows(run.summary?.assets?.discovery?.nuclei_findings);
  const nucleiCVEs = extractNucleiFindingRows(run.summary?.assets?.security?.cve_findings);
  const probeCards = buildHTTPProbeCards(run);
  const advisoryCard = renderReactRSCAdvisoryCard({
    frameworks,
    reactRSC,
    appRouterAssets,
  });

  const cards = [
    renderSummaryCard("产品画像", [
      { label: "产品", values: products, mode: "chips" },
      { label: "框架", values: frameworks, mode: "chips" },
      { label: "部署版本", values: editions, mode: "chips" },
      { label: "X-Powered-By", values: poweredBy, mode: "chips" },
      { label: "Robots", values: robots, mode: "chips" },
    ]),
    renderSummaryCard("关键端点", [
      { label: "控制台 API", values: consoleAPI, mode: "text" },
      { label: "公共 API", values: publicAPI, mode: "text" },
      { label: "市场 API", values: marketplaceAPI, mode: "text" },
      { label: "市场地址", values: marketplaceURL, mode: "text" },
      { label: "页面跳转", values: redirects, mode: "chips" },
    ]),
    renderSummaryCard("页面特征", [
      { label: "状态码", values: statuses, mode: "chips" },
      { label: "页面标题", values: titles, mode: "text" },
      { label: "预加载资源", values: preloadAssets.length > 0 ? [`${preloadAssets.length} 个`] : [], mode: "text" },
      { label: "静态资源", values: staticAssets.length > 0 ? [`${staticAssets.length} 个`] : [], mode: "text" },
    ]),
    renderNucleiFindingsCard(nucleiFindings, nucleiCVEs),
    advisoryCard,
  ].filter(Boolean);

  if (cards.length === 0 && probeCards.length === 0) {
    const hint = run.summary?.status === "running"
      ? "任务运行中，HTTP 指纹会随着节点完成逐步回填。"
      : "当前运行尚未产出可归纳的 HTTP 指纹。";
    httpFingerprintSummaryEl.innerHTML = `<div class="empty">${escapeHTML(hint)}</div>`;
    return;
  }

  httpFingerprintSummaryEl.innerHTML = `
    ${cards.length > 0 ? `<div class="summary-grid">${cards.join("")}</div>` : ""}
    ${probeCards.length > 0 ? `
      <section class="summary-section">
        <div class="summary-section-head">
          <h3>关键页面探测</h3>
          <p>按路径展示每个 HTTP Probe 节点的执行状态、响应码和标题。</p>
        </div>
        <div class="probe-grid">${probeCards.join("")}</div>
      </section>
    ` : ""}
  `;
}

function renderReconSummary(run) {
  if (!reconSummaryEl) {
    return;
  }
  if (!run) {
    reconSummaryEl.innerHTML = `<div class="empty">运行任务后，这里会按主机发现、入口发现、漏洞发现和执行器占坑汇总结果。</div>`;
    return;
  }

  const discovery = asObject(run.summary?.assets?.discovery);
  const network = asObject(run.summary?.assets?.network);
  const web = asObject(run.summary?.assets?.web);
  const security = asObject(run.summary?.assets?.security);
  const actions = asObject(run.summary?.assets?.actions);
  const fingerprints = asObject(web.fingerprints);

  const scopeTargets = extractRecordFieldValues(discovery.scope_targets, "target");
  const discoveredHosts = normalizeHostRows(discovery.hosts);
  const hostLabels = uniqueStrings(discoveredHosts.flatMap((item) => [item.hostport, item.host]));
  const openPorts = buildOpenPortLabels(network.open_ports);
  const weakSignals = buildWeakSignalLabels(security);
  const products = extractRecordFieldValues(fingerprints.products, "product");
  const frameworks = extractRecordFieldValues(fingerprints.frameworks, "framework");
  const editions = extractRecordFieldValues(fingerprints.dify_edition, "edition");
  const entries = normalizeEntryRows(discovery.entries, web);
  const entryLabels = uniqueStrings(entries.map((item) => item.url));
  const redirects = extractRecordFieldValues(web.redirects, "location");
  const staticAssets = extractRecordFieldValues(web.static_assets, "asset");
  const reactRSC = extractRecordFieldValues(fingerprints.react_rsc, "feature");
  const appRouterAssets = uniqueStrings(staticAssets.filter((asset) => asset.includes("/_next/static/chunks/app/")));
  const nucleiFindings = extractNucleiFindingRows(discovery.nuclei_findings);
  const nucleiCVEs = extractNucleiFindingRows(security.cve_findings);
  const vulnFindings = normalizeVulnRows(security.vulnerabilities);
  const pendingExecutions = normalizePendingExecutions(actions.pending_exploitation);
  const advisoryCard = renderReactRSCAdvisoryCard({
    frameworks,
    reactRSC,
    appRouterAssets,
  });

  const cards = [
    renderSummaryCard("主机发现", [
      { label: "Scope", values: scopeTargets, mode: "text" },
      { label: "主机", values: hostLabels, mode: "chips" },
      { label: "开放端口", values: openPorts, mode: "chips" },
      { label: "弱配置线索", values: weakSignals, mode: "text" },
    ]),
    renderSummaryCard("入口发现", [
      { label: "候选入口", values: entryLabels, mode: "text" },
      { label: "产品", values: products, mode: "chips" },
      { label: "框架", values: frameworks, mode: "chips" },
      { label: "部署线索", values: editions, mode: "chips" },
      { label: "页面跳转", values: redirects, mode: "chips" },
    ]),
    renderSummaryCard("漏洞发现", [
      { label: "指纹模板命中", values: nucleiFindings.map((item) => `${item.templateID} [${item.severity}]`), mode: "chips" },
      { label: "CVE 模板命中", values: nucleiCVEs.map((item) => `${item.templateID} [${item.severity}]`), mode: "chips" },
      { label: "其他漏洞线索", values: vulnFindings.map((item) => item.detail), mode: "text" },
    ]),
    renderSummaryCard("执行器占坑", [
      { label: "待执行动作", values: pendingExecutions.map((item) => `${item.executor} -> ${item.entry}`), mode: "text" },
      { label: "当前状态", values: pendingExecutions.map((item) => item.status), mode: "chips" },
      { label: "说明", values: ["当前只沉淀发现结果和待执行动作，不在框架内直接利用。"], mode: "text" },
    ]),
    advisoryCard,
  ].filter(Boolean);

  const detailSections = [
    renderDetailSection(
      "主机资产",
      "给后续网络资产测绘图使用，汇总 scope、主机和开放端口。",
      buildHostDetailCards(scopeTargets, discoveredHosts, openPorts, weakSignals),
    ),
    renderDetailSection(
      "入口清单",
      "按扫描结果收集候选入口，后续可把每个入口交给漏洞模板或你的执行器。",
      buildEntryDetailCards(entries),
    ),
    renderDetailSection(
      "漏洞命中",
      "当前阶段只做到发现，后续利用交给独立执行器。",
      buildVulnerabilityCards(nucleiFindings, nucleiCVEs, vulnFindings),
    ),
    renderDetailSection(
      "执行器占坑",
      "为你自己的利用执行器预留输入契约。",
      buildExecutorDetailCards(pendingExecutions),
    ),
  ].filter(Boolean);

  if (cards.length === 0 && detailSections.length === 0) {
    const hint = run.summary?.status === "running"
      ? "任务运行中，发现结果会随着节点完成逐步回填。"
      : "当前运行还没有沉淀出可展示的发现结果。";
    reconSummaryEl.innerHTML = `<div class="empty">${escapeHTML(hint)}</div>`;
    return;
  }

  reconSummaryEl.innerHTML = `
    ${cards.length > 0 ? `<div class="summary-grid">${cards.join("")}</div>` : ""}
    ${detailSections.join("")}
  `;
}

function renderDetailSection(title, description, cards) {
  if (!Array.isArray(cards) || cards.length === 0) {
    return "";
  }
  return `
    <section class="summary-section">
      <div class="summary-section-head">
        <h3>${escapeHTML(title)}</h3>
        <p>${escapeHTML(description)}</p>
      </div>
      <div class="detail-grid">${cards.join("")}</div>
    </section>
  `;
}

function buildHostDetailCards(scopeTargets, hosts, openPorts, weakSignals) {
  const cards = [];

  uniqueStrings(scopeTargets).forEach((target) => {
    cards.push(renderDetailCard("Scope", target, [`target=${target}`]));
  });
  normalizeHostRows(hosts).forEach((item) => {
    const lines = [];
    if (item.host) {
      lines.push(`host=${item.host}`);
    }
    if (item.hostport) {
      lines.push(`hostport=${item.hostport}`);
    }
    if (item.origin) {
      lines.push(`origin=${item.origin}`);
    }
    cards.push(renderDetailCard("Host", item.hostport || item.host, lines));
  });
  if (openPorts.length > 0) {
    cards.push(renderDetailCard("Open Ports", `${openPorts.length} 项`, openPorts));
  }
  if (weakSignals.length > 0) {
    cards.push(renderDetailCard("Weak Signals", `${weakSignals.length} 项`, weakSignals));
  }
  return cards;
}

function buildEntryDetailCards(entries) {
  return normalizeEntryRows(entries)
    .map((item) => {
      const lines = [];
      if (item.title) {
        lines.push(`title=${item.title}`);
      }
      if (item.statusCode) {
        lines.push(`status=${item.statusCode}`);
      }
      if (item.redirect) {
        lines.push(`redirect=${item.redirect}`);
      }
      if (item.source) {
        lines.push(`source=${item.source}`);
      }
      return renderDetailCard("Entry", item.url, lines);
    });
}

function buildVulnerabilityCards(nucleiFindings, nucleiCVEs, vulnFindings) {
  const cards = [];

  nucleiFindings.forEach((item) => {
    cards.push(renderDetailCard(item.templateID, item.target || item.name, [
      `severity=${item.severity}`,
      item.name || "",
    ]));
  });
  nucleiCVEs.forEach((item) => {
    cards.push(renderDetailCard(item.templateID, item.target || item.name, [
      `severity=${item.severity}`,
      item.name || "",
    ], "is-risk"));
  });
  normalizeVulnRows(vulnFindings).forEach((item) => {
    cards.push(renderDetailCard("scanner finding", item.target || item.detail, [
      `status=${item.status}`,
      item.detail || "",
    ], "is-risk"));
  });

  return cards;
}

function buildExecutorDetailCards(pendingExecutions) {
  return normalizePendingExecutions(pendingExecutions)
    .map((item) => renderDetailCard(item.executor, item.entry || item.target, [
      `target=${item.target}`,
      `source=${item.source}`,
      `status=${item.status}`,
    ]));
}

function renderDetailCard(kicker, title, lines, extraClass = "") {
  const detailLines = uniqueStrings(lines).filter(Boolean);
  if (!normalizeString(title) && detailLines.length === 0) {
    return "";
  }
  return `
    <article class="detail-card ${escapeHTML(extraClass)}">
      <p class="detail-kicker">${escapeHTML(kicker)}</p>
      <div class="detail-card-title">${escapeHTML(title || kicker)}</div>
      <div class="detail-card-body">
        ${detailLines.map((line) => `<div class="detail-line">${escapeHTML(line)}</div>`).join("")}
      </div>
    </article>
  `;
}

function renderSummaryCard(title, items) {
  const rows = items
    .map((item) => renderSummaryRow(item.label, item.values, item.mode))
    .filter(Boolean);
  if (rows.length === 0) {
    return "";
  }
  return `
    <article class="summary-card">
      <p class="summary-kicker">${escapeHTML(title)}</p>
      <div class="summary-list">${rows.join("")}</div>
    </article>
  `;
}

function renderReactRSCAdvisoryCard({ frameworks, reactRSC, appRouterAssets }) {
  const hasNextJS = frameworks.includes("Next.js");
  const hasRSCSignal = reactRSC.length > 0 || appRouterAssets.length > 0;
  if (!hasNextJS && !hasRSCSignal) {
    return "";
  }

  const indicators = [];
  if (hasNextJS) {
    indicators.push("Next.js");
  }
  if (reactRSC.length > 0) {
    indicators.push("Flight stream (__next_f.push)");
  }
  if (appRouterAssets.length > 0) {
    indicators.push(`App Router chunk ${appRouterAssets.length} 个`);
  }

  return renderSummaryCard("安全提示", [
    { label: "RSC 指示器", values: indicators, mode: "chips" },
    {
      label: "CVE-2025-55182",
      values: [
        "若部署版本命中 react-server-dom-* 19.0.0 / 19.1.0 / 19.1.1 / 19.2.0，则属于 React Server Components 预认证 RCE 影响范围。",
      ],
      mode: "text",
    },
    {
      label: "修复版本",
      values: [
        "首轮修复：19.0.1 / 19.1.2 / 19.2.1",
        "按 React 2026-01-26 后续公告，建议至少升级到 19.0.4 / 19.1.5 / 19.2.4。",
      ],
      mode: "text",
    },
    {
      label: "当前判断",
      values: [
        "当前只识别到 Next.js / RSC 暴露迹象，尚未识别 React 具体版本；需要结合 package-lock、镜像清单或 SBOM 进一步确认。",
      ],
      mode: "text",
    },
  ]);
}

function renderNucleiFindingsCard(findings, cves) {
  if (findings.length === 0 && cves.length === 0) {
    return "";
  }

  const findingLabels = findings.map((item) => `${item.templateID} [${item.severity}]`);
  const cveLabels = cves.map((item) => `${item.templateID} [${item.severity}]`);

  return renderSummaryCard("Nuclei 发现", [
    { label: "指纹模板命中", values: findingLabels, mode: "chips" },
    { label: "CVE 模板命中", values: cveLabels, mode: "chips" },
    {
      label: "架构说明",
      values: [
        "当前框架支持把发现能力收敛到 nuclei 模板目录，后续新增指纹或 CVE 主要通过编写模板完成。",
      ],
      mode: "text",
    },
  ]);
}

function renderSummaryRow(label, values, mode = "chips") {
  const normalizedValues = uniqueStrings(Array.isArray(values) ? values : [values]);
  if (normalizedValues.length === 0) {
    return "";
  }

  const renderedValue = mode === "text"
    ? `<div class="summary-value">${normalizedValues.map((value) => escapeHTML(value)).join("<br>")}</div>`
    : `<div class="summary-chips">${normalizedValues.map((value) => `<span class="summary-chip">${escapeHTML(value)}</span>`).join("")}</div>`;

  return `
    <div class="summary-row">
      <div class="summary-label">${escapeHTML(label)}</div>
      ${renderedValue}
    </div>
  `;
}

function buildHTTPProbeCards(run) {
  const vars = asObject(run.summary?.vars);
  const nodeResults = asObject(run.summary?.node_results);

  return HTTP_PROBE_NODES
    .filter((config) => nodeResults[config.nodeID] || toBool(vars[config.varKey], true))
    .map((config) => {
      const node = nodeResults[config.nodeID];
      const enabled = toBool(vars[config.varKey], true);
      const recordTitles = extractNodeRecordValues(node, "html_title", "title");
      const recordRedirects = extractNodeRecordValues(node, "redirect_location", "location");
      const statusCode = extractNodeStatusCode(node);
      const detailRows = [];

      if (!enabled && !node) {
        detailRows.push(renderProbeDetail("状态", "本次运行未启用"));
      } else {
        if (statusCode) {
          detailRows.push(renderProbeDetail("响应码", statusCode));
        }
        if (recordTitles.length > 0) {
          detailRows.push(renderProbeDetail("标题", recordTitles.join(" / ")));
        }
        if (recordRedirects.length > 0) {
          detailRows.push(renderProbeDetail("跳转", recordRedirects.join(" / ")));
        }
        if (node?.error) {
          detailRows.push(renderProbeDetail("错误", node.error));
        }
        if (detailRows.length === 0) {
          if (node?.status === "running") {
            detailRows.push(renderProbeDetail("状态", "节点运行中，等待结果回传"));
          } else if (!node) {
            detailRows.push(renderProbeDetail("状态", run.summary?.status === "running" ? "尚未开始" : "无返回数据"));
          } else {
            detailRows.push(renderProbeDetail("状态", nodeStatusMessage(node)));
          }
        }
      }

      const chipClass = !enabled && !node ? "skipped" : (node?.status || "");
      const chipLabel = !enabled && !node ? "未启用" : statusLabel(node?.status || "pending");
      return `
        <article class="probe-card ${escapeHTML(chipClass)}">
          <div class="probe-card-head">
            <div>
              <div class="probe-title">${escapeHTML(config.label)}</div>
              <p class="probe-path">GET ${escapeHTML(config.path)}</p>
            </div>
            <span class="chip ${escapeHTML(chipClass)}">${escapeHTML(chipLabel)}</span>
          </div>
          <div class="probe-details">${detailRows.join("")}</div>
        </article>
      `;
    });
}

function renderProbeDetail(label, value) {
  const text = normalizeString(value);
  if (!text) {
    return "";
  }
  return `
    <div class="probe-detail">
      <span class="probe-detail-label">${escapeHTML(label)}</span>
      <span class="probe-detail-value">${escapeHTML(text)}</span>
    </div>
  `;
}

function asObject(value) {
  return value && typeof value === "object" && !Array.isArray(value) ? value : {};
}

function extractRecordFieldValues(records, fieldName) {
  if (!Array.isArray(records)) {
    return [];
  }
  return uniqueStrings(records.map((entry) => {
    const record = asObject(entry);
    return normalizeString(record[fieldName]);
  }));
}

function extractNodeRecordValues(node, ruleName, fieldName) {
  if (!Array.isArray(node?.records)) {
    return [];
  }
  return uniqueStrings(node.records
    .filter((record) => record?.rule === ruleName)
    .map((record) => normalizeString(record?.fields?.[fieldName])));
}

function extractNodeStatusCode(node) {
  const raw = node?.metadata?.status_code;
  if (typeof raw === "number" && raw > 0) {
    return String(raw);
  }
  const value = normalizeString(raw);
  return value && value !== "0" ? value : "";
}

function previewCommand(item) {
  return item?.cmd || "(无命令)";
}

function describeNodeExecution(node) {
  if (node?.executor === "http") {
    const outputFiles = Array.isArray(node.metadata?.output_files) ? node.metadata.output_files : [];
    const fileLines = outputFiles
      .map((entry) => normalizeString(entry?.path))
      .filter(Boolean)
      .map((path) => `output -> ${path}`);
    const lines = [
      `HTTP GET ${normalizeString(node.metadata?.url) || normalizeString(node.inputs?.url) || "(unknown url)"}`,
    ];
    const statusCode = extractNodeStatusCode(node);
    if (statusCode) {
      lines.push(`status=${statusCode}`);
    }
    if (fileLines.length > 0) {
      lines.push(...fileLines);
    }
    return lines.join("\n");
  }
  return node?.rendered_command || "(无命令)";
}

function extractNucleiFindingRows(records) {
  if (!Array.isArray(records)) {
    return [];
  }
  return records
    .map((entry) => {
      const record = asObject(entry);
      return {
        templateID: normalizeString(record.template_id),
        name: normalizeString(record.name),
        severity: normalizeString(record.severity) || "unknown",
        target: normalizeString(record.target),
      };
    })
    .filter((item) => item.templateID);
}

function normalizeHostRows(records) {
  if (!Array.isArray(records)) {
    return [];
  }
  const result = [];
  const seen = new Set();
  records.forEach((entry) => {
    const record = asObject(entry);
    const host = normalizeString(record.host);
    const hostport = normalizeString(record.hostport);
    const origin = normalizeString(record.origin);
    const key = `${host}|${hostport}|${origin}`;
    if ((!host && !hostport && !origin) || seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push({ host, hostport, origin });
  });
  return result;
}

function normalizeEntryRows(records, web = {}) {
  const sources = [];
  if (Array.isArray(records)) {
    sources.push(...records.map((entry) => ({ ...asObject(entry), source: normalizeString(asObject(entry).source) || "discovery" })));
  }
  if (Array.isArray(web?.pages)) {
    sources.push(...web.pages.map((entry) => ({ ...asObject(entry), source: "page" })));
  }
  if (Array.isArray(web?.redirects)) {
    sources.push(...web.redirects.map((entry) => {
      const record = asObject(entry);
      return {
        url: normalizeString(record.redirect_url) || normalizeString(record.location),
        redirect: normalizeString(record.location) || normalizeString(record.redirect_url),
        source: "redirect",
      };
    }));
  }

  const result = [];
  const seen = new Set();
  sources.forEach((entry) => {
    const record = asObject(entry);
    const url = normalizeString(record.url);
    const title = normalizeString(record.title);
    const statusCode = normalizeString(record.status_code);
    const redirect = normalizeString(record.redirect) || normalizeString(record.redirect_url) || normalizeString(record.location);
    const source = normalizeString(record.source);
    const key = `${url}|${title}|${statusCode}|${redirect}|${source}`;
    if (!url || seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push({ url, title, statusCode, redirect, source });
  });
  return result;
}

function buildOpenPortLabels(records) {
  if (!Array.isArray(records)) {
    return [];
  }
  return uniqueStrings(records.map((entry) => {
    const record = asObject(entry);
    const port = normalizeString(record.port);
    const service = normalizeString(record.service);
    const version = normalizeString(record.version);
    return [port, service, version].filter(Boolean).join(" ");
  }));
}

function buildWeakSignalLabels(security) {
  const tlsVersions = extractRecordFieldValues(security.legacy_tls, "tls_version").map((value) => `TLSv${value}`);
  const weakCiphers = extractRecordFieldValues(security.weak_ciphers, "cipher");
  const headerFindings = extractRecordFieldValues(security.http_headers, "finding");
  return uniqueStrings([...tlsVersions, ...weakCiphers, ...headerFindings]);
}

function normalizeVulnRows(records) {
  if (!Array.isArray(records)) {
    return [];
  }
  const result = [];
  const seen = new Set();
  records.forEach((entry) => {
    const record = asObject(entry);
    const target = normalizeString(record.target);
    const status = normalizeString(record.status);
    const detail = normalizeString(record.detail);
    const key = `${target}|${status}|${detail}`;
    if ((!target && !detail) || seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push({ target, status, detail });
  });
  return result;
}

function normalizePendingExecutions(records) {
  if (!Array.isArray(records)) {
    return [];
  }
  const result = [];
  const seen = new Set();
  records.forEach((entry) => {
    const record = asObject(entry);
    const target = normalizeString(record.target);
    const entryURL = normalizeString(record.entry);
    const source = normalizeString(record.source);
    const status = normalizeString(record.status);
    const executor = normalizeString(record.executor);
    const key = `${target}|${entryURL}|${source}|${status}|${executor}`;
    if ((!target && !entryURL) || seen.has(key)) {
      return;
    }
    seen.add(key);
    result.push({ target, entry: entryURL, source, status, executor });
  });
  return result;
}

function uniqueStrings(values) {
  const result = [];
  const seen = new Set();
  values.forEach((value) => {
    const text = normalizeString(value);
    if (!text || seen.has(text)) {
      return;
    }
    seen.add(text);
    result.push(text);
  });
  return result;
}

function normalizeString(value) {
  return String(value ?? "").trim();
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
    taskStatusEl.textContent = `任务执行中，已等待 ${formatElapsed(elapsed)}。如果实时事件暂未返回，页面会先显示估算阶段。`;
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
          run_asset_seed: state.chain.run_asset_seed,
          run_host_discovery: state.chain.run_host_discovery,
          run_entry_discovery: state.chain.run_entry_discovery,
          run_nmap_quick: state.chain.run_nmap_quick,
          run_fscan_web: state.chain.run_fscan_web,
          run_nuclei_fingerprint: state.chain.run_nuclei_fingerprint,
          run_nuclei_cve: state.chain.run_nuclei_cve,
          run_executor_placeholder: state.chain.run_executor_placeholder,
          fscan_binary_path: resolveChainBinaryPath("fscan_binary_path", "fscan_scan"),
          nuclei_binary_path: resolveChainBinaryPath("nuclei_binary_path", "nuclei_scan"),
          nmap_quick_args: state.chain.nmap_quick_args,
          fscan_args: state.chain.fscan_args,
          nuclei_fingerprint_templates: state.chain.nuclei_fingerprint_templates,
          nuclei_cve_templates: state.chain.nuclei_cve_templates,
          nuclei_fingerprint_args: state.chain.nuclei_fingerprint_args,
          nuclei_cve_args: state.chain.nuclei_cve_args,
          executor_name: state.chain.executor_name,
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
    return null;
  }
  state.portal.latest_run = run;
  const existing = state.portal.recent_runs || [];
  state.portal.recent_runs = [run, ...existing.filter((item) => item.id !== run.id)].slice(0, 20);
  return run;
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
  return parseTargetDetails(raw).host;
}

function normalizeTargetURL(raw) {
  return parseTargetDetails(raw).url;
}

function parseTargetDetails(raw) {
  const input = String(raw || "").trim();
  if (!input) {
    return { raw: "", url: "", host: "", hostport: "", origin: "" };
  }
  try {
    const parsed = new URL(input.includes("://") ? input : `https://${input}`);
    return {
      raw: input,
      url: parsed.toString(),
      host: parsed.hostname || input,
      hostport: parsed.host || parsed.hostname || input,
      origin: parsed.origin || "",
    };
  } catch (_error) {
    const url = input.includes("://") ? input : `https://${input}`;
    return { raw: input, url, host: input, hostport: input, origin: url };
  }
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
  if (
    field === "run_asset_seed"
    || field === "run_host_discovery"
    || field === "run_entry_discovery"
    || field === "run_nmap_quick"
    || field === "run_fscan_web"
    || field === "run_nuclei_fingerprint"
    || field === "run_nuclei_cve"
    || field === "run_executor_placeholder"
  ) {
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

loadState().then(() => {
  connectEventStream();
}).catch((error) => {
  runStatusEl.className = "status-pill error";
  runStatusEl.textContent = error.message;
  taskStatusEl.textContent = error.message;
});
