import { useState, useEffect, useRef, useCallback } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useWailsEvent } from "@/hooks/useEvents";
import { useI18n } from "@/lib/i18n";
import { TerminalService } from "../../bindings/github.com/kabirz/modhandlergo/service";
import { Cable, Plug, Trash2 } from "lucide-react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";

const BAUD_RATES = [9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600];

const xtermTheme = {
  background: "#282A36",
  foreground: "#F8F8F2",
  cursor: "#F8F8F2",
  selectionBackground: "#44475A",
  black: "#21222C",
  red: "#FF5555",
  green: "#50FA7B",
  yellow: "#F1FA8C",
  blue: "#8BE9FD",
  magenta: "#FF79C6",
  cyan: "#8BE9FD",
  white: "#F8F8F2",
  brightBlack: "#6272A4",
  brightRed: "#FF6E6E",
  brightGreen: "#69FF94",
  brightYellow: "#FFFFA5",
  brightBlue: "#D6ACFF",
  brightMagenta: "#FF92DF",
  brightCyan: "#A4FFFF",
  brightWhite: "#FFFFFF",
};

export function TerminalPage() {
  const { t } = useI18n();

  const [transport, setTransport] = useState<"uart" | "tcp" | "telnet">("uart");
  const [host, setHost] = useState("127.0.0.1");
  const [tcpPort, setTcpPort] = useState("23");
  const [uartPort, setUartPort] = useState("");
  const [baudRate, setBaudRate] = useState("115200");
  const [availablePorts, setAvailablePorts] = useState<string[]>([]);
  const [connected, setConnected] = useState(false);

  const termRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const connectedRef = useRef(false);
  const reconnectRef = useRef<() => void>(() => {});

  useEffect(() => {
    connectedRef.current = connected;
  }, [connected]);

  // Refresh serial ports when switching to UART
  useEffect(() => {
    if (transport === "uart" && !connected) {
      TerminalService.EnumPorts().then((ports) => {
        const names = (ports || []).map((p: any) => p.name || "");
        setAvailablePorts(names);
        if (names.length > 0 && !uartPort) setUartPort(names[0]);
      }).catch(() => {});
    }
  }, [transport, connected]);

  // Initialize xterm
  useEffect(() => {
    if (!termRef.current || xtermRef.current) return;

    const fitAddon = new FitAddon();
    const term = new Terminal({
      theme: xtermTheme,
      cursorBlink: true,
      cursorStyle: "bar",
      fontSize: 12,
      fontFamily: "'JetBrains Mono', 'Cascadia Code', 'Fira Code', 'Consolas', monospace",
      allowProposedApi: true,
      scrollback: 10000,
      convertEol: true,
    });

    fitAddonRef.current = fitAddon;
    xtermRef.current = term;

    term.loadAddon(fitAddon);
    term.open(termRef.current);
    // Delay fit to let layout settle first
    requestAnimationFrame(() => fitAddon.fit());

    // Intercept printable chars in DOM capturing phase to suppress local echo.
    // By preventing the event from reaching xterm's internal textarea,
    // the character is never rendered locally and onData does not fire.
    const onRawKey = (e: KeyboardEvent) => {
      // Disconnected: 'r' to reconnect
      if (!connectedRef.current) {
        if (e.key === "r" && !e.ctrlKey && !e.altKey && !e.metaKey) {
          e.preventDefault();
          e.stopPropagation();
          reconnectRef.current();
        }
        return;
      }
      // Ctrl+L: clear terminal screen
      if (e.ctrlKey && e.key === "l") {
        e.preventDefault();
        e.stopPropagation();
        term.clear();
        return;
      }
      if (
        e.key.length === 1 &&
        !e.ctrlKey &&
        !e.altKey &&
        !e.metaKey
      ) {
        e.preventDefault();
        e.stopPropagation();
        TerminalService.Send(e.key).catch(() => {});
      }
    };
    termRef.current.addEventListener("keydown", onRawKey, true);

    // Backspace: xterm sends \x7f (DEL) by default,
    // but most embedded devices expect \b (0x08).
    term.attachCustomKeyEventHandler((event) => {
      if (event.type === "keydown" && event.key === "Backspace") {
        if (connectedRef.current) {
          TerminalService.Send("\b").catch(() => {});
        }
        return false;
      }
      return true;
    });

    // User input → backend (only non-printable keys reach here:
    // Enter, arrows, Ctrl+C, Tab, etc.)
    term.onData((data) => {
      if (connectedRef.current) {
        TerminalService.Send(data).catch(() => {});
      }
    });

    // Notify backend of terminal size changes (Telnet NAWS)
    term.onResize(({ cols, rows }) => {
      TerminalService.Resize(cols, rows).catch(() => {});
    });

    // Handle resize
    const onResize = () => fitAddon.fit();
    window.addEventListener("resize", onResize);
    const ro = new ResizeObserver(onResize);
    ro.observe(termRef.current);

    return () => {
      window.removeEventListener("resize", onResize);
      termRef.current?.removeEventListener("keydown", onRawKey, true);
      ro.disconnect();
      term.dispose();
      xtermRef.current = null;
    };
  }, []);

  // Auto-fit after connection changes (layout may shift)
  useEffect(() => {
    setTimeout(() => fitAddonRef.current?.fit(), 100);
  }, [connected]);

  const appendOutput = useCallback((text: string) => {
    xtermRef.current?.write(text);
  }, []);

  const clearTerminal = () => {
    xtermRef.current?.clear();
  };

  useEffect(() => {
    TerminalService.IsConnected().then(setConnected).catch(() => {});
    TerminalService.EnumPorts().then((ports) => {
      const names = (ports || []).map((p: any) => p.name || "");
      setAvailablePorts(names);
      if (names.length > 0 && !uartPort) setUartPort(names[0]);
    }).catch(() => {});
  }, []);

  useWailsEvent<boolean>("terminal:status", (running) => {
    setConnected(running);
    if (!running) {
      appendOutput("\r\n\x1b[31m── Disconnected ──\x1b[0m\r\n");
      appendOutput("\x1b[33mPress 'r' to reconnect\x1b[0m\r\n");
    }
  });

  useWailsEvent<string>("terminal:data", (data) => {
    appendOutput(data);
  });

  useWailsEvent<string>("terminal:error", (msg) => {
    appendOutput(`\r\n\x1b[31m[ERR] ${msg}\x1b[0m\r\n`);
  });

  const handleConnect = async () => {
    try {
      const cols = xtermRef.current?.cols ?? 80;
      const rows = xtermRef.current?.rows ?? 24;
      if (transport === "telnet") {
        await TerminalService.ConnectTelnet(host, parseInt(tcpPort) || 23, cols, rows);
      } else if (transport === "tcp") {
        await TerminalService.ConnectTCP(host, parseInt(tcpPort) || 23);
      } else {
        await TerminalService.ConnectUART(uartPort, parseInt(baudRate) || 115200);
      }
    } catch (err: any) {
      appendOutput(`\r\n\x1b[31m[ERR] ${err}\x1b[0m\r\n`);
    }
  };

  // Keep reconnect ref in sync with current connection params
  useEffect(() => {
    reconnectRef.current = handleConnect;
  });

  const handleDisconnect = async () => {
    try {
      await TerminalService.Disconnect();
    } catch (err: any) {
      appendOutput(`\r\n\x1b[31m[ERR] ${err}\x1b[0m\r\n`);
    }
  };

  return (
    <div className="space-y-3 h-full flex flex-col">
      {/* Connection Panel */}
      <Card>
        <CardContent className="pt-3">
          <div className="flex items-center gap-2 flex-wrap">
            <div className="flex items-center gap-1.5">
              <label className="text-xs text-muted-foreground whitespace-nowrap">{t("terminal.transport")}</label>
              <select value={transport} onChange={(e) => setTransport(e.target.value as "uart" | "tcp" | "telnet")} disabled={connected}
                className="h-7 text-xs rounded-md border border-input bg-background px-2 font-mono disabled:opacity-50">
                <option value="uart">UART</option>
                <option value="tcp">TCP</option>
                <option value="telnet">{t("terminal.telnet")}</option>
              </select>
            </div>

            {(transport === "tcp" || transport === "telnet") && (
              <>
                <div className="flex items-center gap-1.5">
                  <label className="text-xs text-muted-foreground whitespace-nowrap">{t("terminal.host")}</label>
                  <Input value={host} onChange={(e) => setHost(e.target.value)} className="h-7 text-xs w-28 font-mono" disabled={connected} />
                </div>
                <div className="flex items-center gap-1.5">
                  <label className="text-xs text-muted-foreground whitespace-nowrap">{t("terminal.port")}</label>
                  <Input value={tcpPort} onChange={(e) => setTcpPort(e.target.value.replace(/\D/g, ""))} className="h-7 text-xs w-16 font-mono" disabled={connected} />
                </div>
              </>
            )}

            {transport === "uart" && (
              <>
                <div className="flex items-center gap-1.5">
                  <label className="text-xs text-muted-foreground whitespace-nowrap">{t("terminal.port")}</label>
                  <select value={uartPort} onChange={(e) => setUartPort(e.target.value)} disabled={connected}
                    className="h-7 text-xs rounded-md border border-input bg-background px-2 font-mono disabled:opacity-50">
                    {availablePorts.length === 0 && <option value="">{t("terminal.noPorts")}</option>}
                    {availablePorts.map((p) => <option key={p} value={p}>{p}</option>)}
                  </select>
                </div>
                <div className="flex items-center gap-1.5">
                  <label className="text-xs text-muted-foreground whitespace-nowrap">{t("terminal.baud")}</label>
                  <select value={baudRate} onChange={(e) => setBaudRate(e.target.value)} disabled={connected}
                    className="h-7 text-xs rounded-md border border-input bg-background px-2 font-mono disabled:opacity-50">
                    {BAUD_RATES.map((b) => <option key={b} value={b}>{b}</option>)}
                  </select>
                </div>
              </>
            )}

            <div className="flex items-center gap-1">
              {connected ? (
                <Button onClick={handleDisconnect} variant="destructive" size="sm" className="h-7 text-xs px-4">
                  <Plug className="h-3 w-3 mr-1" /> {t("terminal.disconnect")}
                </Button>
              ) : (
                <Button onClick={handleConnect} size="sm" className="h-7 text-xs px-4">
                  <Cable className="h-3 w-3 mr-1" /> {t("terminal.connect")}
                </Button>
              )}
            </div>

            <div className="flex-1" />

            <div className="flex items-center gap-1">
              <span className={`h-2 w-2 rounded-full ${connected ? "bg-green-500" : "bg-muted-foreground"}`} />
              <span className="text-[10px] text-muted-foreground">
                {connected ? t("terminal.connected") : t("terminal.disconnected")}
              </span>
            </div>
            <Button variant="ghost" size="icon" onClick={clearTerminal} title={t("terminal.clear")} className="h-7 w-7">
              <Trash2 className="h-3.5 w-3.5" />
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Terminal */}
      <Card className="flex-1 flex flex-col min-h-0">
        <CardContent className="flex-1 min-h-0 p-0">
          <div
            ref={termRef}
            className="h-full rounded-md overflow-hidden"
            style={{ background: "#282A36" }}
          />
        </CardContent>
      </Card>
    </div>
  );
}
