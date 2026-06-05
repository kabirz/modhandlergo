import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Settings, Trash2, Search, RefreshCw, Wifi, WifiOff, Zap, Router, Globe, Radio, Sliders } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { LoRaConfigService } from "../../bindings/github.com/kabirz/modhandlergo/service";

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

export function LoRaConfigPage() {
  const { t } = useI18n();
  const [transport, setTransport] = useState<"udp" | "serial">("udp");
  const [gatewayIP, setGatewayIP] = useState("192.168.1.100");
  const [serialPort, setSerialPort] = useState("");
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
  const [upwidText, setUpwidText] = useState("");
  const [pwr, setPwr] = useState(30);
  const [channel, setChannel] = useState("CH1");
  const [freq, setFreq] = useState(4700);
  const [speed, setSpeed] = useState(7);
  const [atCmd, setAtCmd] = useState("AT+");
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = useCallback((msg: string) => { setLogs((prev) => [...prev.slice(-200), msg]); }, []);

  useWailsEvent<any>("lora:device", (data) => {
    if (data?.mac) setDevMac(data.mac);
    if (data?.name) setDevName(data.name);
    if (data?.version) setDevSw(data.version);
    addLog(`设备发现: MAC=${data?.mac}, 设备=${data?.name}, SW=${data?.version}, IP=${data?.ip}`);
  });
  useWailsEvent<string>("lora:atresponse", (resp) => addLog(resp));
  useWailsEvent<string>("lora:log", (msg) => addLog(msg));
  useWailsEvent<any>("lora:netparams", (params) => {
    if (params?.ip) setNetIP(params.ip);
    if (params?.mask) setNetMask(params.mask);
    if (params?.gateway) setNetGW(params.gateway);
    addLog(`网络参数: IP=${params?.ip}, 掩码=${params?.mask}, 网关=${params?.gateway}`);
  });

  const sendAT = async (cmd: string) => {
    try { addLog(`发送: ${cmd}`); await LoRaConfigService.SendAT(cmd, gatewayIP); }
    catch (err: any) { addLog(`错误: ${err.message || err}`); }
  };

  const isMesh = nwmode === 1;
  const ttmodeOptions = isMesh
    ? [{ value: 0, label: t("cfg.broadcast") }, { value: 1, label: t("cfg.targetNode") }, { value: 2, label: t("cfg.activeReport") }]
    : [{ value: 0, label: t("cfg.broadcast") }, { value: 1, label: t("cfg.targetNode") }];
  const ch = channel === "CH1" ? "1" : "2";

  return (
    <div className="space-y-3">
      {/* Transport + Device Discovery */}
      <div className="grid grid-cols-2 gap-3">
        {/* Transport */}
        <div className="p-3 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Settings className="h-3.5 w-3.5" />} title={t("cfg.transport")} />
          <div className="flex items-center gap-2 flex-wrap">
            <Sel value={transport} onChange={(v) => setTransport(v as any)}
              options={[{ value: "udp", label: t("cfg.udp") }, { value: "serial", label: t("cfg.serial") }]} className="w-28" />
            {transport === "udp" ? null : (
              <>
                <Input value={serialPort} onChange={(e) => setSerialPort(e.target.value)} className="w-20 h-7 text-xs" placeholder="COM3" />
                <Btn><RefreshCw className="h-2.5 w-2.5" /></Btn>
                <Sel value={baudRate} onChange={setBaudRate} className="w-20"
                  options={["9600","19200","38400","57600","115200","230400","460800","921600"].map(b => ({ value: b, label: b }))} />
                <Btn onClick={() => setSerialOpen(!serialOpen)} variant={serialOpen ? "destructive" : "default"}>
                  {serialOpen ? <><WifiOff className="h-2.5 w-2.5 mr-0.5" />{t("cfg.close")}</> : <><Wifi className="h-2.5 w-2.5 mr-0.5" />{t("cfg.open")}</>}
                </Btn>
                <span className={`text-[10px] ${serialOpen ? "text-success font-medium" : "text-muted-foreground"}`}>
                  {serialOpen ? "● " + t("cfg.connected") : "○ " + t("cfg.disconnected")}
                </span>
              </>
            )}
          </div>
        </div>

        {/* Device Discovery */}
        <div className="p-3 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Search className="h-3.5 w-3.5" />} title={t("cfg.discovery")} />
          <div className="flex items-center gap-1.5 mb-2 flex-wrap">
            <Btn onClick={() => LoRaConfigService.SearchDevices()} variant="default"><Search className="h-2.5 w-2.5 mr-0.5" />{t("cfg.search")}</Btn>
            <Btn onClick={() => LoRaConfigService.GetNetParams(gatewayIP)}>{t("cfg.getNet")}</Btn>
            <Btn onClick={() => sendAT("AT+GWID?")}>{t("cfg.queryGwid")}</Btn>
            <Btn onClick={() => LoRaConfigService.Reboot(gatewayIP)}>{t("cfg.reboot")}</Btn>
          </div>
          <div className="grid grid-cols-4 gap-x-4 gap-y-0.5 text-[11px]">
            {[
              { l: "MAC", v: devMac }, { l: "设备", v: devName }, { l: "SW", v: devSw },
              { l: "GWID", v: devGwid },
            ].map(({ l, v }) => (
              <div key={l} className="flex items-center gap-1">
                <span className="text-muted-foreground">{l}:</span>
                <span className="font-mono text-foreground truncate">{v || "-"}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Network + Protocol */}
      <div className="grid grid-cols-2 gap-3">
        {/* Network */}
        <div className="p-3 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Globe className="h-3.5 w-3.5" />} title={t("cfg.network")} />
          <div className="space-y-1.5">
            {/* DHCP */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">DHCP:</span>
              <span className="font-mono text-[11px] min-w-[24px]">{dhcpText || "-"}</span>
              <Btn onClick={() => sendAT("AT+DHCP?")}>{t("cfg.query")}</Btn>
              <Btn onClick={() => sendAT("AT+DHCP=ON")}>{t("cfg.enable")}</Btn>
              <Btn onClick={() => sendAT("AT+DHCP=OFF")}>{t("cfg.disable")}</Btn>
            </div>
            {/* Connection mode */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">{t("cfg.mode")}:</span>
              <Sel value={netOption} onChange={setNetOption} className="w-16"
                options={["socket","serial","mqtt","ali_cloud","usr_cloud"].map(o => ({ value: o, label: o }))} />
              <Btn onClick={() => sendAT(`AT+OPTION=${["socket","serial","mqtt","ali_cloud","usr_cloud"].indexOf(netOption)}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+OPTION?")}>{t("cfg.query")}</Btn>
            </div>
            {/* IP + Mask */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">IP:</span>
              <Input value={netIP} onChange={(e) => setNetIP(e.target.value)} className="w-28 h-6 text-[11px] font-mono" />
              <Btn onClick={() => sendAT(`AT+GWIP=${netIP}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+GWIP?")}>{t("cfg.query")}</Btn>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">{t("cfg.mask")}:</span>
              <Input value={netMask} onChange={(e) => setNetMask(e.target.value)} className="w-28 h-6 text-[11px] font-mono" />
              <Btn onClick={() => sendAT(`AT+MASK=${netMask}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+MASK?")}>{t("cfg.query")}</Btn>
            </div>
            {/* Gateway */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">{t("cfg.gateway")}:</span>
              <Input value={netGW} onChange={(e) => setNetGW(e.target.value)} className="w-28 h-6 text-[11px] font-mono" />
              <Btn onClick={() => sendAT(`AT+GW=${netGW}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+GW?")}>{t("cfg.query")}</Btn>
            </div>
            {/* SOCKEN */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">SOCKEN:</span>
              <span className="text-[10px] text-muted-foreground">A:</span>
              <Sel value={socken} onChange={setSocken} className="w-12"
                options={[{ value: "ON", label: "ON" }, { value: "OFF", label: "OFF" }]} />
              <Btn onClick={() => sendAT(`AT+SOCKEN=${socken},OFF`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+SOCKEN?")}>{t("cfg.query")}</Btn>
            </div>
            {/* SOCKA */}
            <div className="flex items-center gap-2 flex-wrap">
              <span className="text-[10px] text-muted-foreground w-12 text-right shrink-0">SOCKA:</span>
              <Sel value={sockaMode} onChange={setSockaMode} className="w-14"
                options={[{ value: "TCPC", label: "TCPC" }, { value: "TCPS", label: "TCPS" }]} />
              <Input value={sockaIP} onChange={(e) => setSockaIP(e.target.value)} className="w-28 h-6 text-[11px] font-mono" />
              <Input value={sockaRPort} onChange={(e) => setSockaRPort(e.target.value)} className="w-12 h-6 text-[11px] font-mono" placeholder="远端" />
              <Input value={sockaLPort} onChange={(e) => setSockaLPort(e.target.value)} className="w-12 h-6 text-[11px] font-mono" placeholder="本端" />
              <Btn onClick={() => sendAT(`AT+SOCKA=${sockaMode},${sockaIP},${sockaRPort},${sockaLPort}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+SOCKA?")}>{t("cfg.query")}</Btn>
            </div>
          </div>
        </div>

        {/* LoRa Protocol */}
        <div className="p-3 rounded-lg bg-card border border-border/50">
          <SectionHead icon={<Radio className="h-3.5 w-3.5" />} title={t("cfg.loraProto")} />
          <div className="space-y-1.5">
            {/* NWMODE + TTMODE */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-14 text-right shrink-0">{t("cfg.mesh")}:</span>
              <Sel value={nwmode} onChange={(v) => setNwmode(Number(v))} className="w-12"
                options={[{ value: 0, label: t("cfg.meshNo") }, { value: 1, label: t("cfg.meshYes") }]} />
              <Btn onClick={() => sendAT(`AT+NWMODE=${nwmode}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT("AT+NWMODE?")}>{t("cfg.query")}</Btn>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-14 text-right shrink-0">{t("cfg.workMode")}:</span>
              <Sel value={ttmode} onChange={(v) => setTtmode(Number(v))} className="w-20" options={ttmodeOptions} />
              <Btn onClick={() => sendAT(`AT+${isMesh ? "WMODE" : "TTMODE"}=${ttmode}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT(`AT+${isMesh ? "WMODE" : "TTMODE"}?`)}>{t("cfg.query")}</Btn>
            </div>
            {/* UPWID */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-14 text-right shrink-0">{t("cfg.upwid")}:</span>
              <span className="font-mono text-[11px]">{upwidText || "-"}</span>
              <Btn onClick={() => sendAT("AT+UPWID?")}>{t("cfg.query")}</Btn>
              <Btn onClick={() => sendAT("AT+UPWID=ON")}>{t("cfg.enable")}</Btn>
              <Btn onClick={() => sendAT("AT+UPWID=OFF")}>{t("cfg.disable")}</Btn>
            </div>
            {/* Power */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-14 text-right shrink-0">{t("cfg.power")}:</span>
              <Sel value={pwr} onChange={(v) => setPwr(Number(v))} className="w-12"
                options={[24,25,26,27,28,29,30].map(p => ({ value: p, label: String(p) }))} />
              <Btn onClick={() => sendAT(`AT+PWR${ch}=${pwr}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT(`AT+PWR${ch}?`)}>{t("cfg.query")}</Btn>
            </div>
            {/* CH + Freq + Speed */}
            <div className="flex items-center gap-2">
              <span className="text-[10px] text-muted-foreground w-14 text-right shrink-0">{t("cfg.channel")}:</span>
              <Sel value={channel} onChange={setChannel} className="w-12"
                options={[{ value: "CH1", label: "CH1" }, { value: "CH2", label: "CH2" }]} />
              <span className="text-[10px] text-muted-foreground">{t("cfg.freq")}:</span>
              <Sel value={freq} onChange={(v) => setFreq(Number(v))} className="w-14"
                options={[4100,4200,4300,4400,4500,4600,4700,4800,4900,5000,5100].map(f => ({ value: f, label: String(f) }))} />
              <Btn onClick={() => sendAT(`AT+CH${ch}=${freq}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT(`AT+CH${ch}?`)}>{t("cfg.query")}</Btn>
              <span className="text-[10px] text-muted-foreground ml-2">{t("cfg.speed")}:</span>
              <Sel value={speed} onChange={(v) => setSpeed(Number(v))} className="w-12"
                options={[4,5,6,7,8,9,10,11].map(s => ({ value: s, label: String(s) }))} />
              <Btn onClick={() => sendAT(`AT+SPD${ch}=${speed}`)}>{t("cfg.set")}</Btn>
              <Btn onClick={() => sendAT(`AT+SPD${ch}?`)}>{t("cfg.query")}</Btn>
            </div>
          </div>
        </div>
      </div>

      {/* AT Command */}
      <div className="p-2.5 rounded-lg bg-card border border-border/50 flex items-center gap-2">
        <Zap className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
        <Input value={atCmd} onChange={(e) => setAtCmd(e.target.value)} className="flex-1 h-7 text-xs font-mono"
          onKeyDown={(e) => e.key === "Enter" && sendAT(atCmd)} />
        <Btn onClick={() => sendAT(atCmd)} variant="default">{t("lora.send")}</Btn>
        <Btn onClick={() => sendAT("AT+VER?")}>{t("cfg.queryVer")}</Btn>
      </div>

      {/* Log */}
      <div className="p-3 rounded-lg bg-card border border-border/50">
        <div className="flex items-center justify-between mb-2">
          <span className="text-xs font-medium text-foreground/80">{t("cfg.responseLog")}</span>
          <Button variant="ghost" size="icon" onClick={() => setLogs([])} title={t("lora.clear")} className="h-5 w-5">
            <Trash2 className="h-3 w-3" />
          </Button>
        </div>
        <div className="overflow-y-auto bg-terminal-bg rounded-md p-2.5 font-mono text-[11px] text-terminal-fg leading-relaxed terminal-selectable">
          {logs.length === 0 ? <span className="text-muted-foreground/50">{t("cfg.waitResp")}</span> : logs.map((l, i) => <div key={i}>{l}</div>)}
        </div>
      </div>
    </div>
  );
}
