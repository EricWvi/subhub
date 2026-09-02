import { createContext, useContext } from "react";

export type ApiClient = (
  input: RequestInfo | URL,
  init?: RequestInit,
) => Promise<Response>;

const defaultApiClient: ApiClient = (input, init) => fetch(input, init);

export const ApiClientContext = createContext<ApiClient>(defaultApiClient);

export const useApiClient = () => useContext(ApiClientContext);
