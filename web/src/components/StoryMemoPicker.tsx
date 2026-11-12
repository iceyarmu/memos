import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useInfiniteMemos, useMemo } from "@/hooks/useMemoQueries";
import { buildMemoCreatorFilter } from "@/lib/resource-names";
import { storyMemoName } from "@/lib/story-memo";

export default function StoryMemoPicker({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const user = useCurrentUser();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [debounced, setDebounced] = useState("");
  const [lookup, setLookup] = useState(value);
  useEffect(() => {
    const id = window.setTimeout(() => setDebounced(search.trim()), 300);
    return () => window.clearTimeout(id);
  }, [search]);
  useEffect(() => {
    const id = window.setTimeout(() => setLookup(value), 300);
    return () => window.clearTimeout(id);
  }, [value]);
  const name = storyMemoName(lookup, window.location.origin);
  const detail = useMemo(name ?? "", { enabled: !!name });
  const results = useInfiniteMemos(
    {
      pageSize: 10,
      filter: [buildMemoCreatorFilter(user?.name ?? ""), debounced ? `content.contains(${JSON.stringify(debounced)})` : ""]
        .filter(Boolean)
        .join(" && "),
    },
    { enabled: open && !!user },
  );
  return (
    <div className="space-y-2">
      <label className="block text-sm">
        关联 Memo（可选）
        <input
          aria-label="关联 Memo"
          className="mt-1 w-full rounded-md border border-input bg-background px-3 py-2"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="输入 Memo ID，或粘贴本站 Memo 链接"
          onBlur={() => {
            const parsed = storyMemoName(value, window.location.origin);
            if (parsed) onChange(parsed.slice(6));
          }}
        />
      </label>
      {lookup === value && value.trim() && (
        <div className="rounded-lg bg-muted/40 p-3 text-sm" aria-live="polite">
          {!name ? (
            "请输入有效的 Memo ID 或本站 Memo 链接。"
          ) : detail.isLoading ? (
            "正在读取 Memo…"
          ) : detail.data ? (
            detail.data.creator !== user?.name ? (
              "只能关联自己的 Memo。"
            ) : (
              <>
                <p className="text-xs text-muted-foreground">将关联此 Memo</p>
                <p className="mt-1 line-clamp-3 break-words">{detail.data.snippet || "图片或附件 Memo"}</p>
              </>
            )
          ) : detail.isUnavailable ? (
            "未找到 Memo，可能已删除或没有访问权限。"
          ) : (
            <span>
              暂时无法读取。
              <button type="button" className="ml-1 underline" onClick={() => void detail.refetch()}>
                重试
              </button>
            </span>
          )}
        </div>
      )}
      <div className="flex gap-2">
        <Button type="button" variant="outline" size="sm" aria-expanded={open} onClick={() => setOpen(!open)}>
          {open ? "收起 Memo 列表" : "从我的 Memo 中选择"}
        </Button>
        {value && (
          <Button type="button" variant="ghost" size="sm" onClick={() => onChange("")}>
            取消关联
          </Button>
        )}
      </div>
      {open && (
        <div className="space-y-2 rounded-lg border p-3">
          <input
            aria-label="搜索我的 Memo"
            placeholder="搜索正文，留空显示最近记录"
            className="w-full rounded-md border bg-background px-3 py-2 text-sm"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
          <div className="max-h-60 space-y-1 overflow-y-auto" aria-label="Memo 搜索结果">
            {results.isLoading ? (
              <p role="status" className="text-sm">
                正在读取…
              </p>
            ) : results.isError ? (
              <Button type="button" variant="ghost" onClick={() => void results.refetch()}>
                读取失败，重试
              </Button>
            ) : (
              results.data?.pages
                .flatMap((page) => page.memos)
                .map((memo) => (
                  <button
                    key={memo.name}
                    type="button"
                    className="block w-full rounded-md p-2 text-left hover:bg-muted focus-visible:outline-2 focus-visible:outline-ring"
                    onClick={() => {
                      onChange(memo.name.slice(6));
                      setOpen(false);
                    }}
                  >
                    <p className="line-clamp-2 break-words text-sm">{memo.snippet || "图片或附件 Memo"}</p>
                    <p className="mt-1 truncate text-xs text-muted-foreground">{memo.name.slice(6)}</p>
                  </button>
                ))
            )}
            {results.isSuccess && !results.data.pages.some((page) => page.memos.length) && (
              <p className="text-sm text-muted-foreground">没有匹配的 Memo，试试其他关键词。</p>
            )}
            {results.hasNextPage && (
              <Button type="button" variant="ghost" disabled={results.isFetchingNextPage} onClick={() => void results.fetchNextPage()}>
                加载更多
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
