interface Props {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  disabled?: boolean;
}

/** Interruptor de encendido y apagado; `label` es lo que lee el lector de pantalla. */
export function Switch({ checked, onChange, label, disabled }: Props) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative h-8 w-14 shrink-0 rounded-full outline-none focus-visible:ring-2 focus-visible:ring-movo-accent disabled:opacity-60 ${
        checked ? "bg-movo-ok" : "bg-movo-slate/40"
      }`}
    >
      <span
        aria-hidden="true"
        className={`absolute top-1 size-6 rounded-full bg-white shadow transition-[left] ${
          checked ? "left-7" : "left-1"
        }`}
      />
    </button>
  );
}
