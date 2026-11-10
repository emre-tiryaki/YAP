type FormErrorProps = {
    message: string | null;
};

export function FormError({ message }: FormErrorProps) {
    if (!message) return null;
    return (
        <p role="alert" className="text-sm text-red-600">
            {message}
        </p>
    );
}
