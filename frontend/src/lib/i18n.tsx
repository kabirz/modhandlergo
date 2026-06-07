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
  "lora.operation":   { zh: "操作",         en: "Operation" },
  "lora.send":        { zh: "发送",         en: "Send" },
  "lora.clear":       { zh: "清除",         en: "Clear" },
  "lora.history":     { zh: "历史记录",     en: "History" },
  "lora.waitData":    { zh: "等待数据...",   en: "Waiting for data..." },
  "lora.time":        { zh: "时间",         en: "Time" },
  "lora.type":        { zh: "类型",         en: "Type" },
  "lora.data":        { zh: "数据",         en: "Data" },
  "lora.csv":         { zh: "CSV",          en: "CSV" },
  "lora.saved":       { zh: "已保存",       en: "Saved" },
  "lora.records":     { zh: "条记录到 CSV", en: "records to CSV" },
  "lora.connState":   { zh: "连接状态",     en: "Conn State" },
  "lora.connecting":  { zh: "连接中",       en: "Connecting" },

  // LoRa Config
  "cfg.transport":    { zh: "连接方式",     en: "Transport" },
  "cfg.udp":          { zh: "UDP (网络)",   en: "UDP (Network)" },
  "cfg.serial":       { zh: "串口 (COM)",   en: "Serial (COM)" },
  "cfg.com":          { zh: "COM口",        en: "COM Port" },
  "cfg.refresh":      { zh: "刷新",         en: "Refresh" },
  "cfg.baud":         { zh: "波特率",       en: "Baud Rate" },
  "cfg.open":         { zh: "打开",         en: "Open" },
  "cfg.close":        { zh: "关闭",         en: "Close" },
  "cfg.connected":    { zh: "已连接",       en: "Connected" },
  "cfg.disconnected": { zh: "未连接",       en: "Disconnected" },
  "cfg.discovery":    { zh: "设备发现",     en: "Device Discovery" },
  "cfg.search":       { zh: "搜索设备",     en: "Search" },
  "cfg.getNet":       { zh: "获取网络",     en: "Get Network" },
  "cfg.queryGwid":    { zh: "查询GWID",     en: "Query GWID" },
  "cfg.reboot":       { zh: "重启网关",     en: "Reboot" },
  "cfg.network":      { zh: "网络设置",     en: "Network" },
  "cfg.dhcp":         { zh: "DHCP",         en: "DHCP" },
  "cfg.query":        { zh: "查询",         en: "Query" },
  "cfg.enable":       { zh: "开启",         en: "Enable" },
  "cfg.disable":      { zh: "关闭",         en: "Disable" },
  "cfg.mode":         { zh: "模式",         en: "Mode" },
  "cfg.set":          { zh: "设置",         en: "Set" },
  "cfg.mask":         { zh: "掩码",         en: "Mask" },
  "cfg.gateway":      { zh: "网关",         en: "Gateway" },
  "cfg.socken":       { zh: "SOCKEN",       en: "SOCKEN" },
  "cfg.socka":        { zh: "SOCKA",        en: "SOCKA" },
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
  "cfg.atCmd":        { zh: "AT 命令",      en: "AT Command" },
  "cfg.queryVer":     { zh: "查询版本",     en: "Query Version" },
  "cfg.responseLog":  { zh: "响应日志",     en: "Response Log" },
  "cfg.waitResp":     { zh: "等待响应...",   en: "Waiting for response..." },

  // Firmware
  "fw.channel":       { zh: "通道选择",     en: "Channel" },
  "fw.can":           { zh: "CAN",          en: "CAN" },
  "fw.uart":          { zh: "UART",         en: "UART" },
  "fw.baud":          { zh: "波特率",       en: "Baud Rate" },
  "fw.firmwareFile":  { zh: "固件文件",     en: "Firmware File" },
  "fw.browse":        { zh: "浏览",         en: "Browse" },
  "fw.upgrade":       { zh: "升级控制",     en: "Upgrade" },
  "fw.startUpgrade":  { zh: "开始升级",     en: "Start Upgrade" },
  "fw.queryVer":      { zh: "查询版本",     en: "Query Version" },
  "fw.reboot":        { zh: "重启板卡",     en: "Reboot Board" },
  "fw.curVersion":    { zh: "当前版本",     en: "Current Version" },
  "fw.log":           { zh: "日志",         en: "Log" },
  "fw.waitOp":        { zh: "等待操作...",   en: "Waiting..." },
  "fw.selectFile":    { zh: "选择固件文件 (.bin)", en: "Select firmware file (.bin)" },
  "fw.pleaseSelect":  { zh: "请先选择固件文件", en: "Please select a firmware file first" },

  // CAN Command
  "can.frameConfig":  { zh: "帧配置",       en: "Frame Config" },
  "can.canId":        { zh: "CAN ID",       en: "CAN ID" },
  "can.format":       { zh: "格式",         en: "Format" },
  "can.standard":     { zh: "标准",         en: "Std" },
  "can.extended":     { zh: "扩展",         en: "Ext" },
  "can.frameType":    { zh: "帧类型",       en: "Type" },
  "can.dataFrame":    { zh: "数据",         en: "Data" },
  "can.remoteFrame":  { zh: "远程",         en: "Remote" },
  "can.data":         { zh: "数据",         en: "Data" },
  "can.sendFrame":    { zh: "发送帧",       en: "Send Frame" },
  "can.loraConfig":   { zh: "LoRa 配置",    en: "LoRa Config" },
  "can.powerOn":      { zh: "上电",         en: "Power On" },
  "can.powerOff":     { zh: "断电",         en: "Power Off" },
  "can.test":         { zh: "测试",         en: "Test" },
  "can.exitTest":     { zh: "退测试",       en: "Exit Test" },
  "can.busMonitor":   { zh: "总线监视器",   en: "Bus Monitor" },
  "can.autoScroll":   { zh: "自动滚动",     en: "Auto Scroll" },
  "can.time":         { zh: "时间",         en: "Time" },
  "can.label":        { zh: "标注",         en: "Label" },
  "can.dlc":          { zh: "DLC",          en: "DLC" },
  "can.waitFrame":    { zh: "等待 CAN 帧...", en: "Waiting for CAN frames..." },

  // About dialog
  "about.title":       { zh: "关于",         en: "About" },
  "about.appName":     { zh: "应用名称",     en: "App Name" },
  "about.version":     { zh: "版本",         en: "Version" },
  "about.techStack":   { zh: "技术栈",       en: "Tech Stack" },
  "about.frontend":    { zh: "前端框架",     en: "Frontend" },
  "about.canAdapter":  { zh: "CAN 适配器",   en: "CAN Adapter" },
  "about.serial":      { zh: "串口",         en: "Serial" },
  "about.protocol":    { zh: "协议",         en: "Protocol" },
  "about.origProject": { zh: "原版项目",     en: "Original Project" },
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
  "update.noRelease":  { zh: "暂无发布版本",     en: "No releases found" },

  // Common
  "common.error":     { zh: "错误",         en: "Error" },
  "common.loading":   { zh: "加载中...",     en: "Loading..." },
  "common.yes":       { zh: "是",           en: "Yes" },
  "common.no":        { zh: "否",           en: "No" },
  "common.on":        { zh: "ON",           en: "ON" },
  "common.off":       { zh: "OFF",          en: "OFF" },
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
