import { useState } from "react";
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

function App() {
  const [activePage, setActivePage] = useState("lora-data");

  const PageComponent = pages[activePage] || LoRaDataPage;

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      <Sidebar activePage={activePage} onNavigate={setActivePage} />
      <main className="flex-1 overflow-y-auto p-6">
        <PageComponent />
      </main>
    </div>
  );
}

export default App;
