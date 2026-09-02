import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import type { ApiClient } from "../api";
import { renderWithProviders } from "../test/render";
import NodeManager from "./NodeManager";

describe("NodeManager", () => {
  it("loads custom nodes through the injected API client", async () => {
    const apiClient = vi.fn<ApiClient>(async () =>
      Response.json({
        nodes: [
          {
            id: 1,
            name: "Office",
            config: "type: socks5",
            created_at: "2026-09-02T00:00:00+08:00",
            updated_at: "2026-09-02T13:05:09+08:00",
          },
        ],
      }),
    );

    renderWithProviders(<NodeManager />, { apiClient });

    expect(await screen.findByText("Office")).toBeInTheDocument();
    expect(screen.getByText("2026-09-02 13:05:09")).toBeInTheDocument();
    expect(apiClient).toHaveBeenCalledWith("/api/nodes");
  });

  it("opens the add-node form", async () => {
    const user = userEvent.setup();

    renderWithProviders(<NodeManager />);
    await user.click(screen.getByRole("button", { name: /add node/i }));

    expect(screen.getByRole("dialog", { name: "Add Node" })).toBeInTheDocument();
    expect(screen.getByLabelText("Name")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toHaveTextContent("Config");
  });
});
