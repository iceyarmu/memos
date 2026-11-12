import type { ReactNode } from "react";
import { storyInterval } from "@/lib/story-interval";
import type { StoryEvent } from "@/types/proto/api/story_service_pb";

export default function StoryTimeline({
  entries,
  color = "#55A9BE",
  unit = "days",
  children,
}: {
  entries: StoryEvent[];
  color?: string;
  unit?: string;
  children: (entry: StoryEvent) => ReactNode;
}) {
  const main = entries.filter((e) => !e.parentEvent);
  return (
    <section aria-label="经历时间线" className="relative mx-auto w-full max-w-3xl">
      {main.length > 1 && (
        <div aria-hidden className="absolute bottom-6 left-8 top-6 w-[3px] rounded-full" style={{ backgroundColor: color }} />
      )}
      {main.map((entry, index) => (
        <div key={entry.name} className="relative">
          {index > 0 && (
            <div className="flex h-16 items-center pl-3">
              <span
                className="relative rounded-lg border border-border bg-card px-3 py-1 text-xs font-medium tabular-nums shadow-sm"
                aria-label="经历间隔"
              >
                {storyInterval(main[index - 1], entry, unit)}
              </span>
            </div>
          )}
          {children(entry)}
          {entries
            .filter((addition) => addition.parentEvent === entry.uid)
            .map((addition) => (
              <div key={addition.name} className="relative ml-8 border-l-[3px] py-4 pl-4" style={{ borderColor: color }}>
                {children(addition)}
              </div>
            ))}
        </div>
      ))}
    </section>
  );
}
