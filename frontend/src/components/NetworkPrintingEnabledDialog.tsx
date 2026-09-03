import { useEffect, useState } from "react";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { GetTroubleshootInfo } from "../../wailsjs/go/main/App";
import { staticIpAdvice } from "../assets/data/troubleshootStep";
import { renderFormattedText } from "../functions/renderFormattedText";
import Dialog, { type ActionType } from "./Dialog";

export default function NetworkPrintingEnabledDialog() {
  const [openSignal, setOpenSignal] = useState(0);
  const [localIp, setLocalIp] = useState<string | null>(null);

  useEffect(() => {
    return EventsOn("network-printing-changed", (enabled: boolean) => {
      if (!enabled) {
        return;
      }

      setOpenSignal((count) => count + 1);
      GetTroubleshootInfo()
        .then((info) => setLocalIp(info?.localIp ?? null))
        .catch((err) => console.error("Failed to load troubleshoot info", err));
    });
  }, []);

  return (
    <Dialog
      title="Allow Other Devices to Print"
      openSignal={openSignal}
      actions={[{ name: "ok", label: "Got it", variant: "primary" as ActionType }]}
    >
      <p className="text-gray-600 text-sm leading-relaxed">
        Devices on this network will be able to send print jobs to your printers.
      </p>
      {localIp && (
        <p className="text-gray-600 text-sm whitespace-pre-line leading-relaxed mt-3">
          {renderFormattedText(staticIpAdvice(localIp))}
        </p>
      )}
    </Dialog>
  );
}
