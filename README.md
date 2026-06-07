# 激光测距工具 (Laser Range Tool)

基于 Go + Wails v3 的激光测距系统 PC 端配套工具，从 [can-uart-tool](https://github.com/kabirz/can-uart-tool) (C/Win32) 完整迁移而来。

## 功能

### LoRa 数据
- TCP 网关连接，实时遥测数据显示 (X/Y角度、激光测距、按键状态)
- 手柄数据解析与显示（中英文：按下/抬起）
- 测试帧回显
- RSSI 信号查询
- 历史记录表格 + CSV 导出
- 自动滚动 + 日志面板

### LoRa 配置
- UDP/串口双传输方式（滑动切换），USR1566 JSON 协议
- 串口模式自动枚举端口，打开/关闭一键操作
- 设备搜索与发现，设备信息平铺展示
- 网络参数配置 (IP/掩码/网关/DHCP/SOCKA/SOCKEN)
- LoRa 协议参数配置 (组网模式/工作模式/通道滑动切换/频率×100KHz/速度/功率/上行ID携带选择)
- AT 命令控制台 + 响应日志

### 固件升级
- CAN (PCAN) 和 UART 双通道固件烧录（滑动切换）
- 串口自动枚举，选择后直接可用，不需要手动检测
- 公共升级逻辑提取 (internal/upgrade)，CAN/UART 共享状态机
- 进度条显示，版本查询，板卡重启

### CAN 命令
- 自定义 CAN 帧发送 (标准帧/扩展帧/数据帧/远程帧)
- 总线监视器 (自动标注已知帧 ID，稳定 key 优化)
- 快捷命令按钮
- LoRa 远程配参 (协议/模式/通道/NID/GWID/上电/测试模式)
- 回发扫描仪数据 (收到手柄数据后自动响应)
- 未连接 CAN 时标签灰显 + 提示

### 其他
- 版本更新检查（启动自动检测 + 手动点击版本号）
- 中英文完整国际化
- 亮色 (ccx) / 暗色 (Dracula) 双主题
- 自定义 Logo（十字准星 + 激光束 + CAN 波形 + LoRa 信号）

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + Wails v3 |
| 前端 | React 18 + TypeScript + Tailwind CSS v4 |
| UI 组件 | shadcn/ui + lucide-react |
| CAN | PCAN-Basic (Windows) / SocketCAN (Linux) |
| 串口 | go.bug.st/serial (跨平台) |
| 协议 | USR1566 JSON / LoRa / CAN 2.0 |
| 测试 | Go testing (protocol + framing 单元测试) |

## 平台支持

| 平台 | CAN 适配器 | 安装包格式 | 权限 |
|------|-----------|-----------|------|
| Windows | PCAN-Basic | NSIS installer (.exe) | 用户级（无需管理员） |
| Linux | SocketCAN | .deb / .rpm | - |

## 从源码构建

### 前置条件

- [Go 1.25+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/)
- [Wails v3 CLI](https://v3.wails.io): `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`
- Windows: [PCAN-Basic SDK](https://www.peak-system.com/PCAN-Basic.199.0.html) (运行时动态加载，未安装不影响编译)
- Linux: `libgtk-4-dev libwebkit2gtk-4.1-dev`

### 开发模式

```bash
wails3 dev
```

### 生产构建

```bash
wails3 build
```

### 打包安装包

**Windows (NSIS):**
```bash
# 需要安装 NSIS: https://nsis.sourceforge.io
wails3 task windows:create:nsis:installer
```

**Linux:**
```bash
# DEB
wails3 task linux:create:deb

# RPM
wails3 task linux:create:rpm
```

## CI/CD

推送 `v*` tag 时自动触发 GitHub Actions 构建并发布 Release：

```bash
git tag -a v0.1.6 -m "Release v0.1.6"
git push origin v0.1.6
```

构建产物：
- Windows: `modhandlergo-amd64-installer.exe`
- Linux: `.deb` + `.rpm` 包

## 项目结构

```
ModHandlerGo/
├── main.go                          # 入口，注册服务+事件+窗口
├── internal/
│   ├── canhal/                      # CAN HAL 接口 + PCAN/SocketCAN 实现
│   ├── candispatcher/               # 帧分发器 (goroutine + pub/sub + sync.Pool)
│   ├── canmanager/                  # CAN 固件升级 (实现 upgrade.Transport)
│   ├── cancommand/                  # 帧收发 + 总线监视 + 快捷命令
│   ├── uartmanager/                 # 串口固件升级 (实现 upgrade.Transport)
│   ├── upgrade/                     # 公共固件升级逻辑 (Transport 接口 + 状态机)
│   ├── lorasdk/                     # LoRa SDK (TCP/UDP/Serial AT + 单元测试)
│   └── loraservice/                 # Wails 事件桥接层
├── service/                         # Wails 服务绑定层
│   ├── common_service.go            # 共享 CAN 基础设施
│   ├── can_upgrade_service.go       # 固件升级服务
│   ├── can_command_service.go       # CAN 命令服务
│   ├── lora_data_service.go         # LoRa 数据服务
│   └── lora_config_service.go       # LoRa 配置服务
├── frontend/
│   ├── src/
│   │   ├── App.tsx                  # 主布局 + 版本检查 + CAN 状态管理
│   │   ├── lib/i18n.tsx             # 中英文翻译系统 (120+ 键)
│   │   ├── hooks/useEvents.ts       # Wails 事件监听 hook
│   │   ├── components/
│   │   │   └── layout/sidebar.tsx   # 侧边栏 (导航+主题+语言+版本检查+关于)
│   │   └── pages/                   # 4 个功能页面 (React.memo 优化)
│   └── bindings/                    # Wails 自动生成的 TypeScript 绑定
├── build/                           # 构建配置 (NSIS/nfpm/图标)
│   └── windows/
│       ├── icon.svg                 # 矢量 Logo
│       └── icon.ico                 # Windows 图标 (16-256px)
├── .github/workflows/release.yml    # GitHub Actions CI/CD
├── go.mod
└── Taskfile.yml
```

## 架构亮点

- **公共固件升级层** (`internal/upgrade/`)：CAN 和 UART 共享 `Transport` 接口 + 升级状态机，消除约 200 行重复代码
- **串口可靠性**：打开操作 3 秒超时 + AT 模式懒加载，无效端口不会阻塞
- **前端性能**：React.memo 组件优化 + 合并高频 setState + sync.Pool 复用切片 + 稳定列表 key
- **事件驱动**：后端 `app.Event.Emit` → 前端 `useWailsEvent`，零轮询

## 原版对照

| can-uart-tool (C/Win32) | LaserRangeTool (Go + Wails v3) |
|--------------------------|-------------------------------|
| Tab 控件导航 | 左侧边栏导航 |
| Win32 API | 跨平台 (Windows + Linux) |
| PCAN / IXXAT | PCAN / SocketCAN |
| LoRa SDK (C) | LoRa SDK (纯 Go) |
| 无中英文切换 | 中英文完整国际化 |
| 固定主题 | 亮色 (ccx) / 暗色 (Dracula) |
| 无版本更新检查 | 启动自动检测 + 手动检查 |
| 无 Logo | 自定义矢量 Logo |
| 需管理员权限安装 | 用户级安装 (无需 UAC) |

## 许可

[Apache-2.0](LICENSE)

## ⚠️ 安全说明

本软件为**未签名**的开源程序，Windows SmartScreen / 杀毒软件可能会弹出安全警告，这是正常现象。

**原因：** 代码签名证书需要付费购买，本项目暂未配置。

**验证方法：**
- 源代码完全公开，可自行审查 [源码](https://github.com/kabirz/modhandlergo)
- 构建产物可通过 `sha256sum` 校验哈希值
- 也可从源码自行编译：参见 [从源码构建](#从源码构建) 章节

**运行被阻止的程序：**
1. 右键点击安装包 → 属性 → 勾选"解除锁定"
2. 或在 SmartScreen 弹窗中选择"仍要运行"

> 安装包已设置为用户级权限安装 (`RequestExecutionLevel "user"`)，不需要管理员权限。
