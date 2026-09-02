import { App, ConfigProvider } from "antd";
import {
  render,
  type RenderOptions,
  type RenderResult,
} from "@testing-library/react";
import type { ReactElement, ReactNode } from "react";
import type { ApiClient } from "../api";
import ApiProvider from "../ApiProvider";

interface TestRenderOptions extends Omit<RenderOptions, "wrapper"> {
  apiClient?: ApiClient;
}

const emptyApiClient: ApiClient = async () => Response.json({});

export const renderWithProviders = (
  element: ReactElement,
  { apiClient = emptyApiClient, ...options }: TestRenderOptions = {},
): RenderResult => {
  const Wrapper = ({ children }: { children: ReactNode }) => (
    <ApiProvider client={apiClient}>
      <ConfigProvider>
        <App>{children}</App>
      </ConfigProvider>
    </ApiProvider>
  );

  return render(element, { wrapper: Wrapper, ...options });
};
