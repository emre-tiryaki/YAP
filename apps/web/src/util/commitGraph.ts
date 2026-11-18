import type { RepoCommit } from "../types/github";

// One row of the commit graph. Lanes are columns; a lane carries the sha of
// the commit it is heading to, or null when it is free. Lanes never move, so
// a line that passes a row stays in its column.
export type GraphRow = {
    commit: RepoCommit;
    // The column of the commit's dot.
    column: number;
    // Lanes entering the row from above, and leaving it below.
    above: (string | null)[];
    below: (string | null)[];
    // Columns whose line ends in the dot (the commit's children).
    into: number[];
    // Columns the dot's lines go down to (its parents).
    outOf: number[];
};

function freeLane(lanes: (string | null)[]): number {
    const free = lanes.indexOf(null);
    return free < 0 ? lanes.length : free;
}

function trim(lanes: (string | null)[]): (string | null)[] {
    let end = lanes.length;
    while (end > 0 && lanes[end - 1] === null) end--;
    return lanes.slice(0, end);
}

// Lays out commits (newest first, as GitHub lists them) like `git log
// --graph`: a commit takes the lane that was waiting for it, its first parent
// continues in that lane and further parents (merged branches) get lanes of
// their own, unless a lane already waits for them.
export function layoutCommitGraph(commits: RepoCommit[]): GraphRow[] {
    let lanes: (string | null)[] = [];
    const rows: GraphRow[] = [];
    for (const commit of commits) {
        const above = lanes;
        const into: number[] = [];
        above.forEach((sha, i) => {
            if (sha === commit.sha) into.push(i);
        });
        // A commit nothing waits for (the branch tip) starts a lane.
        const column = into.length > 0 ? into[0] : freeLane(above);

        const next = [...above];
        for (const i of into) next[i] = null;
        const outOf: number[] = [];
        commit.parents.forEach((parent, n) => {
            const waiting = next.indexOf(parent);
            if (waiting >= 0) {
                outOf.push(waiting);
                return;
            }
            // The first parent continues the commit's own lane.
            const lane = n === 0 && (next[column] ?? null) === null ? column : freeLane(next);
            next[lane] = parent;
            outOf.push(lane);
        });
        lanes = trim(next);
        rows.push({ commit, column, above, below: lanes, into, outOf });
    }
    return rows;
}
