import { FiMessageSquare } from "react-icons/fi";
import { TbSparkles } from "react-icons/tb";
import { useParams } from "react-router";
import { AiChatList } from "../components/AiChatList";
import { AiChatView } from "../components/AiChatView";
import { Button } from "../components/Button";
import { EmptyState } from "../components/EmptyState";
import { InlineError } from "../components/InlineError";
import { PageHeader } from "../components/PageHeader";
import { Skeleton } from "../components/Skeleton";
import { useAiConnections } from "../hooks/useAi";
import { openSettings } from "../hooks/useSettingsDialog";
import { useTranslate } from "../hooks/useTranslate";

export default function AssistantPage() {
    const t = useTranslate();
    const { chatId } = useParams();
    const connections = useAiConnections();

    return (
        <>
            <PageHeader
                breadcrumb={[
                    {
                        label: t("assistant.title"),
                        icon: <FiMessageSquare aria-hidden />,
                    },
                ]}
            />
            {connections.isPending ? (
                <div className="flex flex-col gap-3 p-6">
                    <Skeleton className="h-8 w-1/3" />
                    <Skeleton className="h-24" />
                </div>
            ) : connections.isError ? (
                <div className="p-6">
                    <InlineError
                        message={t("common.error")}
                        onRetry={() => void connections.refetch()}
                    />
                </div>
            ) : connections.data.length === 0 && !chatId ? (
                <EmptyState
                    icon={TbSparkles}
                    title={t("assistant.noConnection.title")}
                    description={t("assistant.noConnection.description")}
                    action={
                        <Button
                            variant="primary"
                            size="lg"
                            onClick={() => openSettings("ai")}
                        >
                            {t("assistant.noConnection.action")}
                        </Button>
                    }
                />
            ) : (
                <div className="flex min-h-0 flex-1">
                    <AiChatList activeChatId={chatId ?? null} />
                    <AiChatView chatId={chatId ?? null} />
                </div>
            )}
        </>
    );
}
