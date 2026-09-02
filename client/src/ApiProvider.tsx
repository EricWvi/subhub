import type { ReactNode } from "react";
import { ApiClientContext, type ApiClient } from "./api";

interface ApiProviderProps {
  children: ReactNode;
  client: ApiClient;
}

const ApiProvider = ({ children, client }: ApiProviderProps) => (
  <ApiClientContext.Provider value={client}>
    {children}
  </ApiClientContext.Provider>
);

export default ApiProvider;
