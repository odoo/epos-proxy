import { useCallback, useContext, useState } from "react";
import { ToastContext } from "../contexts/ToastContext";
import { errorText } from "../error";

export interface CopyOptions {
  key?: string;
  showToast?: boolean;
}

export function useClipboard(timeout = 2000) {
  const toastContext = useContext(ToastContext);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  const copy = useCallback(
    async (text: string, label = "Text", options?: CopyOptions | string) => {
      const opts: CopyOptions =
        typeof options === "string" ? { key: options } : options || {};
      const activeKey = opts.key || label;
      const shouldShowToast = opts.showToast ?? true;

      try {
        await navigator.clipboard.writeText(text);
        setCopiedKey(activeKey);
        if (shouldShowToast) {
          toastContext.actions.showToast(`${label} copied to clipboard!`, "success");
        }

        setTimeout(() => {
          setCopiedKey((curr) => (curr === activeKey ? null : curr));
        }, timeout);
        return true;
      } catch (err) {
        toastContext.actions.showToast(
          `Copy failed: ${errorText(err, "unknown error")}`,
          "danger",
        );
        return false;
      }
    },
    [toastContext, timeout],
  );

  return {
    copiedKey,
    copy,
    isCopied: (key: string) => copiedKey === key,
  };
}

