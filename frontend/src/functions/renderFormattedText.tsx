export function renderFormattedText(text: string) {
  if (!text) return null;
  const parts = text.split(/\*([^*]+)\*/g);
  return parts.map((part, index) =>
    index % 2 === 1 ? (
      <span key={index} className="font-bold text-odoo">
        {part}
      </span>
    ) : (
      part
    ),
  );
}
