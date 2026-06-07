import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Settings, Trash2, Search, Cable, Unplug, Zap, Globe, Radio } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { LoRaConfigService, CANUpgradeService } from "../../bindings/github.com/kabirz/modhandlergo/service";

// Compact button for inline use
const Btn = ({ children, onClick, disabled, variant = "outline", title }: {
  children: React.ReactNode; onClick?: () => void; disabled?: boolean; variant?: "default" | "outline" | "ghost" | "destructive"; title?: string
}) => (
  <Button onClick={onClick} size="sm" variant={variant} className="h-6 text-[10px] px-1.5 shrink-0" disabled={disabled} title={title}>
    {children}
  </Button>
);

// Inline select
const Sel = ({ value, onChange, options, className = "w-16" }: {
  value: string | number; onChange: (v: string) => void; options: { value: string | number; label: string }[]; className?: string
}) => (
  <select value={value} onChange={(e) => onChange(e.target.value)}
    className={`h-6 text-[11px] bg-background border border-input rounded px-1 shrink-0 ${className}`}>
    {options.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
  </select>
);

// Section header with icon
const SectionHead = ({ icon, title }: { icon: React.ReactNode; title: string }) => (
  <div className="flex items-center gap-1.5 mb-2 pb-1.5 border-b border-border/30">
    {icon}
    <span className="text-xs font-semibold text-foreground/80">{title}</span>
  </div>
);

// Label + value row with set/query buttons
const SettingRow = ({ label, children, className = "w-14" }: {
  label: string; children: React.ReactNode; className?: string;
}) => (
  <div className="flex items-center gap-2">
    <span className={`text-[10px] text-muted-foreground text-right shrink-0 ${className}`}>{label}</span>
    {children}
  </div>
);

export function LoRaConfigPage() {
  const { t } = useI18n();
  const [transport, setTransport] = useState<"udp" | "serial">("udp");
  const [gatewayIP, setGatewayIP] = useState("192.168.1.100");
  const [serialPorts, setSerialPorts] = useState<{ portName: string; friendlyName: string }[]>([]);
  const [selectedPort, setSelectedPort] = useState("");
  const [baudRate, setBaudRate] = useState("115200");
  const [serialOpen, setSerialOpen] = useState(false);
  const [devMac, setDevMac] = useState("");
  const [devName, setDevName] = useState("");
  const [devSw, setDevSw] = useState("");
  const [devGwid, setDevGwid] = useState("");
  const [devCsq, setDevCsq] = useState("");
  const [dhcpText, setDhcpText] = useState("");
  const [netOption, setNetOption] = useState("socket");
  const [netIP, setNetIP] = useState("192.168.1.100");
  const [netMask, setNetMask] = useState("255.255.255.0");
  const [netGW, setNetGW] = useState("192.168.1.1");
  const [socken, setSocken] = useState("ON");
  const [sockaMode, setSockaMode] = useState("TCPC");
  const [sockaIP, setSockaIP] = useState("192.168.1.100");
  const [sockaRPort, setSockaRPort] = useState("1883");
  const [sockaLPort, setSockaLPort] = useState("1234");
  const [nwmode, setNwmode] = useState(0);
  const [ttmode, setTtmode] = useState(0);
  const [wmode, setWmode] = useState(0);
  const [upwidText, setUpwidText] = useState("");
  const [pwr, setPwr] = useState(30);
  const [channel, setChannel] = useState("CH1");
  const [freq, setFreq] = useState(4700);
  const [speed, setSpeed] = useState(7);
  const [atCmd, setAtCmd] = useState("AT+");
  const [logs, setLogs] = useState<{ id: number; text: string }[]>([]);
  let logIdCounter = 0;

  const addLog = useCallback((msg: string) => { setLogs((prev) => [...prev.slice(-200), { id: logIdCounter++, text: msg }]); }, []);

  useWailsEvent<any>("lora:device", (data) => {
    if (data?.mac) setDevMac(data.mac);
    if (data?.name) setDevName(data.name);
    if (data?.version) setDevSw(data.version);
    addLog(`Device found: MAC=${data?.mac}, Name=${data?.name}, SW=${data?.version}, IP=${data?.ip}`);
  });
  useWailsEvent<string>("lora:atresponse", (resp) => addLog(resp));
  useWailsEvent<string>("lora:gwid", (gwid) => setDevGwid(gwid));
  useWailsEvent<any>("lora:log", (data) => {
    if (typeof data === "string") { addLog(data); return; }
    if (data?.src === 1 || data?.src === 2) addLog(data.msg);
  });
  useWailsEvent<any>("lora:netparams", (params) => {
    if (params?.ip) setNetIP(params.ip);
    if (params?.mask) setNetMask(params.mask);
    if (params?.gateway) setNetGW(params.gateway);
    addLog(`Network params: IP=${params?.ip}, Mask=${params?.mask}, GW=${params?.gateway}`);
  });

  useWailsEvent<string>("lora:nwmode", (v) => { const n = parseInt(v); if (!isNaN(n)) setNwmode(n); });
  useWailsEvent<string>("lora:ttmode", (v) => { const n = parseInt(v); if (!isNaN(n)) setTtmode(n); });
  useWailsEvent<string>("lora:wmode", (v) => { const n = parseInt(v); if (!isNaN(n)) setWmode(n); });
  useWailsEvent<string>("lora:dhcp", (v) => setDhcpText(v));
  useWailsEvent<string>("lora:option", (v) => {
    const opts = ["socket", "serial", "mqtt", "ali_cloud", "usr_cloud"];
    const n = parseInt(v);
    if (!isNaN(n) && n >= 0 && n < opts.length) setNetOption(opts[n]);
  });
  useWailsEvent<string>("lora:upwid", (v) => setUpwidText(v));
  useWailsEvent<string>("lora:netip", (v) => setNetIP(v));
  useWailsEvent<string>("lora:netmask", (v) => setNetMask(v));
  useWailsEvent<string>("lora:netgw", (v) => setNetGW(v));
  useWailsEvent<string>("lora:csq", (v) => setDevCsq(v));
  useWailsEvent<string>("lora:chfreq", (v) => { const n = parseInt(v); if (!isNaN(n)) setFreq(n); });
  useWailsEvent<string>("lora:spd", (v) => { const n = parseInt(v); if (!isNaN(n)) setSpeed(n); });
  useWailsEvent<string>("lora:pwr", (v) => { const n = parseInt(v); if (!isNaN(n)) setPwr(n); });
  useWailsEvent<string>("lora:socka", (v) => {
    const parts = v.split(",");
    if (parts.length >= 4) {
      setSockaMode(parts[0]);
      setSockaIP(parts[1]);
      setSockaRPort(parts[2]);
      setSockaLPort(parts[3]);
    }
  });
  useWailsEvent<string>("lora:socken", (v) => {
    const parts = v.split(",");
    if (parts.length >= 1) setSocken(parts[0]);
  });

  const sendAT = async (cmd: string) => {
    try { addLog(`Sent: ${cmd}`); await LoRaConfigService.SendAT(cmd, gatewayIP); }
    catch (err: any) { addLog(`Error: ${err.message || err}`); }
  };

  const isMesh = nwmode === 1;
  // 不组网模式: TTMODE 0=广播透传 / 1=指定节点
  // 组网模式:   WMODE  0=广播透传 / 1=指定节点 / 2=主动上报
  const workModeOptions = isMesh
    ? [{ value: 0, label: t("cfg.broadcast") }, { value: 1, label: t("cfg.targetNode") }, { value: 2, label: t("cfg.activeReport") }]
    : [{ value: 0, label: t("cfg.broadcast") }, { value: 1, label: t("cfg.targetNode") }];
  const workModeCmd = isMesh ? "WMODE" : "TTMODE";
  const workModeVal = isMesh ? wmode : ttmode;
  const ch = channel === "CH1" ? "1" : "2";

  return (
    <div className="space-y-2">
      {/* Row 1: Transport + Device Discovery */}
      <div className="grid grid-cols-2 gap-2">
        {/* Transport — pill toggle like FirmwareUpgradePage */}
        <div className="p-2 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Settings className="h-3.5 w-3.5" />} title={t("cfg.transport")} />
          <div className="flex items-center gap-2 flex-wrap">
            {/* Toggle */}
            <div className="relative flex items-center bg-muted rounded-md p-0.5 h-6 w-44 shrink-0">
              <div className={`absolute top-0.5 bottom-0.5 w-1/2 bg-primary rounded-[4px] transition-transform duration-200 ${transport === "serial" ? "translate-x-full" : "translate-x-0"}`} />
              <button onClick={() => { setTransport("udp"); LoRaConfigService.SetATTransport(0); }}
                className={`relative z-10 flex-1 text-[11px] font-medium text-center rounded transition-colors cursor-pointer ${transport === "udp" ? "text-primary-foreground" : "text-muted-foreground"}`}>
                {t("cfg.udp")}
              </button>
              <button onClick={() => {
                setTransport("serial");
                LoRaConfigService.SetATTransport(1);
                // Auto-fetch serial ports when switching to serial
                CANUpgradeService.DetectSerialPorts().then((ports) => {
                  setSerialPorts(ports);
                  if (ports.length > 0 && !selectedPort) setSelectedPort(ports[0].portName);
                }).catch(() => {});
              }}
                className={`relative z-10 flex-1 text-[11px] font-medium text-center rounded transition-colors cursor-pointer ${transport === "serial" ? "text-primary-foreground" : "text-muted-foreground"}`}>
                {t("cfg.serial")}
              </button>
            </div>
            {/* Serial options — inline when serial selected */}
            {transport === "serial" && (
              <>
                <Sel value={selectedPort} onChange={setSelectedPort} className="w-24 shrink-0"
                  options={serialPorts.map((p) => ({ value: p.portName, label: p.friendlyName || p.portName }))} />
                <span className="text-[10px] text-muted-foreground shrink-0">{t("cfg.baud")}:</span>
                <Sel value={baudRate} onChange={setBaudRate} className="w-16 shrink-0"
                  options={["9600","19200","38400","57600","115200","230400","460800","921600"].map(b => ({ value: b, label: b }))} />
                <Btn onClick={async () => {
                  if (serialOpen) {
                    LoRaConfigService.SerialClose();
                    setSerialOpen(false);
                  } else {
                    if (!selectedPort) { addLog("Please select a serial port"); return; }
                    try {
                      // Always close first to avoid "already open" state
                      LoRaConfigService.SerialClose();
                      await LoRaConfigService.SerialOpen(selectedPort, parseInt(baudRate));
                      const isOpen = await LoRaConfigService.SerialIsOpen();
                      setSerialOpen(isOpen);
                      if (isOpen) addLog(`Serial port ${selectedPort} opened`);
                    } catch (err: any) { addLog(`Error: ${err?.message || err}`); }
                  }
                }} variant={serialOpen ? "destructive" : "default"}>
                  {serialOpen ? <><Unplug className="h-2.5 w-2.5 mr-0.5" />{t("cfg.close")}</> : <><Cable className="h-2.5 w-2.5 mr-0.5" />{t("cfg.open")}</>}
                </Btn>
              </>
            )}
          </div>
        </div>

        {/* Device Discovery */}
        <div className="p-2 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Search className="h-3.5 w-3.5" />} title={t("cfg.discovery")} />
          {/* Buttons evenly distributed */}
          <div className="grid grid-cols-4 gap-1.5 mb-2">
            <Btn onClick={() => LoRaConfigService.SearchDevices()}><Search className="h-2.5 w-2.5 mr-0.5" />{t("cfg.search")}</Btn>
            <Btn onClick={() => LoRaConfigService.GetNetParams(gatewayIP)}>{t("cfg.getNet")}</Btn>
            <Btn onClick={() => sendAT("AT+GWID?")}>{t("cfg.queryGwid")}</Btn>
            <Btn onClick={() => LoRaConfigService.Reboot(gatewayIP)}>{t("cfg.reboot")}</Btn>
          </div>
          {/* Device info — single row, spread evenly */}
          <div className="flex items-center justify-between text-[11px]">
            {[
              { l: "MAC", v: devMac }, { l: "Name", v: devName },
              { l: "SW", v: devSw }, { l: "GWID", v: devGwid },
            ].map(({ l, v }) => (
              <div key={l} className="flex items-center gap-1 min-w-0 flex-1">
                <span className="text-muted-foreground shrink-0">{l}:</span>
                <span className="font-mono text-foreground truncate">{v || "-"}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Row 2: Network + LoRa Protocol */}
      <div className="grid grid-cols-2 gap-2">
        {/* Network */}
        <div className="p-2 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Globe className="h-3.5 w-3.5" />} title={t("cfg.network")} />
          <div className="space-y-1.5">
            {/* DHCP */}
            <SettingRow label="DHCP:">
              <span className="font-mono text-[11px] min-w-[32px]">{dhcpText || "-"}</span>
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT("AT+DHCP?")}>{t("cfg.query")}</Btn>
                <Btn onClick={() => sendAT("AT+DHCP=ON")}>{t("cfg.enable")}</Btn>
                <Btn onClick={() => sendAT("AT+DHCP=OFF")}>{t("cfg.disable")}</Btn>
              </div>
            </SettingRow>
            {/* Mode */}
            <SettingRow label={`${t("cfg.mode")}:`}>
              <Sel value={netOption} onChange={setNetOption} className="w-20"
                options={["socket","serial","mqtt","ali_cloud","usr_cloud"].map(o => ({ value: o, label: o }))} />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+OPTION=${["socket","serial","mqtt","ali_cloud","usr_cloud"].indexOf(netOption)}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+OPTION?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* IP */}
            <SettingRow label="IP:">
              <Input value={netIP} onChange={(e) => setNetIP(e.target.value)} className="w-32 h-6 text-[11px] font-mono" />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+GWIP=${netIP}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+GWIP?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* Mask */}
            <SettingRow label={`${t("cfg.mask")}:`}>
              <Input value={netMask} onChange={(e) => setNetMask(e.target.value)} className="w-32 h-6 text-[11px] font-mono" />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+MASK=${netMask}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+MASK?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* Gateway */}
            <SettingRow label={`${t("cfg.gateway")}:`}>
              <Input value={netGW} onChange={(e) => setNetGW(e.target.value)} className="w-32 h-6 text-[11px] font-mono" />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+GW=${netGW}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+GW?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* SOCKEN */}
            <SettingRow label="SOCKEN:">
              <span className="text-[10px] text-muted-foreground">A:</span>
              <Sel value={socken} onChange={setSocken} className="w-14"
                options={[{ value: "ON", label: "ON" }, { value: "OFF", label: "OFF" }]} />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+SOCKEN=${socken},OFF`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+SOCKEN?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* SOCKA */}
            <SettingRow label="SOCKA:" className="w-14">
              <Sel value={sockaMode} onChange={setSockaMode} className="w-16"
                options={[{ value: "TCPC", label: "TCPC" }, { value: "TCPS", label: "TCPS" }]} />
              <Input value={sockaIP} onChange={(e) => setSockaIP(e.target.value)} className="w-32 h-6 text-[11px] font-mono" />
              <Input value={sockaRPort} onChange={(e) => setSockaRPort(e.target.value)} className="w-14 h-6 text-[11px] font-mono" placeholder="Remote" />
              <Input value={sockaLPort} onChange={(e) => setSockaLPort(e.target.value)} className="w-14 h-6 text-[11px] font-mono" placeholder="Local" />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+SOCKA=${sockaMode},${sockaIP},${sockaRPort},${sockaLPort}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+SOCKA?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
          </div>
        </div>

        {/* LoRa Protocol */}
        <div className="p-2 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Radio className="h-3.5 w-3.5" />} title={t("cfg.loraProto")} />
          <div className="space-y-1.5">
            {/* Mesh */}
            <SettingRow label={`${t("cfg.mesh")}:`} className="w-14">
              <Sel value={nwmode} onChange={(v) => setNwmode(Number(v))} className="w-12"
                options={[{ value: 0, label: t("cfg.meshNo") }, { value: 1, label: t("cfg.meshYes") }]} />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={async () => {
                  await sendAT(`AT+NWMODE=${nwmode}`);
                  // 组网模式切换后自动查询对应的工作模式
                  sendAT(`AT+${nwmode === 1 ? "WMODE" : "TTMODE"}?`);
                }}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+NWMODE?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* Work Mode */}
            <SettingRow label={`${t("cfg.workMode")}:`} className="w-14">
              <Sel value={workModeVal} onChange={(v) => { const n = Number(v); if (isMesh) setWmode(n); else setTtmode(n); }} className="w-20" options={workModeOptions} />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+${workModeCmd}=${workModeVal}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT(`AT+${workModeCmd}?`)}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* UPWID */}
            <SettingRow label={`${t("cfg.upwid")}:`} className="w-14">
              <Sel value={upwidText === "ON" ? "ON" : "OFF"} onChange={(v) => setUpwidText(v)} className="w-16"
                options={[{ value: "ON", label: t("cfg.upwidOn") }, { value: "OFF", label: t("cfg.upwidOff") }]} />
              <div className="flex items-center gap-0.5 ml-auto">
                <Btn onClick={() => sendAT(`AT+UPWID=${upwidText}`)}>{t("cfg.set")}</Btn>
                <Btn onClick={() => sendAT("AT+UPWID?")}>{t("cfg.query")}</Btn>
              </div>
            </SettingRow>
            {/* Channel parameters group */}
            <div className="p-1.5 rounded-md border border-border/40 bg-muted/30 space-y-1.5">
              {/* Channel toggle + label */}
              <div className="flex items-center gap-2">
                <span className="text-[10px] text-muted-foreground font-medium">{t("cfg.channel")}:</span>
                <div className="relative flex items-center bg-muted rounded-md p-0.5 h-6 w-20 shrink-0">
                  <div className={`absolute top-0.5 bottom-0.5 w-1/2 bg-primary rounded-[3px] transition-transform duration-200 ${channel === "CH2" ? "translate-x-full" : "translate-x-0"}`} />
                  <button onClick={() => setChannel("CH1")}
                    className={`relative z-10 flex-1 text-[10px] font-medium text-center rounded transition-colors cursor-pointer ${channel === "CH1" ? "text-primary-foreground" : "text-muted-foreground"}`}>
                    CH1
                  </button>
                  <button onClick={() => setChannel("CH2")}
                    className={`relative z-10 flex-1 text-[10px] font-medium text-center rounded transition-colors cursor-pointer ${channel === "CH2" ? "text-primary-foreground" : "text-muted-foreground"}`}>
                    CH2
                  </button>
                </div>
              </div>
              {/* Freq */}
              <SettingRow label={`${t("cfg.freq")}:`} className="w-14">
                <Sel value={freq} onChange={(v) => setFreq(Number(v))} className="w-14"
                  options={[4100,4200,4300,4400,4500,4600,4700,4800,4900,5000,5100].map(f => ({ value: f, label: String(f) }))} />
                <span className="text-[10px] text-muted-foreground shrink-0">×100KHz</span>
                <div className="flex items-center gap-0.5 ml-auto">
                  <Btn onClick={() => sendAT(`AT+CH${ch}=${freq}`)}>{t("cfg.set")}</Btn>
                  <Btn onClick={() => sendAT(`AT+CH${ch}?`)}>{t("cfg.query")}</Btn>
                </div>
              </SettingRow>
              {/* Speed */}
              <SettingRow label={`${t("cfg.speed")}:`} className="w-14">
                <Sel value={speed} onChange={(v) => setSpeed(Number(v))} className="w-12"
                  options={[4,5,6,7,8,9,10,11].map(s => ({ value: s, label: String(s) }))} />
                <div className="flex items-center gap-0.5 ml-auto">
                  <Btn onClick={() => sendAT(`AT+SPD${ch}=${speed}`)}>{t("cfg.set")}</Btn>
                  <Btn onClick={() => sendAT(`AT+SPD${ch}?`)}>{t("cfg.query")}</Btn>
                </div>
              </SettingRow>
              {/* Power */}
              <SettingRow label={`${t("cfg.power")}:`} className="w-14">
                <Sel value={pwr} onChange={(v) => setPwr(Number(v))} className="w-12"
                  options={[24,25,26,27,28,29,30].map(p => ({ value: p, label: String(p) }))} />
                <div className="flex items-center gap-0.5 ml-auto">
                  <Btn onClick={() => sendAT(`AT+PWR${ch}=${pwr}`)}>{t("cfg.set")}</Btn>
                  <Btn onClick={() => sendAT(`AT+PWR${ch}?`)}>{t("cfg.query")}</Btn>
                </div>
              </SettingRow>
            </div>
          </div>
        </div>
      </div>

      {/* AT Command */}
      <div className="p-2 rounded-lg bg-card border border-border/50 flex items-center gap-2">
        <Zap className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        <Input value={atCmd} onChange={(e) => setAtCmd(e.target.value)} className="flex-1 h-7 text-xs font-mono"
          onKeyDown={(e) => e.key === "Enter" && sendAT(atCmd)} />
        <Btn onClick={() => sendAT(atCmd)} variant="default">{t("lora.send")}</Btn>
        <Btn onClick={() => sendAT("AT+VER?")}>{t("cfg.queryVer")}</Btn>
      </div>

      {/* Log */}
      <div className="p-2 rounded-lg bg-card border border-border/50">
        <div className="flex items-center justify-between mb-1">
          <span className="text-xs font-medium text-foreground/80">{t("cfg.responseLog")}</span>
          <Button variant="ghost" size="icon" onClick={() => setLogs([])} title={t("lora.clear")} className="h-5 w-5">
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
        <div className="overflow-y-auto bg-terminal-bg rounded-md p-2.5 font-mono text-[11px] text-terminal-fg leading-relaxed terminal-selectable terminal-sm">
          {logs.length === 0 ? <span className="text-muted-foreground/50">{t("cfg.waitResp")}</span> : logs.map((l) => <div key={l.id}>{l.text}</div>)}
        </div>
      </div>
    </div>
  );
}
