import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { search } from "../api/client";

const DEBOUNCE_MS = 200;

// Searches the workspace as the user types. The query is debounced, and the
// previous results stay on screen while the next ones load.
export function useSearch(query: string) {
    const trimmed = query.trim();
    const [debounced, setDebounced] = useState(trimmed);

    useEffect(() => {
        const id = setTimeout(() => setDebounced(trimmed), DEBOUNCE_MS);
        return () => clearTimeout(id);
    }, [trimmed]);

    return useQuery({
        queryKey: ["search", debounced],
        queryFn: ({ signal }) => search(debounced, signal),
        enabled: debounced !== "",
        placeholderData: keepPreviousData,
        staleTime: 10_000,
    });
}
