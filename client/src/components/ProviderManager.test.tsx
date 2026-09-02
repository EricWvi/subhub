import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../api";
import { renderWithProviders } from "../test/render";
import ProviderManager from "./ProviderManager";

describe("ProviderManager", () => {
  it("loads providers through the injected API client", async () => {
    const apiClient = vi.fn<ApiClient>(async () =>
      Response.json({
        providers: [
          {
            id: 1,
            name: "Alpha",
            url: "https://example.com/subscription",
            refresh_interval_minutes: 120,
            abbrev: "A",
            auto_fetch: true,
            used: 0,
            total: 0,
            expire: 0,
            created_at: "2026-09-02T00:00:00+08:00",
            updated_at: "2026-09-02T13:05:09+08:00",
          },
        ],
      }),
    );

    renderWithProviders(<ProviderManager />, { apiClient });

    expect(await screen.findByText("Alpha")).toBeInTheDocument();
    expect(screen.getByText("2026-09-02 13:05:09")).toBeInTheDocument();
    expect(apiClient).toHaveBeenCalledWith("/api/providers");
  });

  it("opens the add-provider form", async () => {
    const user = userEvent.setup();

    renderWithProviders(<ProviderManager />);
    await user.click(screen.getByRole("button", { name: /add provider/i }));

    expect(
      screen.getByRole("dialog", { name: "Add Provider" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByLabelText("URL")).toBeInTheDocument();
  });
});
