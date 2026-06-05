import { useState, useEffect, useCallback } from "react";
import { Sidebar } from "@/components/layout/sidebar";
import { LoRaDataPage } from "@/pages/LoRaDataPage";
import { LoRaConfigPage } from "@/pages/LoRaConfigPage";
import { FirmwareUpgradePage } from "@/pages/FirmwareUpgradePage";
import { CanCommandPage } from "@/pages/CanCommandPage";

const pages: Record<string, React.FC> = {
  "lora-data": LoRaDataPage,
  "lora-config": LoRaConfigPage,
  firmware: FirmwareUpgradePage,
  "can-command": CanCommandPage,
};

type ThemeMode = "light" | "dark";

function getInitialTheme(): ThemeMode {
  const stored = localStorage.getItem("modhandler-theme") as ThemeMode | null;
  if (stored === "light" || stored === "dark") return stored;
  // Default to system preference
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

function applyTheme(mode: ThemeMode) {
  const root = document.documentElement;
  if (mode === "dark") {
    root.classList.add("dark");
  } else {
    root.classList.remove("dark");
  }
}

function App() {
  const [activePage, setActivePage] = useState("lora-data");
  const [darkMode, setDarkMode] = useState<ThemeMode>(getInitialTheme);

  // Apply theme on mount and when it changes
  useEffect(() => {
    applyTheme(darkMode);
    localStorage.setItem("modhandler-theme", darkMode);
  }, [darkMode]);

  const toggleTheme = useCallback(() => {
    setDarkMode((prev) => (prev === "dark" ? "light" : "dark"));
  }, []);

  const PageComponent = pages[activePage] || LoRaDataPage;

  return (
    <div className="flex h-screen overflow-hidden bg-background transition-colors duration-300">
      <Sidebar
        activePage={activePage}
        onNavigate={setActivePage}
        darkMode={darkMode === "dark"}
        onToggleTheme={toggleTheme}
      />
      <main className="flex-1 overflow-y-auto p-6">
        <PageComponent />
      </main>
    </div>
  );
}

export default App;
