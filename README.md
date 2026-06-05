# 激光测距工具 (Laser Range Tool)

基于 Go + Wails v3 的激光测距系统 PC 端配套工具，从 [can-uart-tool](https://github.com/kabirz/can-uart-tool) (C/Win32) 完整迁移而来。

## 功能

### LoRa 数据
- TCP 网关连接，实时遥测数据显示 (X/Y角度、激光测距、按键状态)
- 手柄数据解析与显示
- 测试帧回显
- RSSI 信号查询
- 历史记录表格 + CSV 导出

### LoRa 配置
- UDP/串口双传输方式，USR1566 JSON 协议
- 设备搜索与发现
- 网络参数配置 (IP/掩码/网关/DHCP/SOCKA/SOCKEN)
- LoRa 协议参数配置 (组网模式/工作模式/通道/频率/速度/功率/UPWID)
- AT 命令控制台

### 固件升级
- CAN (PCAN) 和 UART 双通道固件烧录
- 串口枚举与选择
- 进度条显示，版本查询，板卡重启

### CAN 命令
- 自定义 CAN 帧发送 (标准帧/扩展帧/数据帧/远程帧)
- 总线监视器 (自动标注已知帧 ID)
- 快捷命令按钮
- LoRa 远程配参 (协议/模式/通道/NID/GWID/上电/测试模式)
- 回发扫描仪数据 (收到手柄数据后自动响应)

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25 + Wails v3 |
| 前端 | React 18 + TypeScript + Tailwind CSS v4 |
| UI 组件 | shadcn/ui + lucide-react |
| CAN | PCAN-Basic (Windows) / SocketCAN (Linux) |
| 串口 | go.bug.st/serial (跨平台) |
| 协议 | USR1566 JSON / LoRa / CAN 2.0 |

## 平台支持

| 平台 | CAN 适配器 | 安装包格式 |
|------|-----------|-----------|
| Windows | PCAN-Basic | NSIS installer (.exe) |
| Linux | SocketCAN | .deb / .rpm |

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
git tag v0.1.0
git push origin v0.1.0
```

构建产物：
- Windows: `ModHandlerGo-amd64-installer.exe`
- Linux: `.deb` + `.rpm` 包

## 项目结构

```
ModHandlerGo/
├── main.go                          # 入口，注册服务+事件+窗口
├── internal/
│   ├── canhal/                      # CAN HAL 接口 + PCAN/SocketCAN 实现
│   ├── candispatcher/               # 帧分发器 (goroutine + pub/sub)
│   ├── canmanager/                  # 固件升级状态机 (0x101/0x102/0x103)
│   ├── cancommand/                  # 帧收发 + 总线监视 + 快捷命令
│   ├── uartmanager/                 # 串口固件升级
│   ├── lorasdk/                     # LoRa SDK (TCP/UDP/Serial AT)
│   └── loraservice/                 # Wails 事件桥接层
├── service/                         # Wails 服务绑定层
│   ├── common_service.go            # 共享 CAN 基础设施
│   ├── can_upgrade_service.go       # 固件升级服务
│   ├── can_command_service.go       # CAN 命令服务
│   ├── lora_data_service.go         # LoRa 数据服务
│   └── lora_config_service.go       # LoRa 配置服务
├── frontend/
│   ├── src/
│   │   ├── App.tsx                  # 主布局 (左侧边栏 + 页面路由)
│   │   ├── lib/i18n.tsx             # 中英文翻译系统
│   │   ├── hooks/useEvents.ts       # Wails 事件监听 hook
│   │   ├── components/              # UI 组件
│   │   └── pages/                   # 4 个功能页面
│   └── bindings/                    # Wails 自动生成的 TypeScript 绑定
├── build/                           # 构建配置 (NSIS/nfpm/图标)
├── .github/workflows/release.yml    # GitHub Actions CI/CD
├── go.mod
└── Taskfile.yml
```

## 原版对照

| can-uart-tool (C/Win32) | LaserRangeTool (Go + Wails v3) |
|--------------------------|-------------------------------|
| Tab 控件导航 | 左侧边栏导航 |
| Win32 API | 跨平台 (Windows + Linux) |
| PCAN / IXXAT | PCAN / SocketCAN |
| LoRa SDK (C) | LoRa SDK (纯 Go) |
| 无中英文切换 | 中英文切换 |
| 固定主题 | 亮色 (ccx) / 暗色 (Dracula) |

## 许可

[Apache-2.0](LICENSE)
