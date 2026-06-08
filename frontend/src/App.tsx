import { useState, useEffect, useCallback } from "react";
import { Sidebar, APP_VERSION } from "@/components/layout/sidebar";
import { I18nProvider } from "@/lib/i18n";
import { useWailsEvent } from "@/hooks/useEvents";
import { LoRaDataPage } from "@/pages/LoRaDataPage";
import { LoRaConfigPage } from "@/pages/LoRaConfigPage";
import { FirmwareUpgradePage } from "@/pages/FirmwareUpgradePage";
import { CanCommandPage } from "@/pages/CanCommandPage";
import { SimulatorPage } from "@/pages/SimulatorPage";
import { GatewaySimPage } from "@/pages/GatewaySimPage";
import { TerminalPage } from "@/pages/TerminalPage";

const pageIds = ["lora-data", "lora-config", "firmware", "can-command", "simulator", "gateway-sim", "terminal"] as const;
type PageId = (typeof pageIds)[number];

type ThemeMode = "light" | "dark";

function getInitialTheme(): ThemeMode {
  const stored = localStorage.getItem("modhandler-theme") as ThemeMode | null;
  if (stored === "light" || stored === "dark") return stored;
  return "light"; // Default: light
}

function applyTheme(mode: ThemeMode) {
  const root = document.documentElement;
  if (mode === "dark") {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
}

/** Compare semver strings, returns true if `latest` is newer than `current`. */
function isNewerVersion(current: string, latest: string): boolean {
  const cur = current.split(".").map(Number);
  const lat = latest.split(".").map(Number);
  for (let i = 0; i < Math.max(cur.length, lat.length); i++) {
    const c = cur[i] || 0;
    const l = lat[i] || 0;
    if (l > c) return true;
    if (l < c) return false;
  }
  return false;
}

function App() {
  const [activePage, setActivePage] = useState<PageId>("lora-data");
  const [darkMode, setDarkMode] = useState<ThemeMode>(getInitialTheme);
  const [showUpdateOnStart, setShowUpdateOnStart] = useState(false);
  const [canConnected, setCanConnected] = useState(false);
  const [showSimulators, setShowSimulators] = useState(false);

  // Listen for CAN connection state
  useWailsEvent<number>("can:connected", () => setCanConnected(true));
  useWailsEvent<any>("can:disconnected", () => setCanConnected(false));

  // Ctrl+Shift+P toggles simulator visibility
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.shiftKey && e.key === "P") {
        e.preventDefault();
        setShowSimulators((prev) => {
          const next = !prev;
          // If hiding and current page is a simulator, navigate away
          if (!next && (activePage === "simulator" || activePage === "gateway-sim")) {
            setActivePage("lora-data");
          }
          return next;
        });
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [activePage]);

  // Apply theme on mount and when it changes
  useEffect(() => {
    applyTheme(darkMode);
    localStorage.setItem("modhandler-theme", darkMode);
  }, [darkMode]);

  // Auto-check for updates on startup
  useEffect(() => {
    const checkOnStart = async () => {
      try {
        const resp = await fetch("https://api.github.com/repos/kabirz/modhandlergo/releases/latest", { cache: "no-store" });
        if (!resp.ok) return;
        const data = await resp.json();
        const tag: string = data.tag_name || "";
        const ver = tag.replace(/^v/, "");
        if (ver && isNewerVersion(APP_VERSION, ver)) {
          setShowUpdateOnStart(true);
        }
      } catch {
        // Silently ignore — user can check manually via version button
      }
    };
    checkOnStart();
  }, []);

  const toggleTheme = useCallback(() => {
    setDarkMode((prev) => (prev === "dark" ? "light" : "dark"));
  }, []);

  return (
    <I18nProvider>
    <div className="flex h-screen overflow-hidden bg-background transition-colors duration-300">
      <Sidebar
        activePage={activePage}
        onNavigate={(id) => setActivePage(id as PageId)}
        darkMode={darkMode === "dark"}
        onToggleTheme={toggleTheme}
        defaultShowUpdate={showUpdateOnStart}
        canConnected={canConnected}
        showSimulators={showSimulators}
      />
      <main className="flex-1 overflow-hidden p-6">
        <div className="h-full overflow-y-auto">
          {/* Keep all pages mounted to preserve connection state and event listeners */}
          <div style={{ display: activePage === "lora-data" ? "contents" : "none" }}><LoRaDataPage /></div>
          <div style={{ display: activePage === "lora-config" ? "contents" : "none" }}><LoRaConfigPage /></div>
          <div style={{ display: activePage === "firmware" ? "contents" : "none" }}><FirmwareUpgradePage /></div>
          <div style={{ display: activePage === "can-command" ? "contents" : "none" }}><CanCommandPage /></div>
          <div style={{ display: activePage === "simulator" ? "contents" : "none" }}><SimulatorPage canConnected={canConnected} /></div>
          <div style={{ display: activePage === "gateway-sim" ? "contents" : "none" }}><GatewaySimPage /></div>
          <div style={{ display: activePage === "terminal" ? "contents" : "none" }}><TerminalPage /></div>
        </div>
      </main>
    </div>
    </I18nProvider>
  );
}

export default App;
