import { useState } from "react";
import { useProjectPlans, useProjects } from "../hooks/useProjects";
import { useTranslate } from "../hooks/useTranslate";
import { Button } from "./Button";
import { Dialog } from "./Dialog";
import { SelectInput } from "./SelectInput";

type PlanPickerDialogProps = {
    // "project" picks a whole project, "plan" one plan of a project.
    mode: "project" | "plan";
    title: string;
    // The label is the project's name, or "Project / Plan".
    onPick: (id: string, label: string) => void;
    onClose: () => void;
};

export function PlanPickerDialog({
    mode,
    title,
    onPick,
    onClose,
}: PlanPickerDialogProps) {
    const t = useTranslate();
    const projects = useProjects();
    const [projectId, setProjectId] = useState("");
    const plans = useProjectPlans(mode === "plan" ? projectId : "");
    const [planId, setPlanId] = useState("");

    const project = projects.data?.find((p) => p.id === projectId);
    const plan = plans.data?.find((p) => p.id === planId);
    const choose = (label: string) => ({ value: "", label: `— ${label} —` });

    function pick() {
        if (!project) return;
        if (mode === "project") onPick(project.id, project.name);
        else if (plan) onPick(plan.id, `${project.name} / ${plan.name}`);
    }

    return (
        <Dialog
            open
            onClose={onClose}
            title={title}
            footer={
                <>
                    <Button size="lg" onClick={onClose}>
                        {t("common.cancel")}
                    </Button>
                    <Button
                        variant="primary"
                        size="lg"
                        disabled={mode === "project" ? !project : !plan}
                        onClick={pick}
                    >
                        {t("assistant.picker.submit")}
                    </Button>
                </>
            }
        >
            <div className="flex flex-col gap-3">
                <SelectInput
                    label={t("resources.link.project")}
                    value={projectId}
                    onChange={(event) => {
                        setProjectId(event.target.value);
                        setPlanId("");
                    }}
                    options={[
                        choose(t("resources.link.project")),
                        ...(projects.data ?? []).map((p) => ({
                            value: p.id,
                            label: p.icon ? `${p.icon} ${p.name}` : p.name,
                        })),
                    ]}
                />
                {mode === "plan" && (
                    <SelectInput
                        label={t("resources.link.plan")}
                        value={planId}
                        disabled={projectId === ""}
                        onChange={(event) => setPlanId(event.target.value)}
                        options={[
                            choose(t("resources.link.plan")),
                            ...(plans.data ?? []).map((p) => ({
                                value: p.id,
                                label: p.name,
                            })),
                        ]}
                    />
                )}
            </div>
        </Dialog>
    );
}
