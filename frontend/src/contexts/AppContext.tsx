import { createContext, useEffect, useState } from "react";
import { main } from "../../wailsjs/go/models";
import { AppVariable } from "../../wailsjs/go/main/App";

const RETRY_INTERVAL = 5000;

type AppContextType = {
  setters: {};
  data: {
    os: string | null;
    app: main.AppVariable | null;
    isWindows: boolean;
    isMac: boolean;
    isLinux: boolean;
    serverIsRunning: boolean;
  };
  actions: {};
};

export const AppContext = createContext({} as AppContextType);

interface AppContextWrapper {
  children: React.ReactNode;
}

export const AppContextWrapper = ({ children }: AppContextWrapper) => {
  const [app, setApp] = useState<main.AppVariable | null>(null);

  const os = app?.os || null;
  const data = {
    app,
    os,
    isWindows: os === "windows",
    isMac: os === "darwin",
    isLinux: os === "linux",
    serverIsRunning: app?.serverRunning ?? false,
  };
  const setters = {};
  const actions = {};

  useEffect(() => {
    let cancelled = false;
    let retryId: number | null = null;

    const fetchAppContext = async () => {
      try {
        const variables = await AppVariable();
        if (cancelled) {
          return;
        }

        setApp(variables);
      } catch (error) {
        console.error("Failed to fetch app context:", error);
        if (!cancelled) {
          retryId = window.setTimeout(fetchAppContext, RETRY_INTERVAL);
        }
      }
    };

    fetchAppContext();

    return () => {
      cancelled = true;
      if (retryId !== null) {
        clearTimeout(retryId);
      }
    };
  }, []);

  return (
    <AppContext.Provider value={{ data, setters, actions }}>
      {children}
    </AppContext.Provider>
  );
};
