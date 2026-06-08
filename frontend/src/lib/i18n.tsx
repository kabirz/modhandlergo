import React, { createContext, useContext, useState, useCallback, useEffect } from "react";

export type Lang = "zh" | "en";

interface I18nContextType {
  lang: Lang;
  setLang: (l: Lang) => void;
  t: (key: string) => string;
}

const I18nContext = createContext<I18nContextType>({ lang: "zh", setLang: () => {}, t: (k) => k });

export function useI18n() {
  return useContext(I18nContext);
}

// Translation dictionary
const dict: Record<string, Record<Lang, string>> = {
  // Sidebar
  "nav.loraData":     { zh: "LoRa 数据",   en: "LoRa Data" },
  "nav.loraConfig":   { zh: "LoRa 配置",   en: "LoRa Config" },
  "nav.firmware":     { zh: "固件升级",     en: "Firmware" },
  "nav.canCommand":   { zh: "CAN 命令",    en: "CAN Command" },
  "nav.canDisabled":  { zh: "请先在固件升级中打开CAN设备", en: "Open CAN device in Firmware page first" },
  "sidebar.theme":    { zh: "主题",         en: "Theme" },
  "sidebar.light":    { zh: "亮色",         en: "Light" },
  "sidebar.dark":     { zh: "暗色",         en: "Dark" },
  "sidebar.version":  { zh: "版本",         en: "Version" },
  "sidebar.lang":     { zh: "语言",         en: "Lang" },

  // LoRa Data
  "lora.conn":        { zh: "连接",         en: "Connect" },
  "lora.disconnect":  { zh: "断开",         en: "Disconnect" },
  "lora.port":        { zh: "端口",         en: "Port" },
  "lora.testMode":    { zh: "测试模式",     en: "Test Mode" },
  "lora.joystick":    { zh: "手柄数据",     en: "Joystick" },
  "lora.xAngle":      { zh: "X 角度",       en: "X Angle" },
  "lora.yAngle":      { zh: "Y 角度",       en: "Y Angle" },
  "lora.btnState":    { zh: "按键状态",     en: "Button" },
  "lora.btnPressed":  { zh: "按下",         en: "Pressed" },
  "lora.btnReleased": { zh: "抬起",         en: "Released" },
  "lora.rawLog":      { zh: "原始日志",     en: "Raw Log" },
  "lora.waitLog":     { zh: "等待日志...",   en: "Waiting for logs..." },
  "lora.send":        { zh: "发送",         en: "Send" },
  "lora.clear":       { zh: "清除",         en: "Clear" },
  "lora.history":     { zh: "历史记录",     en: "History" },
  "lora.waitData":    { zh: "等待数据...",   en: "Waiting for data..." },
  "lora.time":        { zh: "时间",         en: "Time" },
  "lora.type":        { zh: "类型",         en: "Type" },
  "lora.data":        { zh: "数据",         en: "Data" },
  "lora.saved":       { zh: "已保存",       en: "Saved" },
  "lora.connecting":  { zh: "连接中",       en: "Connecting" },

  // LoRa Config
  "cfg.transport":    { zh: "连接方式",     en: "Transport" },
  "cfg.udp":          { zh: "UDP (网络)",   en: "UDP (Network)" },
  "cfg.serial":       { zh: "串口 (COM)",   en: "Serial (COM)" },
  "cfg.baud":         { zh: "波特率",       en: "Baud Rate" },
  "cfg.open":         { zh: "打开",         en: "Open" },
  "cfg.close":        { zh: "关闭",         en: "Close" },
  "cfg.discovery":    { zh: "设备发现",     en: "Device Discovery" },
  "cfg.search":       { zh: "搜索设备",     en: "Search" },
  "cfg.getNet":       { zh: "获取网络",     en: "Get Network" },
  "cfg.queryGwid":    { zh: "查询GWID",     en: "Query GWID" },
  "cfg.reboot":       { zh: "重启网关",     en: "Reboot" },
  "cfg.network":      { zh: "网络设置",     en: "Network" },
  "cfg.query":        { zh: "查询",         en: "Query" },
  "cfg.enable":       { zh: "开启",         en: "Enable" },
  "cfg.disable":      { zh: "关闭",         en: "Disable" },
  "cfg.mode":         { zh: "模式",         en: "Mode" },
  "cfg.set":          { zh: "设置",         en: "Set" },
  "cfg.mask":         { zh: "掩码",         en: "Mask" },
  "cfg.gateway":      { zh: "网关",         en: "Gateway" },
  "cfg.loraProto":    { zh: "LoRa 协议",    en: "LoRa Protocol" },
  "cfg.mesh":         { zh: "是否组网",     en: "Mesh" },
  "cfg.meshNo":       { zh: "否",           en: "No" },
  "cfg.meshYes":      { zh: "是",           en: "Yes" },
  "cfg.workMode":     { zh: "工作模式",     en: "Work Mode" },
  "cfg.broadcast":    { zh: "广播透传",     en: "Broadcast" },
  "cfg.targetNode":   { zh: "指定节点",     en: "Target Node" },
  "cfg.activeReport": { zh: "主动上报",     en: "Active Report" },
  "cfg.upwid":        { zh: "上行ID",         en: "UPWID" },
  "cfg.upwidOn":      { zh: "携带",           en: "On" },
  "cfg.upwidOff":     { zh: "不携带",         en: "Off" },
  "cfg.power":        { zh: "功率",         en: "Power" },
  "cfg.channel":      { zh: "通道",         en: "Channel" },
  "cfg.freq":         { zh: "频率",         en: "Freq" },
  "cfg.speed":        { zh: "速度",         en: "Speed" },
  "cfg.queryVer":     { zh: "查询版本",     en: "Query Version" },
  "cfg.responseLog":  { zh: "响应日志",     en: "Response Log" },
  "cfg.waitResp":     { zh: "等待响应...",   en: "Waiting for response..." },
  "cfg.openSerial":   { zh: "请先打开串口再进行其他操作",   en: "Open serial port first to continue" },

  // Firmware
  "fw.channel":       { zh: "通道选择",     en: "Channel" },
  "fw.baud":          { zh: "波特率",       en: "Baud Rate" },
  "fw.firmwareFile":  { zh: "固件文件",     en: "Firmware File" },
  "fw.browse":        { zh: "浏览",         en: "Browse" },
  "fw.startUpgrade":  { zh: "开始升级",     en: "Start Upgrade" },
  "fw.queryVer":      { zh: "查询版本",     en: "Query Version" },
  "fw.reboot":        { zh: "重启板卡",     en: "Reboot Board" },
  "fw.curVersion":    { zh: "当前版本",     en: "Current Version" },
  "fw.log":           { zh: "日志",         en: "Log" },
  "fw.waitOp":        { zh: "等待操作...",   en: "Waiting..." },
  "fw.selectFile":    { zh: "选择固件文件 (.bin)", en: "Select firmware file (.bin)" },
  "fw.pleaseSelect":  { zh: "请先选择固件文件", en: "Please select a firmware file first" },
  "fw.upgradeDone":  { zh: "固件上传完成，请点击重启板卡完成升级", en: "Firmware uploaded. Click Reboot Board to complete the upgrade." },
  "fw.rebootConfirm":{ zh: "确认重启板卡？", en: "Confirm reboot board?" },
  "fw.ok":            { zh: "确定", en: "OK" },
  "fw.cancel":        { zh: "取消", en: "Cancel" },

  // CAN Command
  "can.frameConfig":  { zh: "帧配置",       en: "Frame Config" },
  "can.format":       { zh: "格式",         en: "Format" },
  "can.standard":     { zh: "标准",         en: "Std" },
  "can.extended":     { zh: "扩展",         en: "Ext" },
  "can.dataFrame":    { zh: "数据",         en: "Data" },
  "can.remoteFrame":  { zh: "远程",         en: "Remote" },
  "can.data":         { zh: "数据",         en: "Data" },
  "can.sendFrame":    { zh: "发送帧",       en: "Send Frame" },
  "can.loraConfig":   { zh: "LoRa 配置",    en: "LoRa Config" },
  "can.powerOn":      { zh: "上电",         en: "Power On" },
  "can.powerOff":     { zh: "断电",         en: "Power Off" },
  "can.testMode":     { zh: "测试模式",     en: "Test Mode" },
  "can.busMonitor":   { zh: "总线监视器",   en: "Bus Monitor" },
  "can.autoScroll":   { zh: "自动滚动",     en: "Auto Scroll" },
  "can.time":         { zh: "时间",         en: "Time" },
  "can.label":        { zh: "标注",         en: "Label" },
  "can.control":      { zh: "控制命令",     en: "Control" },
  "can.response":     { zh: "响应帧",       en: "Response" },
  "can.firmware":     { zh: "固件数据",     en: "Firmware" },
  "can.loraResp":     { zh: "LoRa响应",     en: "LoRa Resp" },
  "can.joystick":     { zh: "手柄状态",     en: "Joystick" },
  "can.laser":        { zh: "超欠挖激光",   en: "Laser" },
  "can.coordXY":      { zh: "XY坐标",       en: "X/Y Coord" },
  "can.coordZ":       { zh: "Z坐标",        en: "Z Coord" },
  "can.heartbeat":    { zh: "心跳",         en: "Heartbeat" },

  // Simulator
  "nav.simulator":        { zh: "设备模拟器",     en: "Simulator" },
  "sim.config":           { zh: "模拟器配置",     en: "Simulator Config" },
  "sim.channel":          { zh: "CAN通道",        en: "CAN Channel" },
  "sim.version":          { zh: "固件版本",       en: "Firmware Version" },
  "sim.handlerInterval":  { zh: "手柄间隔",       en: "Handler Interval" },
  "sim.disableHeartbeat": { zh: "禁用心跳",       en: "Disable Heartbeat" },
  "sim.disableHandler":   { zh: "禁用手柄",       en: "Disable Handler" },
  "sim.control":          { zh: "控制",           en: "Control" },
  "sim.running":          { zh: "运行中",         en: "Running" },
  "sim.stopped":          { zh: "已停止",         en: "Stopped" },
  "sim.start":            { zh: "启动",           en: "Start" },
  "sim.stop":             { zh: "停止",           en: "Stop" },
  "sim.log":              { zh: "日志输出",       en: "Log Output" },
  "sim.noLog":            { zh: "暂无日志",       en: "No log output" },
  "sim.connectFirst":     { zh: "请先连接CAN",    en: "Connect CAN first" },

  // About dialog
  "about.title":       { zh: "关于",         en: "About" },
  "about.appName":     { zh: "应用名称",     en: "App Name" },
  "about.version":     { zh: "版本",         en: "Version" },
  "about.techStack":   { zh: "技术栈",       en: "Tech Stack" },
  "about.frontend":    { zh: "前端框架",     en: "Frontend" },
  "about.canAdapter":  { zh: "CAN 适配器",   en: "CAN Adapter" },
  "about.serial":      { zh: "串口",         en: "Serial" },
  "about.protocol":    { zh: "协议",         en: "Protocol" },
  "about.license":     { zh: "许可证",       en: "License" },
  "about.ok":          { zh: "确定",         en: "OK" },

  // Update check
  "update.check":      { zh: "检查更新",         en: "Check for updates" },
  "update.checking":   { zh: "检查更新中...",     en: "Checking for updates..." },
  "update.latest":     { zh: "已是最新版本",     en: "You are up to date" },
  "update.available":  { zh: "发现新版本",       en: "Update available" },
  "update.current":    { zh: "当前版本",         en: "Current version" },
  "update.latestVer":  { zh: "最新版本",         en: "Latest version" },
  "update.download":   { zh: "下载更新",         en: "Download" },
  "update.failed":     { zh: "检查失败",         en: "Check failed" },
};

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>(() => {
    const stored = localStorage.getItem("modhandler-lang") as Lang | null;
    return stored === "en" ? "en" : "zh";
  });

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    localStorage.setItem("modhandler-lang", l);
  }, []);

  const t = useCallback((key: string): string => {
    const entry = dict[key];
    if (!entry) return key;
    return entry[lang] || entry.zh || key;
  }, [lang]);

  return (
    <I18nContext.Provider value={{ lang, setLang, t }}>
      {children}
    </I18nContext.Provider>
  );
}
