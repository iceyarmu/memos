import { describe, expect, it } from "vitest";
import { storyMemoName } from "@/lib/story-memo";

describe("story memo input", () => {
  const origin = "https://notes.example.com";
  it.each([
    " Abc-123_ ",
    "memos/Abc-123_",
    "/memos/Abc-123_",
    `${origin}/memos/Abc-123_?from=search#content`,
  ])("accepts %s as the same Memo", (input) => expect(storyMemoName(input, origin)).toBe("memos/Abc-123_"));
  it.each([
    "https://other.example.com/memos/abc",
    `${origin}/memos/shares/token`,
    "memos/abc/def",
    "javascript:alert(1)",
    "some words",
  ])("does not turn %s into an unrelated local Memo", (input) => expect(storyMemoName(input, origin)).toBeUndefined());
  it("allows clearing the link", () => expect(storyMemoName(" ", origin)).toBe(""));
});
