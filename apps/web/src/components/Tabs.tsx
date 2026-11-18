type TabsProps<T extends string> = {
    value: T;
    onChange: (tab: T) => void;
    tabs: { id: T; label: string }[];
};

export function Tabs<T extends string>({ value, onChange, tabs }: TabsProps<T>) {
    return (
        <div role="tablist" className="flex gap-1">
            {tabs.map((tab) => (
                <button
                    key={tab.id}
                    type="button"
                    role="tab"
                    aria-selected={value === tab.id}
                    onClick={() => onChange(tab.id)}
                    className="cursor-pointer border-b-2 border-transparent px-3 py-2 text-sm text-gray-500 hover:text-gray-900 aria-selected:border-accent aria-selected:font-medium aria-selected:text-gray-900"
                >
                    {tab.label}
                </button>
            ))}
        </div>
    );
}
