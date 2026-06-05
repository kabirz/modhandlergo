import React, { useState, useCallback, useRef, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Radio, Trash2, Save } from "lucide-react";
import { LoRaDataService } from "../../bindings/github.com/kabirz/modhandlergo/service";

interface HistoryEntry {
  time: string;
  nid: string;
  type: string;
  data: string;
}

export function LoRaDataPage() {
  // Connection
  const [ip, setIp] = useState("192.168.2.100");
  const [port, setPort] = useState("1234");
  const [connected, setConnected] = useState(false);
  const [nid, setNid] = useState("00000000");
  const [testMode, setTestMode] = useState(false);

  // Telemetry
  const [xAngle, setXAngle] = useState("--");
  const [yAngle, setYAngle] = useState("--");
  const [btnState, setBtnState] = useState("--");
  const [rxCount, setRxCount] = useState(0);
  const [txCount, setTxCount] = useState(0);
  const [errCount, setErrCount] = useState(0);

  // Raw log
  const [logLines, setLogLines] = useState<string[]>([]);

  // Send
  const [sendData, setSendData] = useState("");

  // History
  const [history, setHistory] = useState<HistoryEntry[]>([]);

  const logRef = useRef<HTMLDivElement>(null);
  const historyRef = useRef<HTMLDivElement>(null);

  const addLog = useCallback((msg: string) => {
    const ts = new Date().toLocaleTimeString("zh-CN", { hour12: false });
    setLogLines((prev) => [...prev.slice(-500), `[${ts}] ${msg}`]);
  }, []);

  useEffect(() => {
    if (logRef.current) logRef.current.scrollTop = logRef.current.scrollHeight;
  }, [logLines]);

  useEffect(() => {
    if (historyRef.current) historyRef.current.scrollTop = historyRef.current.scrollHeight;
  }, [history]);

  useWailsEvent<number>("lora:connstate", (state) => {
    setConnected(state === 2);
    const labels = ["连接", "连接中", "断开"];
    addLog(`连接状态: ${labels[state] || "未知"}`);
  });

  useWailsEvent<any>("lora:scanner", (data) => {
    if (data) {
      setRxCount((c) => c + 1);
      const ts = new Date().toLocaleTimeString("zh-CN", { hour12: false });

      // Parse telemetry from scanner data
      if (data.overbreakValid) {
        setXAngle(`${(data.overbreak / 10).toFixed(1)}°`);
      }
      if (data.laserValid) {
        setYAngle(`${data.laser}`);
      }

      setHistory((prev) => [...prev.slice(-499), {
        time: ts,
        nid: "—",
        type: "Telemetry",
        data: `X=${xAngle} Y=${yAngle} Laser=${data.laser || "--"}`,
      }]);
    }
  });

  useWailsEvent<string>("lora:log", (msg) => {
    addLog(msg);
  });

  const handleConnect = async () => {
    try {
      if (connected) {
        await LoRaDataService.Disconnect();
      } else {
        await LoRaDataService.Connect(ip, parseInt(port));
      }
    } catch (err: any) {
      addLog(`错误: ${err.message || err}`);
    }
  };

  const handleSend = async () => {
    const bytes = sendData.trim().split(/\s+/).map((b) => parseInt(b, 16)).filter((n) => !isNaN(n));
    if (bytes.length === 0) return;
    const nidVal = parseInt(nid, 16) || 0;
    try {
      await LoRaDataService.SendFrame(nidVal, bytes.map(b => b.toString(16).padStart(2, "0")).join(""));
      setTxCount((c) => c + 1);
    } catch (err: any) {
      addLog(`发送失败: ${err.message || err}`);
    }
  };

  const handleSaveCsv = () => {
    const header = "时间,NID,类型,数据\n";
    const rows = history.map((h) => `${h.time},${h.nid},${h.type},"${h.data}"`).join("\n");
    const blob = new Blob([header + rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "lora_history.csv";
    a.click();
    URL.revokeObjectURL(url);
    addLog(`已保存 ${history.length} 条记录到 CSV`);
  };

  const handleClear = () => {
    setLogLines([]);
    setHistory([]);
    setRxCount(0);
    setTxCount(0);
    setErrCount(0);
  };

  return (
    <div className="space-y-2 text-xs">
      {/* Group 1: Connection */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs flex items-center gap-1"><Radio className="h-3.5 w-3.5" /> 连接</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 flex items-center gap-2">
          <span className="text-muted-foreground">IP:</span>
          <Input value={ip} onChange={(e) => setIp(e.target.value)} className="w-36 h-6 text-xs font-mono" />
          <span className="text-muted-foreground">端口:</span>
          <Input value={port} onChange={(e) => setPort(e.target.value)} className="w-16 h-6 text-xs font-mono" />
          <Button onClick={handleConnect} size="sm" className="h-6 px-3" variant={connected ? "destructive" : "default"}>
            {connected ? "断开" : "连接"}
          </Button>
          <span className="text-muted-foreground ml-4">NID:</span>
          <span className="font-mono">{nid}</span>
          <label className="flex items-center gap-1 ml-4">
            <input type="checkbox" checked={testMode} onChange={(e) => setTestMode(e.target.checked)} /> 测试模式
          </label>
        </CardContent>
      </Card>

      {/* Group 2: Telemetry + Raw Log (side by side) */}
      <div className="grid grid-cols-[280px_1fr] gap-2">
        {/* Telemetry */}
        <Card>
          <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">手柄数据</CardTitle></CardHeader>
          <CardContent className="pt-1 pb-2 space-y-1.5">
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground w-14">X角度:</span>
              <span className="font-mono">{xAngle}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground w-14">Y角度:</span>
              <span className="font-mono">{yAngle}</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground w-14">按键状态:</span>
              <span className="font-mono">{btnState}</span>
            </div>
            <div className="flex items-center gap-3 pt-1 text-muted-foreground">
              <span>RX: {rxCount}</span>
              <span>TX: {txCount}</span>
              <span>ERR: {errCount}</span>
            </div>
          </CardContent>
        </Card>

        {/* Raw Log */}
        <Card>
          <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">原始日志</CardTitle></CardHeader>
          <CardContent className="pt-1 pb-2">
            <div ref={logRef} className="h-40 overflow-y-auto bg-terminal-bg rounded-md p-2 font-mono text-[11px] text-terminal-fg">
              {logLines.length === 0 ? <span className="text-muted-foreground">等待日志...</span> : logLines.map((l, i) => <div key={i}>{l}</div>)}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Group 3: Operations */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">操作</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2 flex items-center gap-2">
          <Input value={sendData} onChange={(e) => setSendData(e.target.value)} placeholder="hex 数据 (空格分隔)" className="w-80 h-6 text-xs font-mono"
            onKeyDown={(e) => e.key === "Enter" && handleSend()} />
          <Button onClick={handleSend} size="sm" className="h-6 px-3">发送</Button>
          <Button onClick={handleSaveCsv} size="sm" variant="outline" className="h-6">
            <Save className="h-3 w-3 mr-1" /> 保存CSV
          </Button>
          <Button onClick={handleClear} size="sm" variant="outline" className="h-6">
            <Trash2 className="h-3 w-3 mr-1" /> 清除
          </Button>
        </CardContent>
      </Card>

      {/* Group 4: History */}
      <Card>
        <CardHeader className="pb-0 pt-1.5"><CardTitle className="text-xs">历史记录</CardTitle></CardHeader>
        <CardContent className="pt-1 pb-2">
          <div ref={historyRef} className="h-48 overflow-y-auto bg-terminal-bg rounded-md text-[11px]">
            <table className="w-full">
              <thead className="sticky top-0 bg-terminal-header">
                <tr className="text-muted-foreground text-left">
                  <th className="px-2 py-0.5 w-32">时间</th>
                  <th className="px-2 py-0.5 w-24">NID</th>
                  <th className="px-2 py-0.5 w-20">类型</th>
                  <th className="px-2 py-0.5">数据</th>
                </tr>
              </thead>
              <tbody>
                {history.length === 0 ? (
                  <tr><td colSpan={4} className="px-2 py-4 text-center text-muted-foreground">等待数据...</td></tr>
                ) : history.map((h, i) => (
                  <tr key={i} className="hover:bg-muted/30">
                    <td className="px-2 py-0.5 font-mono">{h.time}</td>
                    <td className="px-2 py-0.5 font-mono">{h.nid}</td>
                    <td className="px-2 py-0.5">{h.type}</td>
                    <td className="px-2 py-0.5 font-mono">{h.data}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
