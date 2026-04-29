import { PDFDocument, StandardFonts, rgb } from "pdf-lib";

const fontSize = 14;

export async function createPdfBytes(text) {
  const pdfDoc = await PDFDocument.create();
  const page = pdfDoc.addPage([595.28, 841.89]);
  const font = await pdfDoc.embedFont(StandardFonts.Courier);
  const lines = wrapText(text, 500, font, fontSize);
  let y = 800;
  for (const line of lines) {
    page.drawText(line, {
      x: 50,
      y,
      size: fontSize,
      font,
      color: rgb(0, 0, 0),
    });
    y -= fontSize + 5;
  }
  return await pdfDoc.save();
}

function wrapText(text, maxWidth, font, fontSize) {
  const words = text.split(' ');
  let lines = [];
  let currentLine = '';

  for (let word of words) {
    const testLine = currentLine ? currentLine + ' ' + word : word;
    const width = font.widthOfTextAtSize(testLine, fontSize);

    if (width < maxWidth) {
      currentLine = testLine;
    } else {
      lines.push(currentLine);
      currentLine = word;
    }
  }

  if (currentLine) lines.push(currentLine);
  return lines;
}
