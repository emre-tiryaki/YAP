import { FiAlertTriangle } from "react-icons/fi";
import { ApiError } from "../api/client";
import { useDateFormat } from "../hooks/useDateFormat";
import { useRepoCommits } from "../hooks/useGitHub";
import { errorKey } from "../hooks/useResourceOperations";
import { useTranslate } from "../hooks/useTranslate";
import { layoutCommitGraph, type GraphRow } from "../util/commitGraph";

const ROW = 44;
const LANE = 16;
const PAD = 10;
const DOT = 4.5;
// Lane colors, picked by column.
const COLORS = ["#2563eb", "#16a34a", "#d97706", "#db2777", "#7c3aed", "#0891b2", "#dc2626", "#65a30d"];

function x(lane: number) {
    return PAD + lane * LANE;
}

function color(lane: number) {
    return COLORS[lane % COLORS.length];
}

// A line from one lane to another, bending in the middle.
function link(fromLane: number, fromY: number, toLane: number, toY: number) {
    const x1 = x(fromLane);
    const x2 = x(toLane);
    if (x1 === x2) return `M ${x1} ${fromY} L ${x2} ${toY}`;
    const mid = (fromY + toY) / 2;
    return `M ${x1} ${fromY} C ${x1} ${mid}, ${x2} ${mid}, ${x2} ${toY}`;
}

function GraphCell({ row, width }: { row: GraphRow; width: number }) {
    const mid = ROW / 2;
    const { column, above, below, into, outOf, commit } = row;
    return (
        <svg width={width} height={ROW} aria-hidden className="shrink-0">
            {above.map((sha, i) =>
                sha !== null && sha !== commit.sha && below[i] === sha ? (
                    <path key={`pass-${i}`} d={link(i, 0, i, ROW)} stroke={color(i)} strokeWidth={2} fill="none" />
                ) : null,
            )}
            {into.map((i) => (
                <path key={`in-${i}`} d={link(i, 0, column, mid)} stroke={color(i)} strokeWidth={2} fill="none" />
            ))}
            {outOf.map((k) => (
                <path key={`out-${k}`} d={link(column, mid, k, ROW)} stroke={color(k)} strokeWidth={2} fill="none" />
            ))}
            <circle
                cx={x(column)}
                cy={mid}
                r={DOT}
                fill={commit.parents.length > 1 ? "white" : color(column)}
                stroke={color(column)}
                strokeWidth={2}
            />
        </svg>
    );
}

// The history of a repository's branch as a graph, like `git log --graph`:
// merged branches show as lanes joining the main line.
export function CommitGraph({ resourceId, commit }: { resourceId: string; commit: string | null }) {
    const t = useTranslate();
    const format = useDateFormat();
    const commits = useRepoCommits(resourceId, commit);

    if (commits.isLoading) return <p className="text-sm text-gray-400">{t("common.loading")}</p>;
    // A failed "load more" keeps the commits already shown.
    if (commits.isError && !commits.data)
        return (
            <p role="alert" className="flex items-center gap-2 text-sm text-amber-800">
                <FiAlertTriangle aria-hidden className="shrink-0" />
                {commits.error instanceof ApiError && commits.error.message === "github_unavailable"
                    ? t("resources.repo.commits.offline")
                    : t(errorKey(commits.error))}
            </p>
        );

    const rows = layoutCommitGraph(commits.data?.pages.flat() ?? []);
    if (rows.length === 0) return <p className="text-sm text-gray-400">{t("resources.repo.commits.none")}</p>;
    const lanes = Math.max(1, ...rows.map((r) => Math.max(r.above.length, r.below.length, r.column + 1)));
    const width = PAD * 2 + (lanes - 1) * LANE;

    return (
        <div className="flex flex-col">
            <ol className="flex flex-col">
                {rows.map((row) => {
                    const c = row.commit;
                    const [title, ...rest] = c.message.split("\n");
                    return (
                        <li key={c.sha} className="flex items-center gap-3 hover:bg-gray-50">
                            <GraphCell row={row} width={width} />
                            <div className="flex min-w-0 flex-1 items-center gap-3 py-1 pr-2">
                                {c.avatarUrl ? (
                                    <img src={c.avatarUrl} alt="" className="size-6 shrink-0 rounded-full" />
                                ) : (
                                    <span aria-hidden className="size-6 shrink-0 rounded-full bg-gray-200" />
                                )}
                                <div className="min-w-0 flex-1">
                                    <p className="truncate text-sm text-gray-900" title={rest.join("\n").trim() || title}>
                                        {title}
                                    </p>
                                    <p className="truncate text-xs text-gray-500">
                                        {c.authorLogin || c.authorName}
                                        {" · "}
                                        <time dateTime={c.date.toISOString()} title={format.dateTime(c.date)}>
                                            {format.relative(c.date) ?? format.dateTime(c.date)}
                                        </time>
                                        {c.parents.length > 1 && ` · ${t("resources.repo.commits.merge")}`}
                                    </p>
                                </div>
                                <a
                                    href={c.htmlUrl}
                                    target="_blank"
                                    rel="noreferrer"
                                    title={t("resources.repo.openOnGitHub")}
                                    className="shrink-0 rounded px-1.5 py-0.5 font-mono text-xs text-gray-600 hover:bg-gray-200"
                                >
                                    {c.sha.slice(0, 7)}
                                </a>
                            </div>
                        </li>
                    );
                })}
            </ol>
            {commits.hasNextPage && (
                <button
                    type="button"
                    disabled={commits.isFetchingNextPage}
                    onClick={() => void commits.fetchNextPage()}
                    className="mt-3 cursor-pointer self-center rounded-md border border-gray-300 px-3 py-1.5 text-sm hover:bg-gray-100 disabled:opacity-50"
                >
                    {t(commits.isFetchingNextPage ? "common.loading" : "resources.repo.commits.more")}
                </button>
            )}
            {commits.isFetchNextPageError && (
                <p role="alert" className="mt-2 text-center text-sm text-red-600">
                    {t(errorKey(commits.error))}
                </p>
            )}
        </div>
    );
}
