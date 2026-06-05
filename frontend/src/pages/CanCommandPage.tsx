import React, { useState, useCallback, useRef, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Terminal, Trash2 } from "lucide-react";
import { CANCommandService } from "../../bindings/github.com/kabirz/modhandlergo/service";

interface FrameEntry {
  id: number;
  data: number[];
  dlc: number;
  isTX: boolean;
  label: string;
  timestamp: string;
}

const FRAME_LABELS: Record<number, string> = {
  0x101: "控制命令", 0x102: "响应帧", 0x103: "固件数据",
  0x105: "LoRa配参", 0x106: "LoRa响应",
  0x1E3: "手柄状态", 0x263: "激光测距", 0x363: "X/Y坐标", 0x463: "Z坐标", 0x763: "心跳",
};

export function CanCommandPage() {
  // Frame config
  const [canId, setCanId] = useState("101");
  const [dataHex, setDataHex] = useState("00 00 00 00 00 00 00 00");
  const [isExtended, setIsExtended] = useState(false);
  const [isRemote, setIsRemote] = useState(false);

  // LoRa config
  const [loraProt, setLoraProt] = useState(1); // 0=NODE, 1=LG210, 2=LG220
  const [loraMode, setLoraMode] = useState(1); // 0=FP, 1=TRANS, 2=NET
  const [loraCh1Spd, setLoraCh1Spd] = useState(7);
  const [loraCh1Freq, setLoraCh1Freq] = useState(4800);
  const [loraCh2Spd, setLoraCh2Spd] = useState(7);
  const [loraCh2Freq, setLoraCh2Freq] = useState(4800);
  const [loraPnum, setLoraPnum] = useState(0);
  const [loraNid, setLoraNid] = useState(0);
  const [loraGwid, setLoraGwid] = useState(0);
  const [loraPowered, setLoraPowered] = useState(false);
  const [loraTestMode, setLoraTestMode] = useState(false);

  // Monitor
  const [frames, setFrames] = useState<FrameEntry[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const monitorRef = useRef<HTMLDivElement>(null);

  const addFrame = useCallback((frame: FrameEntry) => {
    setFrames((prev) => [...prev.slice(-500), frame]);
  }, []);

  useEffect(() => {
    if (monitorRef.current && autoScroll) {
      monitorRef.current.scrollTop = monitorRef.current.scrollHeight;
    }
  }, [frames, autoScroll]);

  useEffect(() => {
    CANCommandService.StartMonitor().catch(() => {});
  }, []);

  useWailsEvent<any>("can:frame", (ev) => {
    const label = FRAME_LABELS[ev.id] || "";
    addFrame({
      id: ev.id, data: ev.data || [], dlc: ev.dlc || 0,
      isTX: ev.isTx || false, label,
      timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }),
    });
  });

  const parseHexData = (hex: string): number[] => {
    return hex.trim().split(/\s+/).map((b) => parseInt(b, 16)).filter((n) => !isNaN(n));
  };

  const handleSendFrame = async () => {
    const id = parseInt(canId, 16);
    if (isNaN(id)) return;
    const bytes = parseHexData(dataHex);
    const hexStr = bytes.map((b) => b.toString(16).padStart(2, "0")).join("");
    try {
      await CANCommandService.SendFrame(id, hexStr, bytes.length, isExtended, isRemote);
      addFrame({ id, data: bytes, dlc: bytes.length, isTX: true, label: FRAME_LABELS[id] || "", timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }) });
    } catch (err: any) {
      addFrame({ id: 0, data: [], dlc: 0, isTX: true, label: `发送失败: ${err.message || err}`, timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }) });
    }
  };

  const protOptions = ["NODE", "LG210", "LG220"];
  const modeOptions = ["FP", "TRANS", "NET"];
  const spdOptions = [4, 5, 6, 7, 8, 9, 10, 11];
  const freqOptions = [4100, 4200, 4300, 4400, 4500, 4600, 4700, 4800, 4900, 5000, 5100];

  return (
    <div className="space-y-3">
      {/* Row 1: Frame Config + LoRa Config side by side */}
      <div className="grid grid-cols-[480px_1fr] gap-3">
        {/* Frame Config */}
        <Card>
          <CardHeader className="pb-1 pt-2">
            <CardTitle className="flex items-center gap-2 text-sm">
              <Terminal className="h-4 w-4" /> 帧配置
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-1.5">
            <div className="flex items-center gap-2">
              <label className="text-xs text-muted-foreground w-12 shrink-0">CAN ID:</label>
              <Input value={canId} onChange={(e) => setCanId(e.target.value)} className="w-24 h-7 text-xs font-mono" />
              <span className="text-[10px] text-muted-foreground">(Hex)</span>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-muted-foreground w-12 shrink-0">格式:</label>
              <label className="flex items-center gap-1 text-xs"><input type="radio" checked={!isExtended} onChange={() => setIsExtended(false)} /> 标准</label>
              <label className="flex items-center gap-1 text-xs"><input type="radio" checked={isExtended} onChange={() => setIsExtended(true)} /> 扩展</label>
              <span className="w-6" />
              <label className="flex items-center gap-1 text-xs"><input type="radio" checked={!isRemote} onChange={() => setIsRemote(false)} /> 数据</label>
              <label className="flex items-center gap-1 text-xs"><input type="radio" checked={isRemote} onChange={() => setIsRemote(true)} /> 远程</label>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-xs text-muted-foreground w-12 shrink-0">数据:</label>
              <Input value={dataHex} onChange={(e) => setDataHex(e.target.value)} className="flex-1 h-7 text-xs font-mono" />
            </div>
            <Button onClick={handleSendFrame} size="sm" className="w-full h-7 text-xs">发送帧</Button>
          </CardContent>
        </Card>

        {/* LoRa Config */}
        <Card>
          <CardHeader className="pb-1 pt-2">
            <div className="flex items-center justify-between">
              <CardTitle className="text-sm">LoRa 配置</CardTitle>
              <div className="flex items-center gap-1">
                <Button onClick={async () => { await CANCommandService.SendLoraCommand(0x0F, loraPowered ? "00" : "01"); setLoraPowered(!loraPowered); }} size="sm" variant={loraPowered ? "destructive" : "default"} className="h-6 text-[10px] px-2">
                  {loraPowered ? "断电" : "上电"}
                </Button>
                <Button onClick={async () => { await CANCommandService.SendLoraCommand(0x0D, loraTestMode ? "00" : "01"); setLoraTestMode(!loraTestMode); }} size="sm" variant="outline" className="h-6 text-[10px] px-2" disabled={!loraPowered}>
                  {loraTestMode ? "退测试" : "测试"}
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-1.5">
            {/* Protocol + Mode row */}
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">协议:</label>
                <select value={loraProt} onChange={(e) => setLoraProt(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-16" disabled={!loraPowered}>
                  {protOptions.map((o, i) => <option key={o} value={i}>{o}</option>)}
                </select>
              </div>
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">模式:</label>
                <select value={loraMode} onChange={(e) => setLoraMode(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14" disabled={!loraPowered}>
                  {modeOptions.map((o, i) => <option key={o} value={i}>{o}</option>)}
                </select>
              </div>
              <div className="flex-1" />
              <Button onClick={() => CANCommandService.SendLoraCommand(2, "")} size="sm" variant="outline" className="h-6 text-[10px] px-1" disabled={!loraPowered}>查模式</Button>
              <Button onClick={() => CANCommandService.SendLoraCommand(1, ((loraProt << 4) | (loraMode & 0x0F)).toString(16).padStart(2, "0"))} size="sm" className="h-6 text-[10px] px-1" disabled={!loraPowered}>设模式</Button>
            </div>

            {/* CH1 + CH2 */}
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">CH1:</label>
                <select value={loraCh1Spd} onChange={(e) => setLoraCh1Spd(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-12" disabled={!loraPowered}>
                  {spdOptions.map((s) => <option key={s} value={s}>{s}</option>)}
                </select>
                <select value={loraCh1Freq} onChange={(e) => setLoraCh1Freq(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14" disabled={!loraPowered}>
                  {freqOptions.map((f) => <option key={f} value={f}>{f}</option>)}
                </select>
                <Button onClick={() => CANCommandService.SendLoraCommand(4, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>查</Button>
                <Button onClick={() => CANCommandService.SendLoraCommand(3, `${loraCh1Spd.toString(16).padStart(2, "0")}${((loraCh1Freq >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraCh1Freq & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>设</Button>
              </div>
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">CH2:</label>
                <select value={loraCh2Spd} onChange={(e) => setLoraCh2Spd(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-12" disabled={!loraPowered}>
                  {spdOptions.map((s) => <option key={s} value={s}>{s}</option>)}
                </select>
                <select value={loraCh2Freq} onChange={(e) => setLoraCh2Freq(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14" disabled={!loraPowered}>
                  {freqOptions.map((f) => <option key={f} value={f}>{f}</option>)}
                </select>
                <Button onClick={() => CANCommandService.SendLoraCommand(6, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>查</Button>
                <Button onClick={() => CANCommandService.SendLoraCommand(5, `${loraCh2Spd.toString(16).padStart(2, "0")}${((loraCh2Freq >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraCh2Freq & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>设</Button>
              </div>
            </div>

            {/* PNUM + GWID + NID */}
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">PNUM:</label>
                <select value={loraPnum} onChange={(e) => setLoraPnum(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-10" disabled={!loraPowered}>
                  {[0, 1, 2].map((p) => <option key={p} value={p}>{p}</option>)}
                </select>
                <Button onClick={() => CANCommandService.SendLoraCommand(0x0B, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>查</Button>
                <Button onClick={() => CANCommandService.SendLoraCommand(0x0C, loraPnum.toString(16).padStart(2, "0"))} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>设</Button>
              </div>
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">GWID:</label>
                <Input value={loraGwid.toString(16).toUpperCase().padStart(8, "0")} onChange={(e) => setLoraGwid(parseInt(e.target.value, 16) || 0)} className="w-20 h-7 text-[10px] font-mono" disabled={!loraPowered} />
                <Button onClick={() => CANCommandService.SendLoraCommand(9, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>查</Button>
                <Button onClick={() => CANCommandService.SendLoraCommand(0x0A, `${((loraGwid >> 24) & 0xFF).toString(16).padStart(2, "0")}${((loraGwid >> 16) & 0xFF).toString(16).padStart(2, "0")}${((loraGwid >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraGwid & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>设</Button>
              </div>
              <div className="flex items-center gap-1">
                <label className="text-[10px] text-muted-foreground">NID:</label>
                <Input value={loraNid.toString(16).toUpperCase().padStart(8, "0")} onChange={(e) => setLoraNid(parseInt(e.target.value, 16) || 0)} className="w-20 h-7 text-[10px] font-mono" disabled={!loraPowered} />
                <Button onClick={() => CANCommandService.SendLoraCommand(7, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>查</Button>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Row 2: Bus Monitor */}
      <Card>
        <CardHeader className="pb-1 pt-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm">总线监视器</CardTitle>
            <div className="flex items-center gap-2">
              <label className="flex items-center gap-1 text-[10px] text-muted-foreground cursor-pointer">
                <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} /> 自动滚动
              </label>
              <Button variant="ghost" size="icon" onClick={() => setFrames([])} title="清空">
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div ref={monitorRef} className="h-[380px] overflow-y-auto bg-terminal-bg rounded-md font-mono text-[11px]">
            <table className="w-full">
              <thead className="sticky top-0 bg-terminal-header">
                <tr className="text-muted-foreground text-left">
                  <th className="px-2 py-0.5 w-16">时间</th>
                  <th className="px-2 py-0.5 w-6"></th>
                  <th className="px-2 py-0.5 w-16">ID</th>
                  <th className="px-2 py-0.5 w-20">标注</th>
                  <th className="px-2 py-0.5">数据</th>
                  <th className="px-2 py-0.5 w-8">DLC</th>
                </tr>
              </thead>
              <tbody>
                {frames.map((f, i) => (
                  <tr key={i} className={f.isTX ? "text-terminal-tx" : "text-terminal-rx"}>
                    <td className="px-2 py-0.5">{f.timestamp}</td>
                    <td className="px-2 py-0.5">{f.isTX ? "TX" : "RX"}</td>
                    <td className="px-2 py-0.5">0x{f.id.toString(16).toUpperCase().padStart(3, "0")}</td>
                    <td className="px-2 py-0.5 text-terminal-label">{f.label}</td>
                    <td className="px-2 py-0.5">{f.data.map((b) => b.toString(16).toUpperCase().padStart(2, "0")).join(" ")}</td>
                    <td className="px-2 py-0.5">{f.dlc}</td>
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
