import { describe, expect, it } from "vitest";
import { storyInterval } from "@/lib/story-interval";

const date = (eventDate: string, eventTime = "") => ({ eventDate, eventTime });
describe("story intervals", () => {
  it("counts calendar days including leap days and ignores time for day mode", () => {
    expect(storyInterval(date("2024-02-28", "23:00"), date("2024-03-01", "01:00"))).toBe("2 天");
    expect(storyInterval(date("2024-03-01", "23:00"), date("2024-03-01", "01:00"))).toBe("0 天");
  });
  it("uses positive hours in either direction and preserves fractional hours", () => {
    const a = date("2024-12-31", "23:30");
    const b = date("2025-01-01", "02:00");
    expect(storyInterval(a, b, "hours")).toBe("2.5 小时");
    expect(storyInterval(b, a, "hours")).toBe("2.5 小时");
  });
  it("does not invent a time for date-only events", () => {
    expect(storyInterval(date("2024-03-01"), date("2024-03-02", "12:00"), "hours")).toBe("1 天 · 时间未填写");
  });
});
