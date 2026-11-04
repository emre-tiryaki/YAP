import { useQuery } from "@tanstack/react-query";
import { getWorkspace } from "../api/client";

export function useWorkspace() {
    return useQuery({queryKey: ['workspace'], queryFn: getWorkspace})
}