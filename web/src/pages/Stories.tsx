import { useQueryClient } from "@tanstack/react-query";
import { BookHeartIcon, MoreVerticalIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { useSearchParams } from "react-router-dom";
import CustomIconPicker from "@/components/CustomIconPicker";
import { uploadService } from "@/components/MemoEditor/edit/services/uploadService";
import StoryMemoLink from "@/components/StoryMemoLink";
import StoryMemoPicker from "@/components/StoryMemoPicker";
import StoryTimeline from "@/components/StoryTimeline";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { memoServiceClient, storyServiceClient } from "@/connect";
import useCurrentUser from "@/hooks/useCurrentUser";
import { storyKeys, useStories, useStoryEvents } from "@/hooks/useStories";
import { storyMemoName } from "@/lib/story-memo";
import type { Story, StoryEvent } from "@/types/proto/api/story_service_pb";

const Stories = () => {
  const client = useQueryClient();
  const archives = useStories();
  const [params, setParams] = useSearchParams();
  const selected = params.get("archive") ?? undefined;
  const setSelected = (name: string | undefined) =>
    setParams((previous) => {
      const next = new URLSearchParams(previous);
      if (name) next.set("archive", name);
      else next.delete("archive");
      return next;
    });
  const events = useStoryEvents(selected);
  const [form, setForm] = useState<{ kind: "archive" | "event"; archive?: Story; event?: StoryEvent; parentEvent?: string }>();
  const [shareURL, setShareURL] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [tag, setTag] = useState("");
  const [reverse, setReverse] = useState(true);
  const active = archives.data?.find((s) => s.name === selected);
  const reload = () => client.invalidateQueries({ queryKey: storyKeys.all });
  const share = async (enabled: boolean) => {
    if (!confirm(enabled ? "生成新的只读链接？持有链接的人可以查看此档案的全部经历及图片，旧链接将失效。" : "撤销此档案的分享链接？"))
      return;
    setBusy(true);
    try {
      const result = await storyServiceClient.setStoryShare({ name: selected, enabled });
      setShareURL(result.token ? `${window.location.origin}/story-shares/${result.token}` : "");
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const remove = async (event?: StoryEvent) => {
    if (!confirm(event ? "删除这段经历？此操作无法撤销。" : "删除此档案及全部经历？此操作无法撤销。")) return;
    setBusy(true);
    try {
      if (event) await storyServiceClient.deleteStoryEvent({ name: event.name, revision: event.revision });
      else {
        await storyServiceClient.deleteStory({ name: selected });
        setShareURL("");
        setSelected(undefined);
      }
      await reload();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const flatEntries = [...(events.data ?? [])]
    .filter((e) => !tag || e.tags.includes(tag))
    .sort((a, b) => (reverse ? -1 : 1) * (a.eventDate + a.eventTime).localeCompare(b.eventDate + b.eventTime));
  const entries = flatEntries
    .filter((e) => !e.parentEvent)
    .flatMap((e) => [
      e,
      ...flatEntries
        .filter((a) => a.parentEvent === e.uid)
        .sort((a, b) => (a.eventDate + a.eventTime).localeCompare(b.eventDate + b.eventTime)),
    ]);
  return (
    <main className="mx-auto flex w-full max-w-5xl flex-col gap-5 px-4 py-6">
      <header className="flex items-start justify-between gap-3">
        <div>
          {active && (
            <Button
              variant="ghost"
              onClick={() => {
                setShareURL("");
                setSelected(undefined);
                setTag("");
              }}
            >
              ← 全部事迹
            </Button>
          )}
          <h1 className="flex items-center gap-2 text-2xl font-semibold">
            {active ? <span>{active.icon}</span> : <BookHeartIcon className="size-6" />}
            {active?.title ?? "事迹"}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">{active?.description ?? "珍藏人生中的经历与转折。"}</p>
        </div>
        <Button onClick={() => setForm({ kind: active ? "event" : "archive" })}>
          <PlusIcon className="size-4" />
          {active ? "记录经历" : "新建档案"}
        </Button>
      </header>
      {(error || archives.error || events.error) && (
        <p role="alert" className="text-destructive">
          {error || String(archives.error || events.error)}
          <Button
            variant="ghost"
            onClick={() => {
              setError("");
              void reload();
            }}
          >
            重试
          </Button>
        </p>
      )}
      {(archives.isLoading || (selected && events.isLoading)) && <p role="status">正在读取…</p>}
      {!selected ? (
        <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {archives.data?.map((s) => (
            <button
              key={s.name}
              className="rounded-xl border border-border bg-card p-5 text-left transition hover:bg-muted/40"
              onClick={() => {
                setShareURL("");
                setSelected(s.name);
              }}
            >
              <span className="text-2xl">{s.icon}</span>
              <h2 className="mt-3 font-medium">{s.title}</h2>
              <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{s.description}</p>
            </button>
          ))}
          {archives.data?.length === 0 && (
            <p className="col-span-full rounded-xl border border-dashed p-10 text-center text-muted-foreground">
              创建一个档案，记录第一段经历。
            </p>
          )}
        </section>
      ) : (
        <>
          <div className="flex flex-wrap gap-2">
            <select aria-label="标签筛选" className="rounded border bg-background p-2" value={tag} onChange={(e) => setTag(e.target.value)}>
              <option value="">全部标签</option>
              {[...new Set(events.data?.flatMap((e) => e.tags))].map((t) => (
                <option key={t}>{t}</option>
              ))}
            </select>
            <Button variant="outline" onClick={() => setReverse(!reverse)}>
              {reverse ? "从晚到早" : "从早到晚"}
            </Button>
            <Button variant="ghost" onClick={() => setForm({ kind: "archive", archive: active })}>
              档案设置
            </Button>
            <Button disabled={busy} variant="outline" onClick={() => void share(true)}>
              生成只读分享
            </Button>
            <Button disabled={busy} variant="ghost" onClick={() => void share(false)}>
              撤销分享
            </Button>
            <Button disabled={busy} variant="ghost" onClick={() => void remove()}>
              删除档案
            </Button>
          </div>
          {shareURL && (
            <label className="block text-sm">
              分享链接
              <input readOnly value={shareURL} className="mt-1 w-full rounded border p-2" onFocus={(e) => e.target.select()} />
            </label>
          )}
          <StoryTimeline entries={entries} color={active?.color} unit={active?.intervalUnit}>
            {(e) => (
              <article
                key={e.name}
                style={{ borderLeftColor: active?.color }}
                className="rounded-xl border border-border border-l-[3px] bg-card p-4 shadow-sm"
              >
                <header className="flex items-start gap-2">
                  <DropdownMenu>
                    <DropdownMenuTrigger
                      render={
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          className="-ml-2 -mt-1 shrink-0"
                          aria-label={`经历操作：${e.title || "记录这一刻"}`}
                          disabled={busy}
                        />
                      }
                    >
                      <MoreVerticalIcon className="size-4" />
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="start">
                      {!e.parentEvent && (
                        <DropdownMenuItem onClick={() => setForm({ kind: "event", parentEvent: e.uid })}>
                          <PlusIcon />
                          追加记录
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem onClick={() => setForm({ kind: "event", event: e })}>
                        <PencilIcon />
                        编辑
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem variant="destructive" onClick={() => void remove(e)}>
                        <Trash2Icon />
                        删除
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <div className="min-w-0 flex-1">
                    <h2 className="break-words font-bold leading-6">{e.title || "记录这一刻"}</h2>
                    <time className="mt-1 block text-xs text-muted-foreground">
                      {e.parentEvent ? "追加 · " : ""}
                      {e.eventDate} {e.eventTime}
                    </time>
                  </div>
                </header>
                <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-6">{e.content}</p>
                <div className="grid grid-cols-3 gap-2 empty:hidden [&:not(:empty)]:mt-3">
                  {e.attachmentUids.map((uid) => (
                    <a key={uid} href={`/file/attachments/${encodeURIComponent(uid)}`} target="_blank" rel="noreferrer">
                      <img
                        alt="经历图片"
                        className="aspect-square w-full rounded-lg object-cover"
                        src={`/file/attachments/${encodeURIComponent(uid)}`}
                        loading="lazy"
                      />
                    </a>
                  ))}
                </div>
                <div className="flex flex-wrap items-center gap-2 empty:hidden [&:not(:empty)]:mt-3">
                  {e.tags.map((t) => (
                    <button key={t} onClick={() => setTag(t)} className="text-xs text-muted-foreground">
                      #{t}
                    </button>
                  ))}
                </div>
                {e.memo && <StoryMemoLink name={e.memo} from={`/stories?archive=${encodeURIComponent(selected ?? "")}`} />}
              </article>
            )}
          </StoryTimeline>
          {entries.length === 0 && !events.isLoading && <p>还没有符合条件的经历。</p>}
        </>
      )}
      {form && (
        <EntryForm
          parentEvent={form.parentEvent}
          kind={form.kind}
          archive={form.archive}
          event={form.event}
          parent={selected}
          onClose={() => setForm(undefined)}
          onSaved={async () => {
            await reload();
            setForm(undefined);
          }}
        />
      )}
    </main>
  );
};
const EntryForm = ({
  parentEvent,
  kind,
  archive,
  event,
  parent,
  onClose,
  onSaved,
}: {
  kind: "archive" | "event";
  archive?: Story;
  event?: StoryEvent;
  parent?: string;
  parentEvent?: string;
  onClose: () => void;
  onSaved: () => Promise<void>;
}) => {
  const user = useCurrentUser();
  const draftKey = `memos.story-draft:${user?.name}:${kind}:${archive?.name ?? event?.name ?? parentEvent ?? parent ?? "new"}`;
  const [title, setTitle] = useState(archive?.title ?? event?.title ?? "");
  const [content, setContent] = useState(archive?.description ?? event?.content ?? "");
  const [date, setDate] = useState(event?.eventDate ?? new Date().toLocaleDateString("en-CA"));
  const [time, setTime] = useState(event?.eventTime ?? "");
  const [tags, setTags] = useState(event?.tags.join("，") ?? "");
  const [icon, setIcon] = useState(archive?.icon ?? "📖");
  const [color, setColor] = useState(archive?.color ?? "#55A9BE");
  const [intervalUnit, setIntervalUnit] = useState(archive?.intervalUnit || "days");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [images, setImages] = useState(event?.attachmentUids ?? []);
  const [memo, setMemo] = useState(event?.memo.replace(/^memos\//, "") ?? "");
  const [restored, setRestored] = useState(false);
  useEffect(() => {
    try {
      const raw = localStorage.getItem(draftKey);
      if (raw) {
        const d = JSON.parse(raw);
        if (d.revision === (event?.revision ?? 0)) {
          setTitle(d.title);
          setContent(d.content);
          setDate(d.date);
          setTime(d.time);
          setTags(d.tags);
          setIcon(d.icon);
          setColor(d.color);
          setIntervalUnit(d.intervalUnit ?? "days");
          setImages(d.images);
          setMemo(d.memo ?? "");
        } else {
          setError("旧草稿与当前版本不同，未自动覆盖。原草稿仍保留。");
          return;
        }
      }
    } catch {
      setError("无法读取草稿");
    }
    setRestored(true);
  }, [draftKey, event?.revision]);
  useEffect(() => {
    if (!restored) return;
    const leave = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    window.addEventListener("beforeunload", leave);
    try {
      localStorage.setItem(
        draftKey,
        JSON.stringify({ title, content, date, time, tags, icon, color, intervalUnit, images, memo, revision: event?.revision ?? 0 }),
      );
    } catch {
      setError("草稿无法保存，请不要关闭页面。");
    }
    return () => window.removeEventListener("beforeunload", leave);
  }, [restored, draftKey, title, content, date, time, tags, icon, color, intervalUnit, images, memo, event?.revision]);
  const close = () => {
    if (!busy && confirm("关闭编辑？未提交内容会保留为本机草稿。")) onClose();
  };
  const upload = async (files: File[]) => {
    if (images.length + files.length > 6) {
      setError("最多六张图片");
      return;
    }
    setBusy(true);
    try {
      for (const file of files) {
        const attachment = await uploadService.uploadFile({ file, previewUrl: "" });
        setImages((previous) => [...previous, attachment.name.split("/").at(-1)!]);
      }
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  const field = "mt-1 w-full rounded-md border border-input bg-background px-3 py-2";
  const save = async () => {
    setBusy(true);
    setError("");
    try {
      if (kind === "archive") {
        const story = { name: archive?.name, title, description: content, icon, color, intervalUnit };
        if (archive) await storyServiceClient.updateStory({ story });
        else await storyServiceClient.createStory({ story });
      } else {
        const memoName = storyMemoName(memo, window.location.origin);
        if (memoName === undefined) throw new Error("请输入有效的 Memo ID 或本站 Memo 链接。");
        if (memoName) {
          try {
            const linked = await memoServiceClient.getMemo({ name: memoName });
            if (linked.creator !== user?.name) throw new Error("只能关联自己的 Memo。");
          } catch {
            throw new Error("无法关联此 Memo，请确认它存在、属于自己且网络连接正常。");
          }
        }
        const value = {
          name: event?.name,
          story: parent,
          title,
          content,
          eventDate: date,
          eventTime: time,
          tags: tags
            .split(/[,，]/)
            .map((t) => t.trim())
            .filter(Boolean),
          revision: event?.revision ?? 0,
          attachmentUids: images,
          parentEvent: event?.parentEvent ?? parentEvent ?? "",
          memo: memoName,
        };
        if (event) await storyServiceClient.updateStoryEvent({ event: value });
        else await storyServiceClient.createStoryEvent({ parent, event: value });
      }
      localStorage.removeItem(draftKey);
      await onSaved();
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) close();
      }}
    >
      <DialogContent className="max-h-[90dvh] overflow-y-auto">
        <DialogTitle>{kind === "archive" ? "档案设置" : "记录经历"}</DialogTitle>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            void save();
          }}
        >
          <fieldset disabled={busy} className="space-y-3">
            <label className="block">
              {kind === "archive" ? "名称" : "标题"}
              <input
                className={field}
                maxLength={120}
                required={kind === "archive"}
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </label>
            {kind === "event" ? (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <label>
                  发生日期
                  <input required type="date" className={field} value={date} onChange={(e) => setDate(e.target.value)} />
                </label>
                <label>
                  时间（可选）
                  <input type="time" className={field} value={time} onChange={(e) => setTime(e.target.value)} />
                </label>
              </div>
            ) : (
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <p className="mb-2 text-sm">档案 Emoji</p>
                  <CustomIconPicker
                    emojiOnly
                    label="选择档案 Emoji"
                    value={{ value: { case: "emoji", value: icon } }}
                    onChange={(next) => setIcon(next?.value.case === "emoji" ? next.value.value : "📖")}
                  />
                </div>
                <label>
                  线条颜色
                  <input type="color" className={field} value={color} onChange={(e) => setColor(e.target.value)} />
                </label>
              </div>
            )}
            {kind === "archive" && (
              <label className="block">
                间隔单位
                <select aria-label="间隔单位" className={field} value={intervalUnit} onChange={(e) => setIntervalUnit(e.target.value)}>
                  <option value="days">天数</option>
                  <option value="hours">小时</option>
                </select>
                <p className="mt-1 text-xs text-muted-foreground">
                  天数按日期计算；小时按填写的时间计算，最多保留一位小数。未填写时间时显示天数。
                </p>
              </label>
            )}
            <label className="block">
              {kind === "archive" ? "简介" : "正文"}
              <textarea
                aria-label={kind === "archive" ? "简介" : "正文"}
                className={field}
                rows={6}
                maxLength={12000}
                value={content}
                onChange={(e) => setContent(e.target.value)}
              />
            </label>
            {kind === "event" && (
              <label className="block">
                标签
                <input className={field} value={tags} onChange={(e) => setTags(e.target.value)} placeholder="用逗号分隔" />
              </label>
            )}
            {error && (
              <p role="alert" className="text-destructive">
                {error}
              </p>
            )}
            {kind === "event" && <StoryMemoPicker value={memo} onChange={setMemo} />}
            {kind === "event" && (
              <div>
                <label>
                  图片（最多六张）
                  <input
                    type="file"
                    accept="image/jpeg,image/png,image/webp"
                    multiple
                    onChange={(e) => void upload(Array.from(e.target.files ?? []))}
                  />
                </label>
                <div className="mt-2 grid grid-cols-3 gap-2">
                  {images.map((uid) => (
                    <div key={uid}>
                      <img
                        alt="已上传图片"
                        className="aspect-square w-full rounded object-cover"
                        src={`/file/attachments/${encodeURIComponent(uid)}`}
                      />
                      <Button
                        type="button"
                        variant="ghost"
                        disabled={images.indexOf(uid) === 0}
                        onClick={() => {
                          const copy = [...images];
                          const i = copy.indexOf(uid);
                          [copy[i - 1], copy[i]] = [copy[i], copy[i - 1]];
                          setImages(copy);
                        }}
                      >
                        前移
                      </Button>
                      <Button type="button" variant="ghost" onClick={() => setImages(images.filter((v) => v !== uid))}>
                        移除
                      </Button>
                    </div>
                  ))}
                </div>
              </div>
            )}
            <footer className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={close}>
                取消
              </Button>
              <Button type="submit">{busy ? "保存中…" : "保存"}</Button>
            </footer>
          </fieldset>
        </form>
      </DialogContent>
    </Dialog>
  );
};
export default Stories;
