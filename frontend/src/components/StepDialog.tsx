import { useContext, useEffect, useMemo, useState } from "react";
import { BrowserOpenURL } from "../../wailsjs/runtime/runtime";
import type { Step } from "../types";
import { useStepDialog } from "../hooks/useStepDialog";
import { useClipboard } from "../hooks/useClipboard";
import Dialog from "./Dialog";
import { brewSteps, linuxSteps, zadigSteps } from "../assets/data/fixStep";
import { AppContext } from "../contexts/AppContext";

type ContentPhase = "shown" | "leaving" | "entering";

const CONTENT_PHASE_CLASS: Record<ContentPhase, string> = {
  leaving: "opacity-0 -translate-x-4 transition duration-200 ease-in",
  entering: "opacity-0 translate-x-4",
  shown: "opacity-100 translate-x-0 transition duration-300 ease-out",
};

export default function StepDialog({ printerName }: { printerName: string }) {
  const appContext = useContext(AppContext);
  const [contentEl, setContentEl] = useState<HTMLDivElement | null>(null);
  const { currentStep, next, back, setCurrentStep } = useStepDialog();
  const [displayStep, setDisplayStep] = useState(0);
  const [phase, setPhase] = useState<ContentPhase>("shown");
  const [contentHeight, setContentHeight] = useState("auto");
  const { copy, isCopied } = useClipboard();

  useEffect(() => {
    if (currentStep === displayStep) {
      return;
    }

    setPhase("leaving");
    const timeout = setTimeout(() => {
      setDisplayStep(currentStep);
      setPhase("entering");
    }, 200);

    return () => clearTimeout(timeout);
  }, [currentStep, displayStep]);

  useEffect(() => {
    if (!contentEl) {
      // Drop the pinned height so the next open measures from scratch instead
      // of clipping the first step against the last one's height.
      setContentHeight("auto");
      return;
    }

    const observer = new ResizeObserver(() =>
      setContentHeight(`${contentEl.offsetHeight}px`),
    );

    observer.observe(contentEl);
    return () => observer.disconnect();
  }, [contentEl]);

  useEffect(() => {
    if (phase !== "entering") {
      return;
    }

    let inner = 0;
    const outer = requestAnimationFrame(() => {
      inner = requestAnimationFrame(() => setPhase("shown"));
    });

    return () => {
      cancelAnimationFrame(outer);
      cancelAnimationFrame(inner);
    };
  }, [phase]);

  const fixSteps = useMemo<Step[]>(() => {
    if (appContext.data.isWindows) {
      return zadigSteps(printerName);
    }

    if (appContext.data.isMac) {
      return brewSteps(printerName);
    }

    if (appContext.data.isLinux) {
      return linuxSteps(printerName);
    }

    return [];
  }, [appContext.data.os]);

  const step = fixSteps[displayStep];
  if (!step) {
    return null;
  }

  const getFixErrorText = () => {
    if (appContext.data.isWindows) {
      return "Fix - Install WinUSB driver";
    }

    if (appContext.data.isMac || appContext.data.isLinux) {
      return "Fix - Install libusb";
    }

    return "";
  };

  return (
    <Dialog
      openButton={
        <div className="flex-1 border bg-odoo text-white hover:bg-odoo-dark rounded-lg px-4 py-2 text-center cursor-pointer">
          {getFixErrorText()}
        </div>
      }
      title={step.title}
      onOpen={() => setCurrentStep(0)}
    >
      <div
        className="overflow-hidden transition-[height] duration-300 ease-in-out"
        style={{ height: contentHeight }}
      >
        <div ref={setContentEl} className={`${CONTENT_PHASE_CLASS[phase]}`}>
          <p className="text-gray-500 whitespace-pre-line">{step.desc}</p>

          {step.link && (
            <a
              href={step.link}
              target="_blank"
              rel="noreferrer"
              onClick={(event) => {
                event.preventDefault();
                BrowserOpenURL(step.link!);
              }}
              className="inline-flex items-center mt-3 px-3 py-2 rounded-lg border border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400"
            >
              {step.linkLabel}
            </a>
          )}

          {step.image && (
            <img
              src={step.image}
              alt={step.title}
              className="max-w-full mt-3 rounded-lg"
            />
          )}

          {step.codes?.map((code, index) => (
            <div key={index} className="mt-3 relative group">
              <pre className="bg-slate-800 text-emerald-500 text-sm rounded-lg px-4 py-3 overflow-x-auto font-mono">
                {code}
              </pre>
              <button
                className="absolute top-2.5 right-2 px-2 py-1 text-xs rounded-md bg-slate-700 text-slate-300 hover:bg-slate-600 opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer"
                onClick={() => copy(code, "Command", `code-${index}`)}
              >
                {isCopied(`code-${index}`) ? "✓ Copied" : "Copy"}
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="flex gap-2 pt-5">
        {currentStep > 0 && (
          <button
            className="flex-1 py-2 rounded-lg border border-stone-300 text-stone-600 hover:bg-stone-50 hover:border-stone-400 cursor-pointer whitespace-nowrap transition-colors"
            onClick={back}
          >
            Back
          </button>
        )}
        {currentStep < fixSteps.length - 1 && (
          <button
            className="flex-1 py-2 rounded-lg bg-odoo text-white hover:bg-odoo-dark whitespace-nowrap cursor-pointer transition-colors"
            onClick={() => next(fixSteps.length)}
          >
            Next
          </button>
        )}
      </div>
    </Dialog>
  );
}
