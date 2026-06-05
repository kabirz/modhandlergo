import React, { useState, useCallback, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Radio, Trash2, Save, Wifi, WifiOff, Zap, Crosshair, Box } from "lucide-react";
import { LoRaDataService } from "../../bindings/github.com/kabirz/modhandlergo/service";

interface HistoryEntry {
  time: string;
  nid: string;
  type: string;
  data: string;
}

function StatusDot({ active }: { active: boolean }) {
  return (
    <span className={`inline-block w-2 h-2 rounded-full ${active ? "bg-success animate-pulse" : "bg-muted-foreground/30"}`} />
  );
}

function DataCard({ icon, label, value, unit }: { icon: React.ReactNode; label: string; value: string; unit?: string }) {
  return (
    <div className="flex items-center gap-3 p-3 rounded-lg bg-card border border-border/50">
      <div className="flex items-center justify-center w-8 h-8 rounded-md bg-primary/10 text-primary">{icon}</div>
      <div>
        <div className="text-[10px] text-muted-foreground leading-tight">{label}</div>
        <div className="text-base font-mono font-semibold leading-tight text-foreground">
          {value}<span className="text-[10px] text-muted-foreground ml-0.5">{unit}</span>
        </div>
      </div>
    </div>
  );
}

function CounterBadge({ label, count, color }: { label: string; count: number; color: string }) {
  return (
    <span className={`inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono ${color}`}>
      {label}: {count}
    </span>
  );
}

export function LoRaDataPage() {
  const [ip, setIp] = useState("192.168.2.100");
  const [port, setPort] = useState("1234");
  const [connected, setConnected] = useState(false);
  const [nid, setNid] = useState("00000000");
  const [testMode, setTestMode] = useState(false);
  const [xAngle, setXAngle] = useState("--");
  const [yAngle, setYAngle] = useState("--");
  const [btnState, setBtnState] = useState("--");
  const [rxCount, setRxCount] = useState(0);
  const [txCount, setTxCount] = useState(0);
  const [errCount, setErrCount] = useState(0);
  const [logLines, setLogLines] = useState<string[]>([]);
  const [sendData, setSendData] = useState("");
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const logRef = useRef<HTMLDivElement>(null);
  const historyRef = useRef<HTMLDivElement>(null);

  const addLog = useCallback((msg: string) => {
    const ts = new Date().toLocaleTimeString("zh-CN", { hour12: false });
    setLogLines((prev) => [...prev.slice(-500), `[${ts}] ${msg}`]);
  }, []);

  useEffect(() => { if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight; }, [logLines]);
  useEffect(() => { if (historyRef.current) historyRef.current.scrollTop = historyRef.current.scrollHeight; }, [history]);

  useWailsEvent<number>("lora:connstate", (state) => {
    setConnected(state === 2);
    const labels = ["连接", "连接中", "断开"];
    addLog(`连接状态: ${labels[state] || "未知"}`);
  });

  useWailsEvent<any>("lora:scanner", (data) => {
    if (data) {
      setRxCount((c) => c + 1);
      const ts = new Date().toLocaleTimeString("zh-CN", { hour12: false });
      if (data.overbreakValid) setXAngle(`${(data.overbreak / 10).toFixed(1)}°`);
      if (data.laserValid) setYAngle(`${data.laser}`);
      setHistory((prev) => [...prev.slice(-499), {
        time: ts, nid: "—", type: "Telemetry",
        data: `X=${xAngle} Y=${yAngle} Laser=${data.laser || "--"}`,
      }]);
    }
  });

  useWailsEvent<string>("lora:log", (msg) => addLog(msg));

  const handleConnect = async () => {
    try {
      if (connected) { await LoRaDataService.Disconnect(); }
      else { await LoRaDataService.Connect(ip, parseInt(port)); }
    } catch (err: any) { addLog(`错误: ${err.message || err}`); }
  };

  const handleSend = async () => {
    const bytes = sendData.trim().split(/\s+/).map((b) => parseInt(b, 16)).filter((n) => !isNaN(n));
    if (bytes.length === 0) return;
    const nidVal = parseInt(nid, 16) || 0;
    try {
      await LoRaDataService.SendFrame(nidVal, bytes.map(b => b.toString(16).padStart(2, "0")).join(""));
      setTxCount((c) => c + 1);
    } catch (err: any) { addLog(`发送失败: ${err.message || err}`); }
  };

  const handleSaveCsv = () => {
    const header = "时间,NID,类型,数据\n";
    const rows = history.map((h) => `${h.time},${h.nid},${h.type},"${h.data}"`).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = "lora_history.csv"; a.click();
    URL.revokeObjectURL(url);
    addLog(`已保存 ${history.length} 条记录到 CSV`);
  };

  const handleClear = () => {
    setLogLines([]); setHistory([]); setRxCount(0); setTxCount(0); setErrCount(0);
  };

  return (
    <div className="space-y-3">
      {/* Connection Bar */}
      <div className="flex items-center gap-2 p-2.5 rounded-lg bg-card border border-border/50">
        <StatusDot active={connected} />
        <span className="text-xs text-muted-foreground w-6">IP:</span>
        <Input value={ip} onChange={(e) => setIp(e.target.value)} className="w-36 h-7 text-xs font-mono" />
        <span className="text-xs text-muted-foreground w-8">端口:</span>
        <Input value={port} onChange={(e) => setPort(e.target.value)} className="w-16 h-7 text-xs font-mono" />
        <Button onClick={handleConnect} size="sm" className="h-7 px-4" variant={connected ? "destructive" : "default"}>
          {connected ? <><WifiOff className="h-3 w-3 mr-1" />断开</> : <><Wifi className="h-3 w-3 mr-1" />连接</>}
        </Button>
        <div className="flex-1" />
        <span className="text-xs text-muted-foreground">NID:</span>
        <span className="font-mono text-xs text-foreground">{nid}</span>
        <label className="flex items-center gap-1 text-xs text-muted-foreground ml-2 cursor-pointer">
          <input type="checkbox" checked={testMode} onChange={(e) => setTestMode(e.target.checked)} className="rounded" /> 测试模式
        </label>
      </div>

      {/* Telemetry + Log */}
      <div className="grid grid-cols-[320px_1fr] gap-3">
        {/* Telemetry */}
        <div className="space-y-3">
          <div className="p-3 rounded-lg bg-card border border-border/50">
            <div className="text-xs font-medium text-muted-foreground mb-3">手柄数据</div>
            <div className="grid grid-cols-1 gap-2">
              <DataCard icon={<Crosshair className="h-4 w-4" />} label="X 角度" value={xAngle} />
              <DataCard icon={<Crosshair className="h-4 w-4" />} label="Y 角度" value={yAngle} />
              <DataCard icon={<Box className="h-4 w-4" />} label="按键状态" value={btnState} />
            </div>
            <div className="flex items-center gap-2 mt-3 pt-2 border-t border-border/30">
              <CounterBadge label="RX" count={rxCount} color="bg-success/10 text-success" />
              <CounterBadge label="TX" count={txCount} color="bg-info/10 text-info" />
              <CounterBadge label="ERR" count={errCount} color="bg-destructive/10 text-destructive" />
            </div>
          </div>
        </div>

        {/* Raw Log */}
        <div className="p-3 rounded-lg bg-card border border-border/50 flex flex-col">
          <div className="text-xs font-medium text-muted-foreground mb-2">原始日志</div>
          <div ref={logRef} className="flex-1 h-48 overflow-y-auto bg-terminal-bg rounded-md p-2.5 font-mono text-[11px] text-terminal-fg leading-relaxed">
            {logLines.length === 0 ? <span className="text-muted-foreground/50">等待日志...</span> : logLines.map((l, i) => <div key={i}>{l}</div>)}
          </div>
        </div>
      </div>

      {/* Operations */}
      <div className="flex items-center gap-2 p-2.5 rounded-lg bg-card border border-border/50">
        <Zap className="h-3.5 w-3.5 text-muted-foreground" />
        <Input value={sendData} onChange={(e) => setSendData(e.target.value)} placeholder="hex 数据 (空格分隔)"
          className="flex-1 h-7 text-xs font-mono" onKeyDown={(e) => e.key === "Enter" && handleSend()} />
        <Button onClick={handleSend} size="sm" className="h-7 px-4">发送</Button>
        <div className="w-px h-5 bg-border/50" />
        <Button onClick={handleSaveCsv} size="sm" variant="outline" className="h-7">
          <Save className="h-3 w-3 mr-1" /> CSV
        </Button>
        <Button onClick={handleClear} size="sm" variant="outline" className="h-7">
          <Trash2 className="h-3 w-3 mr-1" /> 清除
        </Button>
      </div>

      {/* History */}
      <div className="p-3 rounded-lg bg-card border border-border/50 flex-1">
        <div className="text-xs font-medium text-muted-foreground mb-2">历史记录</div>
        <div ref={historyRef} className="h-48 overflow-y-auto rounded-md text-[11px]">
          <table className="w-full">
            <thead className="sticky top-0 bg-muted/80 backdrop-blur-sm">
              <tr className="text-muted-foreground text-left">
                <th className="px-2.5 py-1.5 w-32 font-medium">时间</th>
                <th className="px-2.5 py-1.5 w-24 font-medium">NID</th>
                <th className="px-2.5 py-1.5 w-20 font-medium">类型</th>
                <th className="px-2.5 py-1.5 font-medium">数据</th>
              </tr>
            </thead>
            <tbody>
              {history.length === 0 ? (
                <tr><td colSpan={4} className="px-2.5 py-6 text-center text-muted-foreground/50">等待数据...</td></tr>
              ) : history.map((h, i) => (
                <tr key={i} className="border-t border-border/20 hover:bg-muted/30 transition-colors">
                  <td className="px-2.5 py-1 font-mono text-foreground/80">{h.time}</td>
                  <td className="px-2.5 py-1 font-mono text-primary">{h.nid}</td>
                  <td className="px-2.5 py-1">
                    <span className="px-1.5 py-0.5 rounded bg-primary/10 text-primary text-[10px]">{h.type}</span>
                  </td>
                  <td className="px-2.5 py-1 font-mono text-foreground/80">{h.data}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
