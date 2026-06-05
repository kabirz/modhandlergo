import React, { useState, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Settings, Trash2 } from "lucide-react";
import { LoRaConfigService } from "../../bindings/github.com/kabirz/modhandlergo/service";

// Small button helper
const Btn = ({ children, onClick, disabled, variant = "outline" }: {
  children: React.ReactNode; onClick?: () => void; disabled?: boolean; variant?: "default" | "outline" | "ghost"
}) => (
  <Button onClick={onClick} size="sm" variant={variant} className="h-6 text-[10px] px-1.5" disabled={disabled}>
    {children}
  </Button>
);

export function LoRaConfigPage() {
  // Transport
  const [transport, setTransport] = useState<"udp" | "serial">("udp");
  const [gatewayIP, setGatewayIP] = useState("192.168.1.100");
  const [serialPort, setSerialPort] = useState("");
  const [baudRate, setBaudRate] = useState("115200");
  const [serialOpen, setSerialOpen] = useState(false);

  // Device info
  const [devMac, setDevMac] = useState("");
  const [devName, setDevName] = useState("");
  const [devSw, setDevSw] = useState("");
  const [devGwid, setDevGwid] = useState("");
  const [devCsq, setDevCsq] = useState("");

  // Network
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

  // LoRa protocol
  const [nwmode, setNwmode] = useState(0); // 0=否, 1=是
  const [ttmode, setTtmode] = useState(0); // 0=广播透传, 1=指定节点
  const [upwidText, setUpwidText] = useState("");
  const [pwr, setPwr] = useState(30); // 24-30
  const [channel, setChannel] = useState("CH1");
  const [freq, setFreq] = useState(4700);
  const [speed, setSpeed] = useState(7); // 4-11

  // AT
  const [atCmd, setAtCmd] = useState("AT+");
  const [logs, setLogs] = useState<string[]>([]);

  const addLog = useCallback((msg: string) => {
    setLogs((prev) => [...prev.slice(-200), msg]);
  }, []);

  useWailsEvent<any>("lora:device", (data) => {
    if (data?.mac) setDevMac(data.mac);
    if (data?.name) setDevName(data.name);
    if (data?.version) setDevSw(data.version);
    addLog(`设备发现: MAC=${data?.mac}, 设备=${data?.name}, SW=${data?.version}, IP=${data?.ip}`);
  });

  useWailsEvent<string>("lora:atresponse", (resp) => {
    addLog(resp);
    // TODO: parse AT response and update fields like ParseAtResponse in C code
  });

  useWailsEvent<string>("lora:log", (msg) => addLog(msg));

  useWailsEvent<any>("lora:netparams", (params) => {
    if (params?.ip) setNetIP(params.ip);
    if (params?.mask) setNetMask(params.mask);
    if (params?.gateway) setNetGW(params.gateway);
    addLog(`网络参数: IP=${params?.ip}, 掩码=${params?.mask}, 网关=${params?.gateway}`);
  });

  const sendAT = async (cmd: string) => {
    try {
      addLog(`发送: ${cmd}`);
      await LoRaConfigService.SendAT(cmd, gatewayIP);
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  const isMesh = nwmode === 1;
  const ttmodeOptions = isMesh ? ["广播透传", "指定节点", "主动上报"] : ["广播透传", "指定节点"];

  return (
    <div className="space-y-2 text-xs">
      {/* Group 0: Transport */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">连接方式</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 flex items-center gap-2 flex-wrap">
          <span className="text-muted-foreground">传输方式:</span>
          <select value={transport} onChange={(e) => setTransport(e.target.value as any)} className="h-6 text-xs bg-background border border-input rounded px-1 w-24">
            <option value="udp">UDP (网络)</option>
            <option value="serial">串口 (COM)</option>
          </select>
          {transport === "serial" && (
            <>
              <span className="text-muted-foreground">COM口:</span>
              <Input value={serialPort} onChange={(e) => setSerialPort(e.target.value)} className="w-20 h-6 text-xs" placeholder="COM3" />
              <Btn>刷新</Btn>
              <span className="text-muted-foreground">波特率:</span>
              <select value={baudRate} onChange={(e) => setBaudRate(e.target.value)} className="h-6 text-xs bg-background border border-input rounded px-1 w-20">
                {["9600","19200","38400","57600","115200","230400","460800","921600"].map(b => <option key={b}>{b}</option>)}
              </select>
              <Btn onClick={() => { if (serialOpen) { setSerialOpen(false); } else { setSerialOpen(true); } }}>{serialOpen ? "关闭串口" : "打开串口"}</Btn>
              <span className={serialOpen ? "text-green-500" : "text-muted-foreground"}>{serialOpen ? "已连接" : "未连接"}</span>
            </>
          )}
        </CardContent>
      </Card>

      {/* Group 1: Device Discovery */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">设备发现</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 space-y-1">
          <div className="flex items-center gap-2">
            <Btn onClick={() => LoRaConfigService.SearchDevices()}>搜索设备</Btn>
            <Btn onClick={() => LoRaConfigService.GetNetParams(gatewayIP)}>获取网络</Btn>
            <Btn onClick={() => sendAT("AT+GWID?")}>查询GWID</Btn>
            <Btn onClick={() => sendAT("AT+CSQ?")}>查询信号</Btn>
            <Btn onClick={() => LoRaConfigService.Reboot(gatewayIP)}>重启网关</Btn>
          </div>
          <div className="flex items-center gap-4">
            <span>MAC: <span className="font-mono">{devMac || "-"}</span></span>
            <span>设备: <span className="font-mono">{devName || "-"}</span></span>
            <span>SW: <span className="font-mono">{devSw || "-"}</span></span>
            <span>GWID: <span className="font-mono">{devGwid || "-"}</span></span>
            <span>信号: <span className="font-mono">{devCsq || "-"}</span></span>
          </div>
        </CardContent>
      </Card>

      {/* Group 2: Network Settings */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">网络设置</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 space-y-1">
          {/* Row 1: DHCP + Connection mode */}
          <div className="flex items-center gap-2">
            <span>DHCP: <span className="font-mono">{dhcpText || "-"}</span></span>
            <Btn onClick={() => sendAT("AT+DHCP?")}>查询</Btn>
            <Btn onClick={() => sendAT("AT+DHCP=ON")}>开启</Btn>
            <Btn onClick={() => sendAT("AT+DHCP=OFF")}>关闭</Btn>
            <span className="ml-4">连接模式:</span>
            <select value={netOption} onChange={(e) => setNetOption(e.target.value)} className="h-6 text-xs bg-background border border-input rounded px-1 w-20">
              {["socket","serial","mqtt","ali_cloud","usr_cloud"].map(o => <option key={o}>{o}</option>)}
            </select>
            <Btn onClick={() => sendAT(`AT+OPTION=${["socket","serial","mqtt","ali_cloud","usr_cloud"].indexOf(netOption)}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+OPTION?")}>查询</Btn>
          </div>
          {/* Row 2: IP + Mask */}
          <div className="flex items-center gap-2">
            <span>IP:</span>
            <Input value={netIP} onChange={(e) => setNetIP(e.target.value)} className="w-32 h-6 text-xs font-mono" />
            <Btn onClick={() => sendAT(`AT+GWIP=${netIP}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+GWIP?")}>查询</Btn>
            <span className="ml-4">掩码:</span>
            <Input value={netMask} onChange={(e) => setNetMask(e.target.value)} className="w-32 h-6 text-xs font-mono" />
            <Btn onClick={() => sendAT(`AT+MASK=${netMask}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+MASK?")}>查询</Btn>
          </div>
          {/* Row 3: Gateway */}
          <div className="flex items-center gap-2">
            <span>网关:</span>
            <Input value={netGW} onChange={(e) => setNetGW(e.target.value)} className="w-32 h-6 text-xs font-mono" />
            <Btn onClick={() => sendAT(`AT+GW=${netGW}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+GW?")}>查询</Btn>
          </div>
          {/* Row 4: SOCKEN */}
          <div className="flex items-center gap-2">
            <span>SOCKEN:</span>
            <span>A:</span>
            <select value={socken} onChange={(e) => setSocken(e.target.value)} className="h-6 text-xs bg-background border border-input rounded px-1 w-14">
              <option>ON</option><option>OFF</option>
            </select>
            <Btn onClick={() => sendAT(`AT+SOCKEN=${socken},OFF`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+SOCKEN?")}>查询</Btn>
          </div>
          {/* Row 5: SOCKA */}
          <div className="flex items-center gap-2">
            <span>SOCKA:</span>
            <select value={sockaMode} onChange={(e) => setSockaMode(e.target.value)} className="h-6 text-xs bg-background border border-input rounded px-1 w-16">
              <option>TCPC</option><option>TCPS</option>
            </select>
            <span>IP:</span>
            <Input value={sockaIP} onChange={(e) => setSockaIP(e.target.value)} className="w-32 h-6 text-xs font-mono" />
            <span>远程端口:</span>
            <Input value={sockaRPort} onChange={(e) => setSockaRPort(e.target.value)} className="w-14 h-6 text-xs font-mono" />
            <span>本地端口:</span>
            <Input value={sockaLPort} onChange={(e) => setSockaLPort(e.target.value)} className="w-14 h-6 text-xs font-mono" />
            <Btn onClick={() => sendAT(`AT+SOCKA=${sockaMode},${sockaIP},${sockaRPort},${sockaLPort}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+SOCKA?")}>查询</Btn>
          </div>
        </CardContent>
      </Card>

      {/* Group 3: LoRa Protocol */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">LoRa 协议</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 space-y-1">
          {/* Row 1: NWMODE + 工作模式 */}
          <div className="flex items-center gap-2">
            <span>是否组网:</span>
            <select value={nwmode} onChange={(e) => setNwmode(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-20">
              <option value={0}>否</option><option value={1}>是</option>
            </select>
            <Btn onClick={() => sendAT(`AT+NWMODE=${nwmode}`)}>设置</Btn>
            <Btn onClick={() => sendAT("AT+NWMODE?")}>查询</Btn>
            <span className="ml-6">工作模式:</span>
            <select value={ttmode} onChange={(e) => setTtmode(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-20">
              {ttmodeOptions.map((o, i) => <option key={i} value={i}>{o}</option>)}
            </select>
            <Btn onClick={() => sendAT(`AT+${isMesh ? "WMODE" : "TTMODE"}=${ttmode}`)}>设置</Btn>
            <Btn onClick={() => sendAT(`AT+${isMesh ? "WMODE" : "TTMODE"}?`)}>查询</Btn>
          </div>
          {/* Row 2: UPWID + 功率 */}
          <div className="flex items-center gap-2">
            <span>上行携带ID:</span>
            <span className="font-mono">{upwidText || "-"}</span>
            <Btn onClick={() => sendAT("AT+UPWID?")}>查询</Btn>
            <Btn onClick={() => sendAT("AT+UPWID=ON")}>开启</Btn>
            <Btn onClick={() => sendAT("AT+UPWID=OFF")}>关闭</Btn>
            <span className="ml-6">功率:</span>
            <select value={pwr} onChange={(e) => setPwr(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14">
              {[24,25,26,27,28,29,30].map(p => <option key={p} value={p}>{p}</option>)}
            </select>
            <Btn onClick={() => sendAT(`AT+PWR${channel === "CH1" ? "1" : "2"}=${pwr}`)}>设置</Btn>
            <Btn onClick={() => sendAT(`AT+PWR${channel === "CH1" ? "1" : "2"}?`)}>查询</Btn>
          </div>
          {/* Row 3: 通道 + 频率 + 速度 */}
          <div className="flex items-center gap-2">
            <span>通道:</span>
            <select value={channel} onChange={(e) => setChannel(e.target.value)} className="h-6 text-xs bg-background border border-input rounded px-1 w-14">
              <option>CH1</option><option>CH2</option>
            </select>
            <span>频率:</span>
            <select value={freq} onChange={(e) => setFreq(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14">
              {[4100,4200,4300,4400,4500,4600,4700,4800,4900,5000,5100].map(f => <option key={f} value={f}>{f}</option>)}
            </select>
            <Btn onClick={() => sendAT(`AT+CH${channel === "CH1" ? "1" : "2"}=${freq}`)}>设置</Btn>
            <Btn onClick={() => sendAT(`AT+CH${channel === "CH1" ? "1" : "2"}?`)}>查询</Btn>
            <span className="ml-4">速度:</span>
            <select value={speed} onChange={(e) => setSpeed(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14">
              {[4,5,6,7,8,9,10,11].map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <Btn onClick={() => sendAT(`AT+SPD${channel === "CH1" ? "1" : "2"}=${speed}`)}>设置</Btn>
            <Btn onClick={() => sendAT(`AT+SPD${channel === "CH1" ? "1" : "2"}?`)}>查询</Btn>
          </div>
        </CardContent>
      </Card>

      {/* Group 4: AT Command */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">AT 命令</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 flex items-center gap-2">
          <Input value={atCmd} onChange={(e) => setAtCmd(e.target.value)} className="w-64 h-6 text-xs font-mono"
            onKeyDown={(e) => e.key === "Enter" && sendAT(atCmd)} />
          <Btn onClick={() => sendAT(atCmd)} variant="default">发送</Btn>
          <Btn onClick={() => sendAT("AT+VER?")}>查询版本</Btn>
        </CardContent>
      </Card>

      {/* Group 5: Log */}
      <Card>
        <CardHeader className="pb-0 pt-1.5">
          <div className="flex items-center justify-between">
            <CardTitle className="text-xs">响应日志</CardTitle>
            <Button variant="ghost" size="icon" onClick={() => setLogs([])} title="清除" className="h-5 w-5">
              <Trash2 className="h-3 w-3" />
            </Button>
          </div>
        </CardHeader>
        <CardContent className="pt-1 pb-2">
          <div className="h-36 overflow-y-auto bg-terminal-bg rounded-md p-2 font-mono text-[11px] text-terminal-fg">
            {logs.length === 0 ? <span className="text-muted-foreground">等待响应...</span> : logs.map((l, i) => <div key={i}>{l}</div>)}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
