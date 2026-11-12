import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, expect, it, vi } from "vitest";
import StoryMemoLink from "@/components/StoryMemoLink";

const state = vi.hoisted(() => ({
  data: undefined as { snippet: string } | undefined,
  isLoading: false,
  isUnavailable: false,
  refetch: vi.fn(),
}));
vi.mock("@/hooks/useMemoQueries", () => ({ useMemo: () => state }));
beforeEach(() => {
  state.data = undefined;
  state.isLoading = false;
  state.isUnavailable = false;
  state.refetch.mockClear();
});

function Location() {
  const location = useLocation();
  return <output>{`${location.pathname}|${location.state?.from ?? ""}`}</output>;
}

it("opens the linked Memo with the archive as its return destination", () => {
  state.data = { snippet: "Graduation memories" };
  render(
    <MemoryRouter>
      <StoryMemoLink name="memos/abc" from="/stories?archive=stories%2Fone" />
      <Location />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole("link", { name: /Graduation memories/ }));
  expect(screen.getByRole("status")).toHaveTextContent("/memos/abc|/stories?archive=stories%2Fone");
});

it("does not offer a broken link when the Memo is unavailable", () => {
  state.isUnavailable = true;
  render(
    <MemoryRouter>
      <StoryMemoLink name="memos/abc" from="/stories" />
    </MemoryRouter>,
  );
  expect(screen.queryByRole("link")).not.toBeInTheDocument();
  expect(screen.getByText("关联 Memo 已删除或无法访问")).toBeInTheDocument();
});

it("allows retrying a temporary failure", () => {
  render(
    <MemoryRouter>
      <StoryMemoLink name="memos/abc" from="/stories" />
    </MemoryRouter>,
  );
  fireEvent.click(screen.getByRole("button", { name: "重试" }));
  expect(state.refetch).toHaveBeenCalledOnce();
});
