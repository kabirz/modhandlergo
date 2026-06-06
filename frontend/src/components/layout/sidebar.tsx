import React, { useState } from "react";
import { cn } from "@/lib/utils";
import { useI18n } from "@/lib/i18n";
import {
  Radio, Settings, Upload, Terminal, Sun, Moon, Languages, Info, X, type LucideIcon,
} from "lucide-react";

interface NavItem { id: string; labelKey: string; icon: LucideIcon; }

const navItems: NavItem[] = [
  { id: "lora-data", labelKey: "nav.loraData", icon: Radio },
  { id: "lora-config", labelKey: "nav.loraConfig", icon: Settings },
  { id: "firmware", labelKey: "nav.firmware", icon: Upload },
  { id: "can-command", labelKey: "nav.canCommand", icon: Terminal },
];

export const APP_VERSION = "0.1.1";

interface SidebarProps {
  activePage: string;
  onNavigate: (pageId: string) => void;
  darkMode: boolean;
  onToggleTheme: () => void;
}

function AboutDialog({ onClose }: { onClose: () => void }) {
  const { t } = useI18n();
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div className="bg-card border border-border rounded-xl shadow-2xl w-[380px] p-6 space-y-4"
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
            <span className="text-muted-foreground">{t("about.protocol")}</span>
            <span>USR1566 / LoRa / CAN 2.0</span>
          </div>
          <div className="border-t border-border pt-3 mt-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("about.origProject")}</span>
              <span className="font-mono text-xs">can-uart-tool</span>
            </div>
            <div className="flex justify-between mt-1">
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

export function Sidebar({ activePage, onNavigate, darkMode, onToggleTheme }: SidebarProps) {
  const { lang, setLang, t } = useI18n();
  const [showAbout, setShowAbout] = useState(false);

  return (
    <>
      <aside className="w-[200px] min-w-[200px] h-screen bg-sidebar border-r border-sidebar-border flex flex-col transition-colors duration-300">
        {/* Navigation */}
        <nav className="flex-1 py-2 px-2 space-y-1">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activePage === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onNavigate(item.id)}
                className={cn(
                  "w-full flex items-center gap-3 px-3 py-2.5 rounded-md text-sm transition-colors cursor-pointer",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium"
                    : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-accent-foreground"
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
            <span className="text-[11px] text-muted-foreground shrink-0">{t("sidebar.theme")}：</span>
            <button
              onClick={() => { if (darkMode) onToggleTheme(); }}
              className={cn(
                "flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
                !darkMode ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              <Sun className="h-3 w-3" />{t("sidebar.light")}
            </button>
            <button
              onClick={() => { if (!darkMode) onToggleTheme(); }}
              className={cn(
                "flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[11px] cursor-pointer transition-colors",
                darkMode ? "bg-primary text-primary-foreground" : "bg-background text-foreground border border-input hover:bg-muted"
              )}
            >
              <Moon className="h-3 w-3" />{t("sidebar.dark")}
            </button>
          </div>

          {/* Language */}
          <div className="flex items-center gap-1 px-2">
            <Languages className="h-3 w-3 text-muted-foreground shrink-0" />
            <span className="text-[11px] text-muted-foreground shrink-0">{t("sidebar.lang")}：</span>
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
              EN
            </button>
          </div>

          {/* Version + About */}
          <div className="flex items-center justify-between px-2">
            <span className="text-[11px] text-muted-foreground">
              {t("sidebar.version")}：v{APP_VERSION}
            </span>
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
    </>
  );
}
