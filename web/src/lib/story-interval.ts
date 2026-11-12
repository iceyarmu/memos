import type { StoryEvent } from "@/types/proto/api/story_service_pb";

type EventDate = Pick<StoryEvent, "eventDate" | "eventTime">;

export function storyInterval(a: EventDate, b: EventDate, unit = "days"): string {
  // These are user-entered calendar dates and wall times, without a timezone.
  const days = Math.abs(Date.parse(`${a.eventDate}T00:00:00Z`) - Date.parse(`${b.eventDate}T00:00:00Z`)) / 86400000;
  if (unit !== "hours") return `${days} 天`;
  if (!a.eventTime || !b.eventTime) return `${days} 天 · 时间未填写`;
  const hours = Math.abs(Date.parse(`${a.eventDate}T${a.eventTime}:00Z`) - Date.parse(`${b.eventDate}T${b.eventTime}:00Z`)) / 3600000;
  return `${Number(hours.toFixed(1))} 小时`;
}
