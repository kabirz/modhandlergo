import React, { useState, useCallback, useRef, useEffect } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { useWailsEvent } from "@/hooks/useEvents";
import { Terminal } from "lucide-react";

interface FrameEntry {
  id: number;
  data: number[];
  dlc: number;
  isTX: boolean;
  label: string;
  timestamp: string;
}

const FRAME_LABELS: Record<number, string> = {
  0x101: "控制命令",
  0x102: "响应帧",
  0x103: "固件数据",
  0x105: "LoRa配参",
  0x106: "LoRa配参响应",
  0x1E3: "手柄状态",
  0x263: "激光测距",
  0x363: "X/Y坐标",
  0x463: "Z坐标",
  0x763: "心跳",
};

const quickCommands = [
  { name: "启动升级", id: 0x101, data: [0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00] },
  { name: "确认", id: 0x101, data: [0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00] },
  { name: "查版本", id: 0x101, data: [0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00] },
  { name: "重启", id: 0x101, data: [0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00] },
];

export function CanCommandPage() {
  const [canId, setCanId] = useState("101");
  const [dataHex, setDataHex] = useState("00 00 00 00 00 00 00 00");
  const [isExtended, setIsExtended] = useState(false);
  const [isRemote, setIsRemote] = useState(false);
  const [monitoring, setMonitoring] = useState(false);
  const [frames, setFrames] = useState<FrameEntry[]>([]);
  const monitorRef = useRef<HTMLDivElement>(null);

  const addFrame = useCallback((frame: FrameEntry) => {
    setFrames((prev) => [...prev.slice(-200), frame]);
  }, []);

  useEffect(() => {
    if (monitorRef.current) {
      monitorRef.current.scrollTop = monitorRef.current.scrollHeight;
    }
  }, [frames]);

  useWailsEvent<any>("can:frame", (ev) => {
    const label = FRAME_LABELS[ev.id] || "";
    addFrame({
      id: ev.id,
      data: ev.data || [],
      dlc: ev.dlc || 0,
      isTX: ev.isTx || false,
      label,
      timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }),
    });
  });

  const handleSend = () => {
    const id = parseInt(canId, 16);
    if (isNaN(id)) return;
    const bytes = dataHex.trim().split(/\s+/).map((b) => parseInt(b, 16));
    addFrame({
      id,
      data: bytes,
      dlc: bytes.length,
      isTX: true,
      label: FRAME_LABELS[id] || "",
      timestamp: new Date().toLocaleTimeString("zh-CN", { hour12: false }),
    });
  };

  return (
    <div className="space-y-4">
      {/* Frame Send */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Terminal className="h-4 w-4" /> 发送 CAN 帧
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-3 flex-wrap">
            <div>
              <label className="text-xs text-muted-foreground">帧 ID (hex)</label>
              <Input
                value={canId}
                onChange={(e) => setCanId(e.target.value)}
                placeholder="101"
                className="w-24 mt-1 font-mono"
              />
            </div>
            <div className="flex-1 min-w-[200px]">
              <label className="text-xs text-muted-foreground">数据 (hex, 空格分隔)</label>
              <Input
                value={dataHex}
                onChange={(e) => setDataHex(e.target.value)}
                placeholder="00 00 00 00 00 00 00 00"
                className="mt-1 font-mono"
              />
            </div>
            <label className="flex items-center gap-1.5 text-sm mt-5">
              <input type="checkbox" checked={isExtended} onChange={(e) => setIsExtended(e.target.checked)} />
              扩展帧
            </label>
            <label className="flex items-center gap-1.5 text-sm mt-5">
              <input type="checkbox" checked={isRemote} onChange={(e) => setIsRemote(e.target.checked)} />
              远程帧
            </label>
            <Button onClick={handleSend} className="mt-5">发送</Button>
          </div>
        </CardContent>
      </Card>

      {/* Quick Commands */}
      <Card>
        <CardHeader>
          <CardTitle>快捷命令</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 flex-wrap">
            {quickCommands.map((cmd, i) => (
              <Button key={i} variant="outline" size="sm" onClick={() => {
                setCanId(cmd.id.toString(16));
                setDataHex(cmd.data.map((b) => b.toString(16).padStart(2, "0")).join(" "));
              }}>
                {cmd.name}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Bus Monitor */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>总线监视器</CardTitle>
            <Button
              variant={monitoring ? "destructive" : "outline"}
              size="sm"
              onClick={() => setMonitoring(!monitoring)}
            >
              {monitoring ? "停止监视" : "开始监视"}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div
            ref={monitorRef}
            className="h-64 overflow-y-auto bg-black/30 rounded-md font-mono text-xs"
          >
            <table className="w-full">
              <thead className="sticky top-0 bg-zinc-900">
                <tr className="text-muted-foreground text-left">
                  <th className="px-2 py-1 w-20">时间</th>
                  <th className="px-2 py-1 w-8"></th>
                  <th className="px-2 py-1 w-20">ID</th>
                  <th className="px-2 py-1 w-24">标注</th>
                  <th className="px-2 py-1">数据</th>
                  <th className="px-2 py-1 w-10">DLC</th>
                </tr>
              </thead>
              <tbody>
                {frames.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-2 py-4 text-center text-muted-foreground">
                      等待 CAN 帧...
                    </td>
                  </tr>
                ) : (
                  frames.map((f, i) => (
                    <tr key={i} className={f.isTX ? "text-blue-400" : "text-green-400"}>
                      <td className="px-2 py-0.5">{f.timestamp}</td>
                      <td className="px-2 py-0.5">{f.isTX ? "TX" : "RX"}</td>
                      <td className="px-2 py-0.5">0x{f.id.toString(16).toUpperCase().padStart(3, "0")}</td>
                      <td className="px-2 py-0.5 text-yellow-400">{f.label}</td>
                      <td className="px-2 py-0.5">
                        {f.data.map((b) => b.toString(16).toUpperCase().padStart(2, "0")).join(" ")}
                      </td>
                      <td className="px-2 py-0.5">{f.dlc}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      {/* LoRa Remote Config */}
      <Card>
        <CardHeader>
          <CardTitle>LoRa 远程配参 (0x105/0x106)</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-4 gap-3">
            <div>
              <label className="text-xs text-muted-foreground">协议</label>
              <Select className="mt-1">
                <option>SET_MODE</option>
                <option>QUERY_MODE</option>
                <option>SET_CH</option>
                <option>SET_NID</option>
                <option>SET_GWID</option>
                <option>SET_TEST</option>
                <option>SET_POWER</option>
              </Select>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">参数值</label>
              <Input placeholder="0" className="mt-1 font-mono" />
            </div>
            <div className="flex items-end">
              <Button size="sm">发送配置</Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
