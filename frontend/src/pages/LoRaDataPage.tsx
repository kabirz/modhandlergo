import React, { useState, useCallback, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Wifi, WifiOff, Zap, Crosshair, Box, Save, Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";
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

function DataCard({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3 p-3 rounded-lg bg-card border border-border/50">
      <div className="flex items-center justify-center w-8 h-8 rounded-md bg-primary/10 text-primary">{icon}</div>
      <div>
        <div className="text-[10px] text-muted-foreground leading-tight">{label}</div>
        <div className="text-base font-mono font-semibold leading-tight text-foreground">{value}</div>
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

const TYPE_LABELS: Record<number, string> = {
  0x01: "HANDLER",
  0x02: "TEST",
  0x03: "RSSI",
};

export function LoRaDataPage() {
  const { t } = useI18n();
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

  // Connection state
  useWailsEvent<number>("lora:connstate", (state) => {
    setConnected(state === 2);
    const labels = ["已断开", "连接中", "已连接"];
    addLog(`连接状态: ${labels[state] || "未知"}`);
  });

  // All frames: update history + NID + joystick
  useWailsEvent<any>("lora:frame", (data) => {
    if (!data) return;
    setRxCount((c) => c + 1);
    const ts = new Date().toLocaleTimeString("zh-CN", { hour12: false });

    let payload: number[] = [];
    if (Array.isArray(data.payload)) {
      payload = data.payload;
    } else if (data.payload instanceof Uint8Array) {
      payload = Array.from(data.payload);
    } else if (typeof data.payload === "string") {
      try { const bytes = atob(data.payload); payload = Array.from(bytes, (c) => c.charCodeAt(0)); } catch { payload = []; }
    }

    const type = payload[0] ?? 0xFF;
    const typeLabel = TYPE_LABELS[type] || `0x${type.toString(16).toUpperCase().padStart(2, "0")}`;
    const hexData = payload.slice(1).map((b: number) => b.toString(16).toUpperCase().padStart(2, "0")).join(" ");

    setHistory((prev) => [...prev.slice(-499), {
      time: ts,
      nid: data.nid ? `0x${Number(data.nid).toString(16).toUpperCase().padStart(8, "0")}` : "—",
      type: typeLabel,
      data: hexData,
    }]);

    // Update NID
    if (data.nid) {
      setNid(Number(data.nid).toString(16).toUpperCase().padStart(8, "0"));
    }

    // Parse joystick telemetry (type 0x01, body_len==8, trailing 0xFF)
    if (type === 0x01 && payload.length >= 9) {
      const body = payload.slice(1, 9);
      if (body[5] === 0xFF && body[6] === 0xFF && body[7] === 0xFF) {
        const xSigned = new Int16Array([body[0] << 8 | body[1]])[0];
        const ySigned = new Int16Array([body[2] << 8 | body[3]])[0];
        const btn = body[4] & 0x01;
        setXAngle(`${(xSigned / 10).toFixed(1)}°`);
        setYAngle(`${(ySigned / 10).toFixed(1)}°`);
        setBtnState(btn === 0 ? "按下" : "松开");
      }
    }
  });

  useWailsEvent<any>("lora:log", (msg) => addLog(msg));

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
        <span className="text-xs text-muted-foreground w-8">{t("lora.port")}:</span>
        <Input value={port} onChange={(e) => setPort(e.target.value)} className="w-16 h-7 text-xs font-mono" />
        <Button onClick={handleConnect} size="sm" className="h-7 px-4" variant={connected ? "destructive" : "default"}>
          {connected ? <><WifiOff className="h-3 w-3 mr-1" />{t("lora.disconnect")}</> : <><Wifi className="h-3 w-3 mr-1" />{t("lora.conn")}</>}
        </Button>
        <div className="flex-1" />
        <span className="text-xs text-muted-foreground">NID:</span>
        <span className="font-mono text-xs text-foreground">{nid}</span>
        <label className="flex items-center gap-1 text-xs text-muted-foreground ml-2 cursor-pointer">
          <input type="checkbox" checked={testMode} onChange={(e) => setTestMode(e.target.checked)} className="rounded" /> {t("lora.testMode")}
        </label>
      </div>

      {/* Telemetry + Log */}
      <div className="grid grid-cols-[320px_1fr] gap-3">
        <div className="space-y-3">
          <div className="p-3 rounded-lg bg-card border border-border/50">
            <div className="text-xs font-medium text-muted-foreground mb-3">{t("lora.joystick")}</div>
            <div className="grid grid-cols-1 gap-2">
              <DataCard icon={<Crosshair className="h-4 w-4" />} label={t("lora.xAngle")} value={xAngle} />
              <DataCard icon={<Crosshair className="h-4 w-4" />} label={t("lora.yAngle")} value={yAngle} />
              <DataCard icon={<Box className="h-4 w-4" />} label={t("lora.btnState")} value={btnState} />
            </div>
            <div className="flex items-center gap-2 mt-3 pt-2 border-t border-border/30">
              <CounterBadge label="RX" count={rxCount} color="bg-success/10 text-success" />
              <CounterBadge label="TX" count={txCount} color="bg-info/10 text-info" />
              <CounterBadge label="ERR" count={errCount} color="bg-destructive/10 text-destructive" />
            </div>
          </div>
        </div>

        <div className="p-3 rounded-lg bg-card border border-border/50 flex flex-col">
          <div className="text-xs font-medium text-muted-foreground mb-2">{t("lora.rawLog")}</div>
          <div ref={logRef} className="overflow-y-auto bg-terminal-bg rounded-md p-2.5 font-mono text-[11px] text-terminal-fg leading-relaxed terminal-selectable">
            {logLines.length === 0 ? <span className="text-muted-foreground/50">{t("lora.waitLog")}</span> : logLines.map((l, i) => <div key={i}>{l}</div>)}
          </div>
        </div>
      </div>

      {/* Operations */}
      <div className="flex items-center gap-2 p-2.5 rounded-lg bg-card border border-border/50">
        <Zap className="h-3.5 w-3.5 text-muted-foreground" />
        <Input value={sendData} onChange={(e) => setSendData(e.target.value)} placeholder="hex 数据 (空格分隔)"
          className="flex-1 h-7 text-xs font-mono" onKeyDown={(e) => e.key === "Enter" && handleSend()} />
        <Button onClick={handleSend} size="sm" className="h-7 px-4">{t("lora.send")}</Button>
        <div className="w-px h-5 bg-border/50" />
        <Button onClick={handleSaveCsv} size="sm" variant="outline" className="h-7">
          <Save className="h-3 w-3 mr-1" /> CSV
        </Button>
        <Button onClick={handleClear} size="sm" variant="outline" className="h-7">
          <Trash2 className="h-3 w-3 mr-1" /> {t("lora.clear")}
        </Button>
      </div>

      {/* History */}
      <div className="p-3 rounded-lg bg-card border border-border/50 flex-1">
        <div className="text-xs font-medium text-muted-foreground mb-2">{t("lora.history")}</div>
        <div ref={historyRef} className="overflow-y-auto rounded-md text-[11px] terminal-selectable">
          <table className="w-full">
            <thead className="sticky top-0 bg-muted/80 backdrop-blur-sm">
              <tr className="text-muted-foreground text-left">
                <th className="px-2.5 py-1.5 w-32 font-medium">{t("lora.time")}</th>
                <th className="px-2.5 py-1.5 w-24 font-medium">NID</th>
                <th className="px-2.5 py-1.5 w-20 font-medium">{t("lora.type")}</th>
                <th className="px-2.5 py-1.5 font-medium">{t("lora.data")}</th>
              </tr>
            </thead>
            <tbody>
              {history.length === 0 ? (
                <tr><td colSpan={4} className="px-2.5 py-6 text-center text-muted-foreground/50">{t("lora.waitData")}</td></tr>
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
