import React, { useState, useCallback, useEffect } from "react";
import { cn } from "@/lib/utils";
import { useI18n } from "@/lib/i18n";
import {
  Radio, Settings, Upload, Terminal, Sun, Moon, Languages, Info, X, RefreshCw, Download, Check, Cpu, Router, Monitor, type LucideIcon,
} from "lucide-react";

interface NavItem { id: string; labelKey: string; icon: LucideIcon; }

const navItems: NavItem[] = [
  { id: "lora-data", labelKey: "nav.loraData", icon: Radio },
  { id: "lora-config", labelKey: "nav.loraConfig", icon: Settings },
  { id: "firmware", labelKey: "nav.firmware", icon: Upload },
  { id: "can-command", labelKey: "nav.canCommand", icon: Terminal },
  { id: "simulator", labelKey: "nav.simulator", icon: Cpu },
  { id: "gateway-sim", labelKey: "nav.gatewaySim", icon: Router },
  { id: "terminal", labelKey: "nav.terminal", icon: Monitor },
];

export const APP_VERSION = "0.2.3";

interface SidebarProps {
  activePage: string;
  onNavigate: (pageId: string) => void;
  darkMode: boolean;
  onToggleTheme: () => void;
  defaultShowUpdate?: boolean;
  canConnected?: boolean;
  showSimulators?: boolean;
}

function AboutDialog({ onClose }: { onClose: () => void }) {
  const { t } = useI18n();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="bg-card border border-border rounded-xl shadow-2xl w-[460px] p-6 space-y-4"
        onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-foreground">{t("about.title")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>
        <div className="space-y-3 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.appName")}</span>
            <span className="font-medium">激光测距工具</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.version")}</span>
            <span className="font-mono">v{APP_VERSION}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.techStack")}</span>
            <span>Go + Wails v3 + React</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.frontend")}</span>
            <span>TypeScript + Tailwind CSS v4</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.canAdapter")}</span>
            <span>PCAN (Windows) / SocketCAN (Linux)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.serial")}</span>
            <span>go.bug.st/serial (跨平台)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.terminal")}</span>
            <span>xterm.js + Telnet (IAC/NAWS/TTYPE)</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">{t("about.protocol")}</span>
            <span>USR1566 / LoRa / CAN 2.0 / Telnet</span>
          </div>
          <div className="border-t border-border pt-3 mt-3 space-y-2">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("about.shortcuts")}</span>
              <span>Ctrl+Q 退出 · Ctrl+Shift+P 终端</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("about.license")}</span>
              <span>Apache-2.0</span>
            </div>
          </div>
        </div>
        <div className="flex justify-end pt-2">
          <button onClick={onClose}
            className="px-4 py-1.5 rounded-md bg-primary text-primary-foreground text-sm hover:bg-primary/90 cursor-pointer">
            {t("about.ok")}
          </button>
        </div>
      </div>
    </div>
  );
}

function UpdateCheckDialog({ onClose, autoCheck = false }: { onClose: () => void; autoCheck?: boolean }) {
  const { t } = useI18n();
  const [status, setStatus] = useState<"idle" | "checking" | "found" | "latest" | "error">("idle");
  const [latestVersion, setLatestVersion] = useState("");
  const [downloadUrl, setDownloadUrl] = useState("");
  const [errorMsg, setErrorMsg] = useState("");

  const checkUpdate = useCallback(async () => {
    setStatus("checking");
    try {
      const resp = await fetch("https://api.github.com/repos/kabirz/modhandlergo/releases/latest", { cache: "no-store" });
      if (!resp.ok) {
        setStatus("latest");
        return;
      }
      const data = await resp.json();
      const tag: string = data.tag_name || "";
      const ver = tag.replace(/^v/, "");
      setLatestVersion(ver || tag);

      const asset = (data.assets || []).find((a: any) =>
        a.name && a.name.endsWith("-installer.exe") && a.name.includes("amd64")
      );
      setDownloadUrl(asset ? asset.browser_download_url : (data.html_url || ""));

      // Compare versions
      const curParts = APP_VERSION.split(".").map(Number);
      const newParts = ver.split(".").map(Number);
      let isNewer = false;
      for (let i = 0; i < Math.max(curParts.length, newParts.length); i++) {
        const c = curParts[i] || 0;
        const n = newParts[i] || 0;
        if (n > c) { isNewer = true; break; }
        if (n < c) break;
      }
      setStatus(isNewer ? "found" : "latest");
    } catch (err: any) {
      setErrorMsg(err.message || String(err));
      setStatus("error");
    }
  }, []);

  // Auto-check on mount when opened from startup
  useEffect(() => {
    if (autoCheck) checkUpdate();
  }, [autoCheck, checkUpdate]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="bg-card border border-border rounded-xl shadow-2xl w-[400px] p-6 space-y-4"
        onClick={(e) => e.stopPropagation()}>
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold text-foreground">{t("update.check")}</h2>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground cursor-pointer">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-3 text-sm">
          {/* Current version */}
          <div className="flex justify-between items-center">
            <span className="text-muted-foreground">{t("update.current")}</span>
            <span className="font-mono">v{APP_VERSION}</span>
          </div>

          {/* Status */}
          {status === "idle" && (
            <div className="text-center py-4">
              <button onClick={checkUpdate}
                className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm hover:bg-primary/90 cursor-pointer">
                <RefreshCw className="h-4 w-4" /> {t("update.check")}
              </button>
            </div>
          )}

          {status === "checking" && (
            <div className="flex items-center justify-center gap-2 py-4 text-muted-foreground">
              <RefreshCw className="h-4 w-4 animate-spin" /> {t("update.checking")}
            </div>
          )}

          {status === "latest" && (
            <div className="flex items-center justify-center gap-2 py-4 text-success">
              <Check className="h-5 w-5" /> {t("update.latest")}
            </div>
          )}

          {status === "found" && (
            <>
              <div className="flex justify-between items-center">
                <span className="text-muted-foreground">{t("update.latestVer")}</span>
                <span className="font-mono font-semibold text-success">v{latestVersion}</span>
              </div>
              <div className="text-center py-2">
                <span className="inline-block px-3 py-1 rounded-md bg-success/10 text-success text-sm font-medium">{t("update.available")}</span>
              </div>
              <div className="flex justify-center">
                <a href={downloadUrl} target="_blank" rel="noopener noreferrer"
                  className="inline-flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground text-sm hover:bg-primary/90 no-underline">
                  <Download className="h-4 w-4" /> {t("update.download")}
                </a>
              </div>
            </>
          )}

          {status === "error" && (
            <div className="text-center py-4 text-destructive text-sm">
              <p>{t("update.failed")}</p>
              <p className="text-xs text-muted-foreground mt-1 font-mono">{errorMsg}</p>
              <button onClick={checkUpdate}
                className="mt-2 px-3 py-1 rounded-md bg-muted text-sm hover:bg-muted/80 cursor-pointer">
                {t("update.check")}
              </button>
            </div>
          )}
        </div>

        <div className="flex justify-end pt-2">
          <button onClick={onClose}
            className="px-4 py-1.5 rounded-md bg-primary text-primary-foreground text-sm hover:bg-primary/90 cursor-pointer">
            {t("about.ok")}
          </button>
        </div>
      </div>
    </div>
  );
}

export function Sidebar({ activePage, onNavigate, darkMode, onToggleTheme, defaultShowUpdate = false, canConnected = false, showSimulators = false }: SidebarProps) {
  const { lang, setLang, t } = useI18n();
  const [showAbout, setShowAbout] = useState(false);
  const [showUpdate, setShowUpdate] = useState(defaultShowUpdate);

  const canDisabledTooltip = canConnected ? "" : t("nav.canDisabled");

  return (
    <>
      <aside className="w-[200px] min-w-[200px] h-screen bg-sidebar border-r border-sidebar-border flex flex-col transition-colors duration-300">
        {/* Navigation */}
        <nav className="flex-1 py-2 px-2 space-y-1">
          {navItems.filter((item) => {
            if (item.id === "simulator" || item.id === "gateway-sim" || item.id === "terminal") return showSimulators;
            return true;
          }).map((item) => {
            const Icon = item.icon;
            const isActive = activePage === item.id;
            const isDisabled = (item.id === "can-command" || item.id === "simulator") && !canConnected;
            return (
              <button
                key={item.id}
                onClick={() => { if (!isDisabled) onNavigate(item.id); }}
                title={isDisabled ? canDisabledTooltip : ""}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors",
                  isDisabled
                    ? "opacity-40 cursor-not-allowed"
                    : "cursor-pointer",
                  !isDisabled && (
                    isActive
                      ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                      : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
                  )
                )}
              >
                <Icon className="h-4 w-4" />
                <span>{t(item.labelKey)}</span>
              </button>
            );
          })}
        </nav>

        {/* Footer */}
        <div className="px-2 py-3 border-t border-sidebar-border space-y-2">
          {/* Theme */}
          <div className="flex items-center gap-1 px-2">
            <span className="text-[11px] text-muted-foreground shrink-0 min-w-12 text-right whitespace-nowrap">{t("sidebar.theme")}：</span>
            <button
              onClick={() => { if (darkMode) onToggleTheme(); }}
              className={cn(
                "p-2 rounded cursor-pointer transition-colors",
                !darkMode ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              <Sun className="h-4 w-4" />
            </button>
            <button
              onClick={() => { if (!darkMode) onToggleTheme(); }}
              className={cn(
                "p-2 rounded cursor-pointer transition-colors",
                darkMode ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              <Moon className="h-4 w-4" />
            </button>
          </div>

          {/* Language */}
          <div className="flex items-center gap-1 px-2">
            <span className="text-[11px] text-muted-foreground shrink-0 min-w-12 text-right whitespace-nowrap">{t("sidebar.lang")}：</span>
            <button
              onClick={() => setLang("zh")}
              className={cn(
                "px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
                lang === "zh" ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              中文
            </button>
            <button
              onClick={() => setLang("en")}
              className={cn(
                "px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
                lang === "en" ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              English
            </button>
          </div>

          {/* Version + About */}
          <div className="flex items-center justify-between px-2">
            <button
              onClick={() => setShowUpdate(true)}
              className="text-[11px] text-muted-foreground hover:text-foreground cursor-pointer transition-colors"
            >
              {t("sidebar.version")}：v{APP_VERSION}
            </button>
            <button
              onClick={() => setShowAbout(true)}
              className="flex items-center gap-1 px-1.5 py-0.5 rounded text-[11px] text-muted-foreground hover:text-foreground hover:bg-muted cursor-pointer transition-colors"
            >
              <Info className="h-3 w-3" />{t("about.title")}
            </button>
          </div>
        </div>
      </aside>
      {showAbout && <AboutDialog onClose={() => setShowAbout(false)} />}
      {showUpdate && <UpdateCheckDialog onClose={() => setShowUpdate(false)} autoCheck={defaultShowUpdate} />}
    </>
  );
}
