import { useContext, useState } from "react";
import { ToastContext } from "../contexts/ToastContext";
import { printer } from "../../wailsjs/go/models";
import { errorText } from "../error";
import { executePrint } from "../functions/executePrint";
import { useClipboard } from "../hooks/useClipboard";

interface PrinterActionsProps {
  printer: printer.Device;
}

export default function PrinterActions({ printer }: PrinterActionsProps) {
  const toastContext = useContext(ToastContext);
  const { copy, isCopied } = useClipboard();
  const [isTestPrinting, setIsTestPrinting] = useState(false);
  const [isCashDrawerOpening, setIsCashDrawerOpening] = useState(false);

  const copiedIp = isCopied(printer.ip);

  function onCopy() {
    copy(printer.ip, "Printer IP", printer.ip);
  }

  async function onTest() {
    setIsTestPrinting(true);
    try {
      await executePrint(printer);
      toastContext.actions.showToast(
        `Test print sent to ${printer.name}`,
        "success",
      );
    } catch (err) {
      toastContext.actions.showToast(
        errorText(err, "Test print failed"),
        "danger",
      );
    } finally {
      setIsTestPrinting(false);
    }
  }

  async function onCashDrawerOpen() {
    setIsCashDrawerOpening(true);
    try {
      await executePrint(printer, true);
      toastContext.actions.showToast(
        `Cash drawer opened for ${printer.name}`,
        "success",
      );
    } catch (err) {
      toastContext.actions.showToast(
        errorText(err, "Failed to open the cash drawer"),
        "danger",
      );
    } finally {
      setIsCashDrawerOpening(false);
    }
  }

  return (
    <div className="flex gap-2 mt-4 flex-wrap">
      <button
        onClick={onCopy}
        className={`flex-1 border text-sm rounded-lg px-3 py-2 cursor-pointer whitespace-nowrap ${
          copiedIp
            ? "bg-success text-white"
            : "bg-odoo text-white hover:bg-odoo-dark"
        }`}
      >
        {copiedIp ? "✓ Copied!" : "Copy IP"}
      </button>

      <button
        onClick={onTest}
        disabled={isTestPrinting}
        className="flex-1 border rounded-lg text-sm px-3 py-2 cursor-pointer border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {isTestPrinting ? "Printing..." : "Test"}
      </button>

      {printer.type === "receipt" && (
        <button
          onClick={onCashDrawerOpen}
          disabled={isCashDrawerOpening}
          className="flex-1 break-keep border rounded-lg text-sm px-3 py-2 cursor-pointer border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isCashDrawerOpening ? "Opening..." : "Cash Drawer"}
        </button>
      )}
    </div>
  );
}
