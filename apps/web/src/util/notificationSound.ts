let context: AudioContext | null = null;

// A short two-note chime made with Web Audio, so no sound file is needed.
// Browsers may keep audio muted until the user has interacted with the page;
// then this does nothing.
export function playNotificationSound() {
    try {
        context ??= new AudioContext();
        const start = context.currentTime;
        [880, 1320].forEach((frequency, i) => {
            const ctx = context as AudioContext;
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();
            const at = start + i * 0.12;
            osc.type = "sine";
            osc.frequency.value = frequency;
            gain.gain.setValueAtTime(0.0001, at);
            gain.gain.exponentialRampToValueAtTime(0.15, at + 0.02);
            gain.gain.exponentialRampToValueAtTime(0.0001, at + 0.3);
            osc.connect(gain).connect(ctx.destination);
            osc.start(at);
            osc.stop(at + 0.32);
        });
    } catch {
        // No audio available.
    }
}
