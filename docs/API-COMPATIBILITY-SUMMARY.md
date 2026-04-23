# API兼容性总结 - 三种CVE类型支持

## 概述

系统现在完全支持三种CVE漏洞利用类型，所有API层都已更新以兼容这些类型。

## 三种CVE类型

### 1. RCE (Remote Code Execution)
- **exploit_kind**: `"execute"` 或 `"rce"`
- **特点**: 支持命令执行
- **必需字段**: `command`
- **示例**: CVE-2025-55182 (React2Shell)

### 2. LEAK (File Disclosure/Read)
- **exploit_kind**: `"leak"`
- **特点**: 文件泄露/读取
- **自动注入字段**: `leak_path`
- **示例**: CVE-2025-48957 (AstrBot Path Traversal)

### 3. UPLOAD (Arbitrary File Upload)
- **exploit_kind**: `"upload"`
- **特点**: 任意文件上传
- **自动注入字段**: `upload_file`, `remote_path`
- **示例**: CVE-2025-26319 (FlowiseAI File Upload)

## API层修改

### 1. 后端Go API (`internal/portal/api_exploit.go`)

#### exploitRequest结构体
```go
type exploitRequest struct {
    Target     string         `json:"target"`
    ExploitID  string         `json:"exploit_id,omitempty"`
    Mode       string         `json:"mode,omitempty"`
    Command    string         `json:"command,omitempty"`
    LeakPath   string         `json:"leak_path,omitempty"`      // LEAK类型
    UploadFile string         `json:"upload_file,omitempty"`    // UPLOAD类型
    RemotePath string         `json:"remote_path,omitempty"`    // UPLOAD类型
    Options    map[string]any `json:"options,omitempty"`
    ProjectID  string         `json:"project_id,omitempty"`
    TargetID   string         `json:"target_id,omitempty"`
}
```

#### exploitOptions函数
自动将顶层字段合并到options中：
- `leak_path` → `options["leak_path"]`
- `upload_file` → `options["upload_file"]`
- `remote_path` → `options["remote_path"]`

#### enrichExploitResponse函数
在响应的echo字段中包含所有请求参数，便于调试和追踪。

### 2. 后端Go聚合 (`internal/cveindex/aggregate.go`)

#### ExpModuleInfo结构体
```go
type ExpModuleInfo struct {
    ID              string   `json:"id"`
    Name            string   `json:"name"`
    Description     string   `json:"description"`
    CVE             string   `json:"cve"`
    Severity        string   `json:"severity"`
    Tags            []string `json:"tags"`
    SupportsCheck   bool     `json:"supports_check"`
    SupportsExploit bool     `json:"supports_exploit"`
    SupportsCommand bool     `json:"supports_command"`
    ExploitKind     string   `json:"exploit_kind"`    // 支持三种类型
    DefaultCommand  string   `json:"default_command"`
}
```

### 3. 前端TypeScript类型 (`web/src/types/index.ts`)

#### ExploitInfo接口
```typescript
export interface ExploitInfo {
  id: string
  name: string
  description: string
  cve: string
  severity: string
  targets: string[]
  supports_check?: boolean
  supports_exploit?: boolean
  supports_execute?: boolean
  supports_command?: boolean
  exploit_kind?: string              // 支持三种类型
  tags?: string[]
  options?: ExploitOption[]
  default_command?: string
}
```

#### ExploitRequest接口
```typescript
export interface ExploitRequest {
  target: string
  exploit_id?: string
  mode?: 'check' | 'execute'
  command?: string
  leak_path?: string      // LEAK类型
  upload_file?: string    // UPLOAD类型
  remote_path?: string    // UPLOAD类型
  options?: Record<string, string>
}
```

### 4. 前端API调用 (`web/src/api/exploit.ts`)

```typescript
export function triggerExploit(payload: ExploitRequestWithContext): Promise<any> {
  const normalizedOptions = payload.options || undefined
  return post<any>('/api/exploit', {
    target: payload.target,
    exploit_id: payload.exploit_id || 'auto',
    mode: payload.mode || 'check',
    command: payload.command || '',
    leak_path: payload.leak_path || payload.options?.leak_path || '',
    upload_file: payload.upload_file || payload.options?.upload_file || '',
    remote_path: payload.remote_path || payload.options?.remote_path || '',
    options: normalizedOptions,
    project_id: payload.project_id,
    target_id: payload.target_id,
  })
}
```

### 5. Python模型 (`cve/exp/expcore/models.py`)

#### ModuleInfo自动注入
```python
@model_validator(mode="after")
def inject_leak_option(self) -> "ModuleInfo":
    exploit_kind = (self.exploit_kind or "").strip().lower()
    existing_keys = {option.key for option in self.options}

    # LEAK类型自动注入
    if (exploit_kind == "leak" or "leak" in tags) and "leak_path" not in existing_keys:
        self.options.append(ModuleOption(
            key="leak_path",
            label="Leak Path",
            placeholder="../../../../etc/passwd",
            description="Optional target file path for leak-type exploits.",
            required=False,
            modes=["execute"],
        ))

    # UPLOAD类型自动注入
    if (exploit_kind == "upload" or "upload" in tags):
        if "upload_file" not in existing_keys:
            self.options.append(ModuleOption(
                key="upload_file",
                label="Upload File",
                type="file",
                placeholder="Select file to upload",
                description="Local file to upload to the target server",
                required=False,
                modes=["execute"],
            ))
        if "remote_path" not in existing_keys:
            self.options.append(ModuleOption(
                key="remote_path",
                label="Remote Path",
                type="text",
                placeholder="/tmp/webshell.php",
                description="Target path on the remote server",
                required=False,
                modes=["execute"],
            ))

    return self
```

### 6. 前端UI (`web/src/views/ExploitPanel.vue`)

#### 文件类型输入框
```vue
<el-input
  v-if="option.type === 'file'"
  v-model="exploitOptionValues[option.key]"
  :placeholder="option.placeholder || 'Enter file path'"
>
  <template #append>
    <el-button @click="selectFile(option.key)">Browse</el-button>
  </template>
</el-input>
```

#### selectFile函数
```typescript
function selectFile(optionKey: string) {
  const input = document.createElement('input')
  input.type = 'file'
  input.onchange = (e: Event) => {
    const target = e.target as HTMLInputElement
    if (target.files && target.files.length > 0) {
      exploitOptionValues.value[optionKey] = target.files[0].name
    }
  }
  input.click()
}
```

#### 提交时的特殊处理
```typescript
if (Object.keys(options).length > 0) {
  payload.options = options
  if (options.leak_path) payload.leak_path = options.leak_path
  if (options.upload_file) payload.upload_file = options.upload_file
  if (options.remote_path) payload.remote_path = options.remote_path
}
```

### 7. 前端TargetWorkspace (`web/src/views/TargetWorkspace.vue`)

#### inferExploitKind函数
```typescript
function inferExploitKind(vuln: VulnerabilityGroup): 'execute' | 'leak' | 'upload' {
  const text = `${vuln.name} ${vuln.hits.map(hit => `${hit.cve || ''} ${hit.detail || ''}`).join(' ')}`.toLowerCase()
  
  // 检测UPLOAD类型
  if (/(upload|file upload|arbitrary upload|unrestricted upload|path traversal.*upload)/.test(text)) {
    return 'upload'
  }
  
  // 检测LEAK类型
  if (/(disclosure|exposure|exposed|leak|leakage|sensitive|secret|credential|token|dump|directory listing|file read|source code|config|env\b)/.test(text)) {
    return 'leak'
  }
  
  // 默认RCE类型
  return 'execute'
}
```

#### buildFallbackExploit函数
```typescript
function buildFallbackExploit(vuln: VulnerabilityGroup): ExploitInfo {
  const exploitKind = inferExploitKind(vuln)
  
  return {
    id: 'auto',
    name: vuln.name,
    // ...
    exploit_kind: exploitKind,
    options: exploitKind === 'leak'
      ? [{ key: 'leak_path', label: 'Leak Path', ... }]
      : exploitKind === 'upload'
      ? [
          { key: 'upload_file', label: 'Upload File', type: 'file', ... },
          { key: 'remote_path', label: 'Remote Path', ... }
        ]
      : [],
    // ...
  }
}
```

## 数据流

### UPLOAD类型的完整数据流

1. **用户操作**:
   - 在ExploitPanel中选择CVE-2025-26319
   - 点击"Exploit"按钮
   - 填写Target URL
   - 点击"Browse"选择文件或输入文件路径
   - 输入Remote Path

2. **前端处理**:
   - `exploitOptionValues` 收集用户输入
   - `buildPayloadOptions()` 构建options对象
   - `submitAction()` 调用 `triggerExploit()`

3. **API调用**:
   - `triggerExploit()` 将 `upload_file` 和 `remote_path` 提取到顶层字段
   - POST `/api/exploit` 发送请求

4. **后端Go处理**:
   - `handleExploit()` 解析 `exploitRequest`
   - `exploitOptions()` 将顶层字段合并到options
   - 调用Python服务的 `/api/v1/execute` 端点

5. **Python处理**:
   - ExpExecutor接收请求
   - 调用模块的 `exploit()` 方法
   - 检测 `kwargs.get("upload_file")` 和 `kwargs.get("remote_path")`
   - 执行文件上传逻辑

6. **响应返回**:
   - Python返回 `ExploitResult`
   - Go的 `enrichExploitResponse()` 添加echo字段
   - 前端显示结果

## 兼容性保证

### 向后兼容
- 所有新增字段都是可选的（`omitempty`）
- 现有RCE和LEAK类型的exploit模块无需修改
- 旧的API调用仍然有效

### 类型安全
- Go使用结构体标签确保JSON序列化正确
- TypeScript接口提供编译时类型检查
- Python使用Pydantic模型验证

### 自动化
- Python模型自动注入类型特定的选项
- 前端自动识别漏洞类型并生成对应UI
- 后端自动合并顶层字段到options

## 测试建议

### 1. RCE类型测试
```bash
curl -X POST http://localhost:8080/api/exploit \
  -H "Content-Type: application/json" \
  -d '{
    "target": "http://target.com",
    "exploit_id": "cve-2025-55182",
    "mode": "execute",
    "command": "id"
  }'
```

### 2. LEAK类型测试
```bash
curl -X POST http://localhost:8080/api/exploit \
  -H "Content-Type: application/json" \
  -d '{
    "target": "http://target.com",
    "exploit_id": "cve-2025-48957",
    "mode": "execute",
    "leak_path": "../../../../etc/passwd"
  }'
```

### 3. UPLOAD类型测试
```bash
curl -X POST http://localhost:8080/api/exploit \
  -H "Content-Type: application/json" \
  -d '{
    "target": "http://target.com",
    "exploit_id": "cve-2025-26319",
    "mode": "execute",
    "upload_file": "/tmp/webshell.php",
    "remote_path": "/var/www/html/shell.php"
  }'
```

## 总结

所有API层现在都完全兼容三种CVE类型：
- ✅ 后端Go API结构体
- ✅ 后端Go聚合逻辑
- ✅ 前端TypeScript类型
- ✅ 前端API调用
- ✅ Python模型验证
- ✅ 前端UI组件
- ✅ 自动类型推断

系统现在可以无缝处理RCE、LEAK和UPLOAD三种类型的CVE漏洞利用。
