# ModHandlerGo — 激光测距系统 PC 配套工具

## 项目概述

Go + Wails v3 的激光测距系统 PC 配套工具。支持 CAN 总线调试、固件升级、LoRa 网关通信、设备/网关模拟器。

## 技术栈

- **后端**: Go 1.25 + Wails v3.0.0-alpha.98
- **前端**: React 18 + TypeScript + Tailwind CSS v4 + shadcn/ui
- **CAN**: PCAN-Basic (Windows, syscall.LazyDLL) / SocketCAN (Linux, netlink)
- **串口**: go.bug.st/serial (跨平台)
- **终端**: xterm.js (ANSI/VT100 仿真)
- **协议**: USR1566 JSON / LoRa / CAN 2.0 (大端序)

## 目录结构

```
├── main.go                     # 入口：注册服务+事件+窗口
├── internal/
│   ├── canhal/                 # CAN HAL 接口 + PCAN/SocketCAN 实现
│   ├── candispatcher/          # goroutine 读循环 + channel pub/sub + FeedFrame
│   ├── canmanager/             # 固件升级状态机 (0x101/0x102/0x103)
│   ├── cancommand/             # 帧收发 + 总线监视 + 快捷命令
│   ├── uartmanager/            # 串口固件升级 (帧格式: 0xAA+TYPE+LEN_BE+DATA+CRC16_BE+0x55)
│   ├── lorasdk/                # LoRa SDK: TCP 数据流 / UDP 发现 / Serial AT
│   └── loraservice/            # Wails 事件桥接
├── service/                    # Wails 服务绑定层 (8 个 service)
│   ├── common_service.go       # 共享 CAN 基础设施 + notifyingBackend
│   ├── can_upgrade_service.go  # 固件升级服务
│   ├── can_command_service.go  # CAN 命令服务
│   ├── simulator_service.go    # CAN 设备模拟器 (纯 Go, 共享 CAN HAL)
│   ├── gateway_sim_service.go  # LoRa 网关模拟器 (纯 Go, TCP+UDP)
│   ├── lora_data_service.go    # LoRa 数据服务
│   ├── lora_config_service.go  # LoRa 配置服务
│   └── terminal_service.go     # UART/TCP 终端
├── frontend/src/
│   ├── App.tsx                 # 左侧边栏 + 页面路由 (7 个页面)
│   ├── lib/i18n.tsx            # 中英文翻译 (170+ 键)
│   ├── hooks/useEvents.ts      # Wails 事件监听 (useRef 保持稳定)
│   ├── components/              # shadcn/ui 风格组件
│   └── pages/                  # LoRa数据/LoRa配置/固件升级/CAN命令/设备模拟器/网关模拟器/终端
├── build/                      # 构建配置 (NSIS/nfpm/图标)
└── .github/workflows/release.yml  # CI/CD: tag → 自动构建+发布
```

## 构建命令

```bash
# 开发
wails3 dev

# 生产构建
wails3 build

# Windows 安装包 (需要 NSIS)
wails3 task windows:create:nsis:installer

# Linux 打包
wails3 task linux:create:deb
wails3 task linux:create:rpm

# 重新生成 TypeScript 绑定
wails3 generate bindings -d frontend/bindings -ts ./...
```

## 关键设计决策

1. **无 CGo**: PCAN 用 `syscall.LazyDLL`，SocketCAN 用纯 Go netlink — 简化跨平台编译
2. **事件驱动**: 后端 `app.Event.Emit` → 前端 `Events.On`，不轮询
3. **UDP 广播**: 所有 LoRa 网关通信走广播 (port 1566)，MAC 字段标识目标
4. **帧格式**: UART 帧 `0xAA+TYPE+LEN_BE+DATA+CRC16_BE+0x55`，CAN 帧大端序
5. **主题**: 亮色 (ccx 风格) / 暗色 (Dracula 配色)
6. **终端**: xterm.js 实现 ANSI 仿真，RAW 模式逐字符发送，无本地回显
7. **模拟器共享 CAN HAL**: 设备模拟器通过 `dispatcher.FeedFrame` 注入帧，`notifyingBackend` 拦截发送帧，避免 loopback 依赖
8. **网关模拟器纯 Go**: TCP 帧协议 (CRC16-CCITT) + UDP 设备发现/AT 命令全部 Go 实现

## CAN 协议 (mod-can.h)

| 帧 ID | 方向 | 用途 |
|-------|------|------|
| 0x101 | 平台→设备 | 控制命令 (升级/确认/版本/重启) |
| 0x102 | 设备→平台 | 响应帧 |
| 0x103 | 平台→设备 | 固件数据传输 |
| 0x105 | 平台→设备 | LoRa 配参命令 |
| 0x106 | 设备→平台 | LoRa 配参响应 |
| 0x763 | 设备→平台 | 心跳 |
| 0x1E3 | 设备→平台 | 手柄状态 (X/Y BE + 按键) |
| 0x263 | 平台→设备 | 超欠挖 + 激光测距 |
| 0x363 | 平台→设备 | X/Y 坐标 |
| 0x463 | 平台→设备 | Z 坐标 |

## LoRa UDP 协议

所有 UDP 消息格式: `USR1566` + JSON + `USR1566` (port 1566)

- **搜索**: `{"VER":"1.0","MSG":"SEARCH","TYPE":"LORA"}`
- **AT 命令**: `{"VER":"1.0","MSG":"GETPARA/SETPARA","TYPE":"AT","CMD":"AT+...","USER":"admin","PSW":"admin","MAC":"..."}`
- **网络参数**: `{"VER":"1.0","MSG":"GETPARA","TYPE":"JSON","CMD":"NETDEV",...}`

## 开发注意事项

- 后端日志用英文（技术日志不翻译）
- 前端 UI 标签通过 `t("key")` 翻译
- 提交前必须验证构建通过 (`go build .` + `tsc --noEmit`)
- tag 格式: `v*.*.*` (如 v0.1.0)
- 新增 Service 后需运行 `wails3 generate bindings -d frontend/bindings -ts ./...` 更新绑定

## LoRa 工作模式协议

NWMODE 决定组网模式，工作模式根据 NWMODE 选择不同的 AT 命令：

| NWMODE | 含义 | 工作 AT | 模式值 |
|--------|------|---------|--------|
| 0 | 不组网(透传) | AT+TTMODE | 0=广播透传 / 1=指定节点 |
| 1 | 组网 | AT+WMODE | 0=广播透传 / 1=指定节点 / 2=主动上报 |

前端根据 `isMesh` 自动切换下拉选项和 AT 命令，SET NWMODE 后自动查询对应工作模式。

## 模拟器

### CAN 设备模拟器 (simulator_service.go)
- 纯 Go 实现，共享 CAN HAL（通过 `notifyingBackend` + `dispatcher.FeedFrame`）
- 模拟固件升级响应 (0x102)、LoRa 配参 (0x106)、心跳 (0x763)、手柄状态 (0x1E3)
- 需先通过固件升级页连接 CAN，再启动模拟器

### LoRa 网关模拟器 (gateway_sim_service.go)
- 纯 Go 实现，TCP 服务器 (帧协议) + UDP 服务器 (设备发现/AT 命令)
- 支持遥测发送、RSSI 查询、自动遥测
- 独立运行，无需 CAN 连接

### UART/TCP 终端 (terminal_service.go)
- 支持 TCP 客户端和串口两种连接方式
- readLoop goroutine 读取数据并通过 `terminal:data` 事件推送前端
- 连接断开自动清理状态，主动断开不报错
