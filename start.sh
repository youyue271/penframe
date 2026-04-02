#!/usr/bin/env bash
#
# Penframe 一键启动脚本
# 同时启动：Go API Server / Python Exp Service / Vue Dev Server
#
# 用法：
#   ./start.sh                          # 默认启动全部
#   ./start.sh --target https://x:3000  # 指定默认目标
#   ./start.sh --no-exp                 # 不启动 Exp 服务
#   ./start.sh --no-web                 # 不启动 Vue 前端
#

set -e

ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT"

# ---------- 参数解析 ----------
TARGET=""
WORKFLOW="examples/live/workflow.yaml"
TOOLS="examples/mvp/tools.yaml"
API_PORT=8080
EXP_PORT=8787
WEB_PORT=5173
SKIP_EXP=false
SKIP_WEB=false

while [[ $# -gt 0 ]]; do
  case $1 in
    --target)      TARGET="$2"; shift 2 ;;
    --workflow)    WORKFLOW="$2"; shift 2 ;;
    --tools)       TOOLS="$2"; shift 2 ;;
    --api-port)    API_PORT="$2"; shift 2 ;;
    --exp-port)    EXP_PORT="$2"; shift 2 ;;
    --web-port)    WEB_PORT="$2"; shift 2 ;;
    --no-exp)      SKIP_EXP=true; shift ;;
    --no-web)      SKIP_WEB=true; shift ;;
    -h|--help)
      echo "Usage: $0 [options]"
      echo "  --target URL       Default scan target"
      echo "  --workflow PATH    Workflow YAML (default: examples/live/workflow.yaml)"
      echo "  --tools PATH       Tools YAML (default: examples/mvp/tools.yaml)"
      echo "  --api-port PORT    Go API port (default: 8080)"
      echo "  --exp-port PORT    Python Exp port (default: 8787)"
      echo "  --web-port PORT    Vue dev port (default: 5173)"
      echo "  --no-exp           Skip Python Exp service"
      echo "  --no-web           Skip Vue frontend"
      exit 0
      ;;
    *) echo "Unknown option: $1"; exit 1 ;;
  esac
done

# ---------- 颜色 ----------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; exit 1; }
venv_ready() { [[ -x "$1/bin/python" ]] && "$1/bin/python" -c "import sys" >/dev/null 2>&1; }
ensure_writable_dir() {
  local dir="$1"
  local probe
  mkdir -p "$dir" 2>/dev/null || return 1
  probe="$dir/.penframe-write-test.$$"
  if ! : >"$probe" 2>/dev/null; then
    return 1
  fi
  rm -f "$probe" 2>/dev/null || true
  return 0
}
build_portal_binary() {
  local output_path="$1"
  go build -o "$output_path" ./cmd/portal 2>&1
}
ensure_started() {
  local pid="$1"
  local name="$2"
  local port="$3"
  local checks="${4:-2}"
  local i
  for ((i = 0; i < checks; i++)); do
    if ! kill -0 "$pid" 2>/dev/null; then
      wait "$pid" 2>/dev/null || true
      fail "$name failed to start; port $port may already be in use"
    fi
    sleep 1
  done
}

# ---------- 前置检查 ----------
info "Checking dependencies..."

command -v go      >/dev/null 2>&1 || fail "go not found"
command -v python3 >/dev/null 2>&1 || fail "python3 not found"
command -v node    >/dev/null 2>&1 || fail "node not found"
command -v npm     >/dev/null 2>&1 || fail "npm not found"

ok "go:      $(go version | awk '{print $3}')"
ok "python3: $(python3 --version 2>&1 | awk '{print $2}')"
ok "node:    $(node --version)"

# ---------- Go 构建 ----------
info "Building Go API server..."
PORTAL_BIN_DIR_DEFAULT="$ROOT/bin"
PORTAL_BIN_DIR_FALLBACK="/tmp/penframe/bin"
PORTAL_BIN_DIR="${PENFRAME_BIN_DIR:-$PORTAL_BIN_DIR_DEFAULT}"
BUILD_OUTPUT=""

if [[ -z "${PENFRAME_BIN_DIR:-}" && "$ROOT" == /mnt/* ]]; then
  PORTAL_BIN_DIR="$PORTAL_BIN_DIR_FALLBACK"
fi

if ! ensure_writable_dir "$PORTAL_BIN_DIR"; then
  if [[ "$PORTAL_BIN_DIR" != "$PORTAL_BIN_DIR_FALLBACK" ]]; then
    warn "Build output directory is not writable: $PORTAL_BIN_DIR; falling back to $PORTAL_BIN_DIR_FALLBACK"
    PORTAL_BIN_DIR="$PORTAL_BIN_DIR_FALLBACK"
  fi
  ensure_writable_dir "$PORTAL_BIN_DIR" || fail "Failed to prepare writable build directory"
fi

PORTAL_BIN="$PORTAL_BIN_DIR/portal"
if ! BUILD_OUTPUT="$(build_portal_binary "$PORTAL_BIN")"; then
  if [[ -n "$BUILD_OUTPUT" ]]; then
    echo "$BUILD_OUTPUT"
  fi
  if [[ -z "${PENFRAME_BIN_DIR:-}" && "$PORTAL_BIN_DIR" != "$PORTAL_BIN_DIR_FALLBACK" ]]; then
    warn "Go build failed in $PORTAL_BIN_DIR; retrying with $PORTAL_BIN_DIR_FALLBACK"
    PORTAL_BIN_DIR="$PORTAL_BIN_DIR_FALLBACK"
    ensure_writable_dir "$PORTAL_BIN_DIR" || fail "Failed to prepare writable build directory"
    PORTAL_BIN="$PORTAL_BIN_DIR/portal"
    BUILD_OUTPUT="$(build_portal_binary "$PORTAL_BIN")" || {
      [[ -n "$BUILD_OUTPUT" ]] && echo "$BUILD_OUTPUT"
      fail "Go build failed"
    }
  else
    fail "Go build failed"
  fi
fi
ok "Go binary: $PORTAL_BIN"

# ---------- Python 依赖 ----------
EXP_VENV_DEFAULT="$ROOT/examples/exp/.venv"
EXP_VENV_FALLBACK="/tmp/penframe-exp/.venv"
EXP_VENV="${PENFRAME_EXP_VENV:-$EXP_VENV_DEFAULT}"
if [[ "$SKIP_EXP" != "true" ]]; then
  if ! venv_ready "$EXP_VENV"; then
    if [[ -e "$EXP_VENV" ]]; then
      warn "Python venv exists but is incomplete: $EXP_VENV"
    fi

    info "Creating Python venv at $EXP_VENV ..."
    if ! python3 -m venv "$EXP_VENV" 2>/tmp/penframe-venv.err; then
      if [[ "$EXP_VENV" != "$EXP_VENV_FALLBACK" ]]; then
        warn "Failed to create venv at $EXP_VENV; falling back to $EXP_VENV_FALLBACK"
        EXP_VENV="$EXP_VENV_FALLBACK"
        if venv_ready "$EXP_VENV"; then
          info "Reusing existing fallback venv at $EXP_VENV"
        else
          if [[ -e "$EXP_VENV" ]]; then
            warn "Fallback venv exists but is incomplete: $EXP_VENV"
          fi
          mkdir -p "$(dirname "$EXP_VENV")" || fail "Failed to prepare fallback venv directory"
          python3 -m venv --copies "$EXP_VENV" || fail "Failed to create fallback venv"
        fi
      else
        fail "Failed to create venv"
      fi
    fi
  fi
  ok "Python venv: $EXP_VENV"

  info "Checking Python dependencies..."
  if ! "$EXP_VENV/bin/python" -c "import fastapi, uvicorn, httpx, pydantic" 2>/dev/null; then
    info "Installing Python dependencies into venv..."
    "$EXP_VENV/bin/pip" install -q -r examples/exp/requirements.txt 2>&1 || fail "pip install failed"
  fi
  ok "Python dependencies ready"
fi

# ---------- Vue 依赖 ----------
WEB_PIDS=()
WEB_VITE_BIN="$ROOT/web/node_modules/vite/bin/vite.js"
if [[ "$SKIP_WEB" != "true" ]]; then
  if [[ ! -f "$WEB_VITE_BIN" ]]; then
    info "Installing Vue dependencies (first time)..."
    (cd web && npm install --no-fund --no-audit --no-bin-links 2>&1) || fail "npm install failed"
  fi
  ok "Vue dependencies ready"
fi

# ---------- 清理函数 ----------
PIDS=()
cleanup() {
  echo ""
  info "Shutting down all services..."
  for pid in "${PIDS[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  wait 2>/dev/null
  ok "All services stopped"
}
trap cleanup EXIT INT TERM

# ---------- 启动 Python Exp 服务 ----------
if [[ "$SKIP_EXP" != "true" ]]; then
  info "Starting Python Exp service on port ${EXP_PORT}..."
  (
    cd examples/exp || exit 1
    exec "$EXP_VENV/bin/uvicorn" server:app \
      --host 0.0.0.0 --port "$EXP_PORT" \
      --log-level info \
      > >(sed "s/^/  ${YELLOW}[EXP]${NC} /") 2>&1
  ) &
  PIDS+=($!)
  ensure_started "${PIDS[-1]}" "Exp service" "$EXP_PORT" 3
  ok "Exp service: http://localhost:${EXP_PORT}"
fi

# ---------- 启动 Go API 服务 ----------
EXP_FLAG=""
if [[ "$SKIP_EXP" != "true" ]]; then
  EXP_FLAG="--exp-url http://127.0.0.1:${EXP_PORT}"
fi

info "Starting Go API server on port ${API_PORT}..."
(
  exec "$PORTAL_BIN" \
    -listen ":${API_PORT}" \
    -tools "$TOOLS" \
    -workflow "$WORKFLOW" \
    $EXP_FLAG \
    > >(sed "s/^/  ${CYAN}[API]${NC} /") 2>&1
) &
PIDS+=($!)
ensure_started "${PIDS[-1]}" "API server" "$API_PORT" 3
ok "API server: http://localhost:${API_PORT}"

# ---------- 启动 Vue Dev Server ----------
if [[ "$SKIP_WEB" != "true" ]]; then
  info "Starting Vue dev server on port ${WEB_PORT}..."
  (
    cd web || exit 1
    exec node node_modules/vite/bin/vite.js --port "$WEB_PORT" --host \
      > >(sed "s/^/  ${GREEN}[WEB]${NC} /") 2>&1
  ) &
  PIDS+=($!)
  ensure_started "${PIDS[-1]}" "Vue frontend" "$WEB_PORT" 4
  ok "Vue frontend: http://localhost:${WEB_PORT}"
fi

# ---------- 就绪 ----------
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Penframe is ready!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "  API Server:   ${CYAN}http://localhost:${API_PORT}${NC}"
if [[ "$SKIP_EXP" != "true" ]]; then
echo -e "  Exp Service:  ${YELLOW}http://localhost:${EXP_PORT}${NC}"
echo -e "  Exp Modules:  ${YELLOW}http://localhost:${EXP_PORT}/api/v1/exploits${NC}"
fi
if [[ "$SKIP_WEB" != "true" ]]; then
echo -e "  Web UI:       ${GREEN}http://localhost:${WEB_PORT}${NC}"
fi
echo ""
echo -e "  API Endpoints:"
echo -e "    POST /api/scan          Start a scan"
echo -e "    GET  /api/assets        Asset graph (Cytoscape JSON)"
echo -e "    GET  /api/tasks         Scan task queue"
echo -e "    GET  /api/events        SSE live events"
echo -e "    GET  /api/exploits      List exploit modules"
echo -e "    POST /api/exploit       Trigger exploit"
echo -e "    GET  /api/runs          Run history"
echo -e "    POST /api/run           Legacy workflow run"
echo ""
if [[ -n "$TARGET" ]]; then
  echo -e "  Default target: ${CYAN}${TARGET}${NC}"
  echo ""
fi
echo -e "  Press ${RED}Ctrl+C${NC} to stop all services"
echo ""

# ---------- 等待所有子进程 ----------
wait
