import { createContext, useCallback, useEffect, useState } from "react";
import { main } from "../../wailsjs/go/models";
import {
  ConfirmDisconnectOdoo,
  CheckOdooStatus,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

export type OdooContextType = {
  data: {
    status: main.OdooStatusInterface | null;
    isConnected: boolean;
  };
  actions: {
    refreshStatus: () => Promise<void>;
    disconnectOdoo: () => Promise<boolean>;
  };
};

export const OdooContext = createContext({} as OdooContextType);

interface OdooContextWrapperProps {
  children: React.ReactNode;
}

export const OdooContextWrapper = ({ children }: OdooContextWrapperProps) => {
  const [status, setStatus] = useState<main.OdooStatusInterface | null>(null);

  const refreshStatus = useCallback(async () => {
    try {
      const odooStatus = await CheckOdooStatus();
      setStatus(odooStatus);
    } catch (error) {
      console.error("Failed to fetch Odoo status:", error);
    }
  }, []);

  const disconnectOdoo = useCallback(async () => {
    try {
      const confirmed = await ConfirmDisconnectOdoo();
      if (confirmed) {
        setStatus((prev) => ({
          dbUrl: "",
          websocketStatus: "disconnected",
          lanStatus: "disconnected",
          appId: prev?.appId || "",
          ipAddress: prev?.ipAddress || "",
        }));
        await refreshStatus();
        return true;
      }
      return false;
    } catch (error) {
      console.error("Failed to disconnect Odoo:", error);
      return false;
    }
  }, [refreshStatus]);

  useEffect(() => {
    // Initial fetch on mount
    refreshStatus();

    // Listen to real-time status updates pushed from backend
    const unsubscribe = EventsOn("odoo:status_changed", (newStatus: main.OdooStatusInterface) => {
      setStatus(newStatus);
    });

    return () => {
      if (unsubscribe) {
        unsubscribe();
      }
    };
  }, [refreshStatus]);

  const isConnected = Boolean(status?.dbUrl);

  const data = {
    status,
    isConnected,
  };

  const actions = {
    refreshStatus,
    disconnectOdoo,
  };

  return (
    <OdooContext.Provider value={{ data, actions }}>
      {children}
    </OdooContext.Provider>
  );
};

