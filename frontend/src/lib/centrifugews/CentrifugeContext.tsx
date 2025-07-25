// CentrifugeContext.tsx
import React, {createContext, useContext, useRef} from "react";
import CentrifugeManager from "./CentrifugeManager";

interface ICentrifugeContext {
  subscribe: CentrifugeManager['subscribe'];
  unsubscribe: CentrifugeManager['unsubscribe'];
  publish: CentrifugeManager['publish'];
  getConnectionState: CentrifugeManager['getConnectionState'];
}

type Props = {
  url: string;
  options?: object;
  children: React.ReactNode;
};

const CentrifugeContext = createContext<ICentrifugeContext | undefined>(undefined);

export const CentrifugeProvider: React.FC<Props> = ({url, options, children}) => {
  const managerRef = useRef<CentrifugeManager>();

  if (!managerRef.current) {
    managerRef.current = CentrifugeManager.getInstance(url, options);
  }

  const contextValue: ICentrifugeContext = {
    subscribe: managerRef.current.subscribe.bind(managerRef.current),
    unsubscribe: managerRef.current.unsubscribe.bind(managerRef.current),
    publish: managerRef.current.publish.bind(managerRef.current),
    getConnectionState: managerRef.current.getConnectionState.bind(managerRef.current),
  };

  return (
    <CentrifugeContext.Provider value={contextValue}>
      {children}
    </CentrifugeContext.Provider>
  );
};

export function useCentrifuge() {
  const ctx = useContext(CentrifugeContext);
  if (!ctx) throw new Error("useCentrifuge must be used within a CentrifugeProvider");
  return ctx;
}