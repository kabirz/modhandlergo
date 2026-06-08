import React, { useState, useCallback, useRef, useEffect, memo } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { Terminal, Trash2 } from "lucide-react";
import { useI18n } from "@/lib/i18n";
import { msTimestamp } from "@/lib/utils";
import { CANCommandService } from "../../bindings/github.com/kabirz/modhandlergo/service";

interface FrameEntry {
  id: number;
  canId: number;
  data: number[];
  dlc: number;
  isTX: boolean;
  label: string;
  timestamp: string;
}

let frameIdCounter = 0;

function useFrameLabels(): Record<number, string> {
  const { t } = useI18n();
  return {
    0x101: t("can.control"), 0x102: t("can.response"), 0x103: t("can.firmware"),
    0x105: t("can.loraConfig"), 0x106: t("can.loraResp"),
    0x1E3: t("can.joystick"), 0x263: t("can.laser"), 0x363: t("can.coordXY"),
    0x463: t("can.coordZ"), 0x763: t("can.heartbeat"),
  };
}

const FrameConfigCard = memo(function FrameConfigCard() {
  const { t } = useI18n();
  const [canId, setCanId] = useState("101");
  const [dataHex, setDataHex] = useState("00 00 00 00 00 00 00 00");
  const [isExtended, setIsExtended] = useState(false);
  const [isRemote, setIsRemote] = useState(false);

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
    } catch (err: any) {
      // Error is logged by the service layer
    }
  };

  return (
    <Card>
      <CardHeader className="pb-1 pt-2">
        <CardTitle className="flex items-center gap-2 text-sm">
          <Terminal className="h-4 w-4" /> {t("can.frameConfig")}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1.5">
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted-foreground w-12 shrink-0">CAN ID:</label>
          <Input value={canId} onChange={(e) => setCanId(e.target.value)} className="w-24 h-7 text-xs font-mono" />
          <span className="text-[10px] text-muted-foreground">(Hex)</span>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted-foreground w-12 shrink-0">{t("can.format")}:</label>
          <label className="flex items-center gap-1 text-xs"><input type="radio" checked={!isExtended} onChange={() => setIsExtended(false)} /> {t("can.standard")}</label>
          <label className="flex items-center gap-1 text-xs"><input type="radio" checked={isExtended} onChange={() => setIsExtended(true)} /> {t("can.extended")}</label>
          <span className="w-6" />
          <label className="flex items-center gap-1 text-xs"><input type="radio" checked={!isRemote} onChange={() => setIsRemote(false)} /> {t("can.dataFrame")}</label>
          <label className="flex items-center gap-1 text-xs"><input type="radio" checked={isRemote} onChange={() => setIsRemote(true)} /> {t("can.remoteFrame")}</label>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-xs text-muted-foreground w-12 shrink-0">{t("can.data")}:</label>
          <Input value={dataHex} onChange={(e) => setDataHex(e.target.value)} className="flex-1 h-7 text-xs font-mono" />
        </div>
        <Button onClick={handleSendFrame} size="sm" className="w-full h-7 text-xs">{t("can.sendFrame")}</Button>
      </CardContent>
    </Card>
  );
});

const LORA_CMD_QUERY_MODE = 0x02;
const LORA_CMD_QUERY_CH1  = 0x04;
const LORA_CMD_QUERY_CH2  = 0x06;
const LORA_CMD_QUERY_NID  = 0x07;
const LORA_CMD_QUERY_GWID = 0x09;
const LORA_CMD_QUERY_PNUM = 0x0B;
const LORA_CMD_SET_TEST   = 0x0D;
const LORA_CMD_SET_POWER  = 0x0F;
const LORA_CONFIG_TX      = 0x106;

const LoraConfigCard = memo(function LoraConfigCard() {
  const { t } = useI18n();
  const [loraProt, setLoraProt] = useState(1);
  const [loraMode, setLoraMode] = useState(1);
  const [loraCh1Spd, setLoraCh1Spd] = useState(7);
  const [loraCh1Freq, setLoraCh1Freq] = useState(4800);
  const [loraCh2Spd, setLoraCh2Spd] = useState(7);
  const [loraCh2Freq, setLoraCh2Freq] = useState(4800);
  const [loraPnum, setLoraPnum] = useState(0);
  const [loraNid, setLoraNid] = useState(0);
  const [loraGwid, setLoraGwid] = useState(0);
  const [loraGwidInput, setLoraGwidInput] = useState("00000000");
  const [loraPowered, setLoraPowered] = useState(false);
  const [loraTestMode, setLoraTestMode] = useState(false);

  // Parse 0x106 LoRa config responses
  useWailsEvent<any>("can:frame", (ev) => {
    if (ev.id !== LORA_CONFIG_TX || !ev.data || ev.data.length < 2) return;
    const d = ev.data;
    switch (d[0]) {
      case LORA_CMD_QUERY_MODE:
        setLoraProt((d[1] >> 4) & 0x0F);
        setLoraMode(d[1] & 0x0F);
        break;
      case LORA_CMD_QUERY_CH1:
        setLoraCh1Spd(d[1]);
        setLoraCh1Freq((d[2] << 8) | d[3]);
        break;
      case LORA_CMD_QUERY_CH2:
        setLoraCh2Spd(d[1]);
        setLoraCh2Freq((d[2] << 8) | d[3]);
        break;
      case LORA_CMD_QUERY_PNUM:
        setLoraPnum(d[1]);
        break;
      case LORA_CMD_QUERY_NID:
        if (d.length >= 8) setLoraNid((d[4] << 24) | (d[5] << 16) | (d[6] << 8) | d[7]);
        break;
      case LORA_CMD_QUERY_GWID:
        if (d.length >= 8) {
          const gwid = (d[4] << 24) | (d[5] << 16) | (d[6] << 8) | d[7];
          setLoraGwid(gwid);
          setLoraGwidInput(gwid.toString(16).toUpperCase().padStart(8, "0"));
        }
        break;
      case LORA_CMD_SET_TEST:
        setLoraTestMode(d[1] !== 0);
        break;
      case LORA_CMD_SET_POWER:
        setLoraPowered(d[1] !== 0);
        break;
      default:
        break;
    }
  });

  const protOptions = ["NODE", "LG210", "LG220"];
  const modeOptions = ["FP", "TRANS", "NET"];
  const spdOptions = [4, 5, 6, 7, 8, 9, 10, 11];
  const freqOptions = [4100, 4200, 4300, 4400, 4500, 4600, 4700, 4800, 4900, 5000, 5100];

  return (
    <Card>
      <CardHeader className="pb-1 pt-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm">{t("can.loraConfig")}</CardTitle>
          <div className="flex items-center gap-2">
            <Button onClick={async () => { await CANCommandService.SendLoraCommand(0x0F, loraPowered ? "00" : "01"); setLoraPowered(!loraPowered); }} size="sm" variant={loraPowered ? "destructive" : "default"} className="h-6 text-[10px] px-2">
              {loraPowered ? t("can.powerOff") : t("can.powerOn")}
            </Button>
            <label className={`flex items-center gap-1 text-[10px] ${!loraPowered ? "text-muted-foreground" : "cursor-pointer"}`}>
              <input
                type="checkbox"
                checked={loraTestMode}
                onChange={async (e) => {
                  await CANCommandService.SendLoraCommand(0x0D, e.target.checked ? "01" : "00");
                  setLoraTestMode(e.target.checked);
                }}
                disabled={!loraPowered}
              />
              {t("can.testMode")}
            </label>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-1.5">
        {/* Protocol + Mode */}
        <div className="flex items-center gap-2">
          <label className="text-[10px] text-muted-foreground shrink-0">{t("cfg.loraProto")}:</label>
          <select value={loraProt} onChange={(e) => setLoraProt(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-20" disabled={!loraPowered}>
            {protOptions.map((o, i) => <option key={o} value={i}>{o}</option>)}
          </select>
          <label className="text-[10px] text-muted-foreground shrink-0 ml-2">{t("cfg.mode")}:</label>
          <select value={loraMode} onChange={(e) => setLoraMode(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-20" disabled={!loraPowered}>
            {modeOptions.map((o, i) => <option key={o} value={i}>{o}</option>)}
          </select>
          <Button onClick={() => CANCommandService.SendLoraCommand(2, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5 ml-2" disabled={!loraPowered}>{t("cfg.query")}</Button>
          <Button onClick={() => CANCommandService.SendLoraCommand(1, ((loraProt << 4) | (loraMode & 0x0F)).toString(16).padStart(2, "0"))} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.set")}</Button>
        </div>

        {/* CH1 + CH2 */}
        <div className="flex items-center gap-2">
          <label className="text-[10px] text-muted-foreground shrink-0">CH1:</label>
          <select value={loraCh1Spd} onChange={(e) => setLoraCh1Spd(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14" disabled={!loraPowered}>
            {spdOptions.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
          <select value={loraCh1Freq} onChange={(e) => setLoraCh1Freq(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-18" disabled={!loraPowered}>
            {freqOptions.map((f) => <option key={f} value={f}>{f}</option>)}
          </select>
          <Button onClick={() => CANCommandService.SendLoraCommand(4, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-1" disabled={!loraPowered}>{t("cfg.query")}</Button>
          <Button onClick={() => CANCommandService.SendLoraCommand(3, `${loraCh1Spd.toString(16).padStart(2, "0")}${((loraCh1Freq >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraCh1Freq & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-1" disabled={!loraPowered}>{t("cfg.set")}</Button>
        </div>
        <div className="flex items-center gap-2">
          <label className="text-[10px] text-muted-foreground shrink-0">CH2:</label>
          <select value={loraCh2Spd} onChange={(e) => setLoraCh2Spd(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-14" disabled={!loraPowered}>
            {spdOptions.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>
          <select value={loraCh2Freq} onChange={(e) => setLoraCh2Freq(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-18" disabled={!loraPowered}>
            {freqOptions.map((f) => <option key={f} value={f}>{f}</option>)}
          </select>
          <Button onClick={() => CANCommandService.SendLoraCommand(6, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-1" disabled={!loraPowered}>{t("cfg.query")}</Button>
          <Button onClick={() => CANCommandService.SendLoraCommand(5, `${loraCh2Spd.toString(16).padStart(2, "0")}${((loraCh2Freq >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraCh2Freq & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-1" disabled={!loraPowered}>{t("cfg.set")}</Button>
        </div>

        {/* PNUM + GWID + NID */}
        <div className="flex items-center gap-1">
          <label className="text-[10px] text-muted-foreground shrink-0">PNUM:</label>
          <select value={loraPnum} onChange={(e) => setLoraPnum(Number(e.target.value))} className="h-6 text-xs bg-background border border-input rounded px-1 w-12" disabled={!loraPowered}>
            {[0, 1, 2].map((p) => <option key={p} value={p}>{p}</option>)}
          </select>
          <Button onClick={() => CANCommandService.SendLoraCommand(0x0B, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.query")}</Button>
          <Button onClick={() => CANCommandService.SendLoraCommand(0x0C, loraPnum.toString(16).padStart(2, "0"))} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.set")}</Button>
          <label className="text-[10px] text-muted-foreground shrink-0">GWID:</label>
          <Input
            value={loraGwidInput}
            onChange={(e) => {
              const v = e.target.value.replace(/[^0-9a-fA-F]/g, "").slice(0, 8);
              setLoraGwidInput(v);
              if (v.length === 8) setLoraGwid(parseInt(v, 16));
            }}
            onBlur={() => setLoraGwidInput(loraGwid.toString(16).toUpperCase().padStart(8, "0"))}
            className="w-20 h-6 text-[10px] font-mono"
            disabled={!loraPowered}
            maxLength={8}
          />
          <Button onClick={() => CANCommandService.SendLoraCommand(9, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.query")}</Button>
          <Button onClick={() => CANCommandService.SendLoraCommand(0x0A, `000000${((loraGwid >> 24) & 0xFF).toString(16).padStart(2, "0")}${((loraGwid >> 16) & 0xFF).toString(16).padStart(2, "0")}${((loraGwid >> 8) & 0xFF).toString(16).padStart(2, "0")}${(loraGwid & 0xFF).toString(16).padStart(2, "0")}`)} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.set")}</Button>
          <label className="text-[10px] text-muted-foreground shrink-0">NID:</label>
          <span className="w-20 h-6 text-[10px] font-mono bg-muted border border-input rounded px-1 flex items-center">{loraNid.toString(16).toUpperCase().padStart(8, "0")}</span>
          <Button onClick={() => CANCommandService.SendLoraCommand(7, "")} size="sm" variant="ghost" className="h-5 text-[10px] px-0.5" disabled={!loraPowered}>{t("cfg.query")}</Button>
        </div>
      </CardContent>
    </Card>
  );
});

const FrameRow = memo(function FrameRow({ frame }: { frame: FrameEntry }) {
  return (
    <tr className={`whitespace-nowrap ${frame.isTX ? "text-terminal-tx" : "text-terminal-rx"}`}>
      <td className="px-2 py-0.5">{frame.timestamp}</td>
      <td className="px-2 py-0.5">{frame.isTX ? "TX" : "RX"}</td>
      <td className="px-2 py-0.5">0x{frame.canId.toString(16).toUpperCase().padStart(3, "0")}</td>
      <td className="px-2 py-0.5 text-terminal-label">{frame.label}</td>
      <td className="px-2 py-0.5">{frame.data.map((b) => b.toString(16).toUpperCase().padStart(2, "0")).join(" ")}</td>
      <td className="px-2 py-0.5">{frame.dlc}</td>
    </tr>
  );
});

// --- Main page ---

export function CanCommandPage() {
  const { t } = useI18n();
  const [frames, setFrames] = useState<FrameEntry[]>([]);
  const [autoScroll, setAutoScroll] = useState(true);
  const monitorRef = useRef<HTMLDivElement>(null);
  const frameLabels = useFrameLabels();

  const addFrame = useCallback((frame: FrameEntry) => {
    setFrames((prev) => [...prev.slice(-500), frame]);
  }, []);

  useEffect(() => {
    if (monitorRef.current && autoScroll) {
      monitorRef.current.scrollTop = monitorRef.current.scrollHeight;
    }
  }, [frames, autoScroll]);

  const monitorActiveRef = useRef(false);

  // Initialize on mount: if CAN already connected, start monitor immediately
  useEffect(() => {
    CANCommandService.StartMonitor().then(() => {
      monitorActiveRef.current = true;
    }).catch(() => {});
  }, []);

  useEffect(() => {
    return () => {
      if (monitorActiveRef.current) {
        CANCommandService.StopMonitor().catch(() => {});
      }
    };
  }, []);

  useWailsEvent<number>("can:connected", (channel) => {
    CANCommandService.SetChannel(channel).catch(() => {});
    CANCommandService.StartMonitor().catch(() => {});
    monitorActiveRef.current = true;
  });

  useWailsEvent<any>("can:disconnected", () => {
    CANCommandService.StopMonitor().catch(() => {});
    monitorActiveRef.current = false;
  });

  useWailsEvent<any>("can:frame", (ev) => {
    const label = frameLabels[ev.id] || "";
    addFrame({
      id: frameIdCounter++, canId: ev.id, data: ev.data || [], dlc: ev.dlc || 0,
      isTX: ev.isTx || false, label,
      timestamp: msTimestamp(),
    });
  });

  return (
    <div className="space-y-3">
      {/* Row 1: Frame Config + LoRa Config side by side */}
      <div className="grid grid-cols-[480px_1fr] gap-3">
        <FrameConfigCard />
        <LoraConfigCard />
      </div>

      {/* Row 2: Bus Monitor */}
      <Card>
        <CardHeader className="pb-1 pt-2">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm">{t("can.busMonitor")}</CardTitle>
            <div className="flex items-center gap-2">
              <label className="flex items-center gap-1 text-[10px] text-muted-foreground cursor-pointer">
                <input type="checkbox" checked={autoScroll} onChange={(e) => setAutoScroll(e.target.checked)} /> {t("can.autoScroll")}
              </label>
              <Button variant="ghost" size="icon" onClick={() => setFrames([])} title="Clear">
                <Trash2 className="h-3.5 w-3.5" />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div ref={monitorRef} className="overflow-y-auto bg-terminal-bg rounded-md font-mono text-[11px] terminal-selectable terminal-lg">
            <table className="w-full">
              <thead className="sticky top-0 bg-terminal-header">
                <tr className="text-muted-foreground text-left whitespace-nowrap">
                  <th className="px-2 py-0.5 w-28">{t("can.time")}</th>
                  <th className="px-2 py-0.5 w-6"></th>
                  <th className="px-2 py-0.5 w-16">ID</th>
                  <th className="px-2 py-0.5 w-20">{t("can.label")}</th>
                  <th className="px-2 py-0.5">{t("can.data")}</th>
                  <th className="px-2 py-0.5 w-8">DLC</th>
                </tr>
              </thead>
              <tbody>
                {frames.map((f) => <FrameRow key={f.id} frame={f} />)}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
