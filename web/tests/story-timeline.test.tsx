import { create } from "@bufbuild/protobuf";
import { render, screen } from "@testing-library/react";
import { expect, it } from "vitest";
import StoryTimeline from "@/components/StoryTimeline";
import { StoryEventSchema } from "@/types/proto/api/story_service_pb";

it("connects main experiences without counting later additions as milestones", () => {
  const entries = [
    create(StoryEventSchema, { name: "storyEvents/a", uid: "a", title: "First", eventDate: "2024-06-20" }),
    create(StoryEventSchema, { name: "storyEvents/c", uid: "c", title: "Reflection", parentEvent: "a", eventDate: "2025-01-01" }),
    create(StoryEventSchema, { name: "storyEvents/b", uid: "b", title: "Second", eventDate: "2024-06-22" }),
  ];
  render(<StoryTimeline entries={entries}>{(entry) => <article>{entry.title}</article>}</StoryTimeline>);
  expect(screen.getAllByLabelText("经历间隔")).toHaveLength(1);
  expect(screen.getByLabelText("经历间隔")).toHaveTextContent("2 天");
  expect(screen.getAllByRole("article").map((el) => el.textContent)).toEqual(["First", "Reflection", "Second"]);
});
