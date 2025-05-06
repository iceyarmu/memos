import { observer } from "mobx-react-lite";
import { useMemo } from "react";
import MemoView from "@/components/MemoView";
import PagedMemoList from "@/components/PagedMemoList";
import useCurrentUser from "@/hooks/useCurrentUser";
import { useMemoFilterStore } from "@/store/v1";
import { viewStore, userStore } from "@/store/v2";
import { Direction, State } from "@/types/proto/api/v1/common";
import { Memo } from "@/types/proto/api/v1/memo_service";

const RandomReview = observer(() => {
  const user = useCurrentUser();
  const memoFilterStore = useMemoFilterStore();
  const selectedShortcut = userStore.state.shortcuts.find((shortcut) => shortcut.id === memoFilterStore.shortcut);

  const { filter: memoListFilter, randomSeed } = useMemo(() => {
    const conditions = [];
    const contentSearch: string[] = [];
    const tagSearch: string[] = [];
    let randomSeed: number = Math.floor(Math.random() * 0x100000000);
    for (const filter of memoFilterStore.filters) {
      if (filter.factor === "contentSearch") {
        contentSearch.push(`"${filter.value}"`);
      } else if (filter.factor === "tagSearch") {
        tagSearch.push(`"${filter.value}"`);
      } else if (filter.factor === "pinned") {
        conditions.push(`pinned == true`);
      } else if (filter.factor === "property.hasLink") {
        conditions.push(`has_link == true`);
      } else if (filter.factor === "property.hasTaskList") {
        conditions.push(`has_task_list == true`);
      } else if (filter.factor === "property.hasCode") {
        conditions.push(`has_code == true`);
      } else if (filter.factor === "displayTime") {
        const filterDate = new Date(filter.value);
        const filterUtcTimestamp = filterDate.getTime() + filterDate.getTimezoneOffset() * 60 * 1000;
        const timestampAfter = filterUtcTimestamp / 1000;
        conditions.push(`display_time_after == ${timestampAfter}`);
        conditions.push(`display_time_before == ${timestampAfter + 60 * 60 * 24}`);
      } else if (filter.factor === "seed") {
        randomSeed = parseInt(filter.value);
      }
    }
    if (contentSearch.length > 0) {
      conditions.push(`content_search == [${contentSearch.join(", ")}]`);
    }
    if (tagSearch.length > 0) {
      conditions.push(`tag_search == [${tagSearch.join(", ")}]`);
    }
    return {
      filter: conditions.join(" && "),
      randomSeed,
    }
  }, [user, memoFilterStore.filters, viewStore.state.orderByTimeAsc]);

  return (
    <PagedMemoList
      renderer={(memo: Memo) => <MemoView key={`${memo.name}-${memo.displayTime}`} memo={memo} showVisibility showPinned compact />}
      listSort={(memos: Memo[]) => memos.filter((memo) => memo.state === State.NORMAL)}
      owner={user.name}
      direction={Direction.DIRECTION_UNSPECIFIED}
      filter={selectedShortcut?.filter || ""}
      oldFilter={memoListFilter}
      sort={"random:" + randomSeed}
    />
  );
});

export default RandomReview;
