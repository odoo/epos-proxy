import { PDFDocument, StandardFonts, rgb } from "pdf-lib";

const fontSize = 14;
const margin = 50;
const lineHeight = fontSize + 5;
const pageHeight = 841.89;

export async function createPdfBytes(text, duplex) {
  const pdfDoc = await PDFDocument.create();
  const font = await pdfDoc.embedFont(StandardFonts.Courier);

  const lines = wrapText(text, 500, font, fontSize);
  const maxLinesPage1 = duplex === "duplex" ? 1 : 3;

  const page1Lines = lines.slice(0, maxLinesPage1);
  const page2Lines = lines.slice(maxLinesPage1);

  let page1 = pdfDoc.addPage([595.28, pageHeight]);
  let y = pageHeight - margin;

  for (const line of page1Lines) {
    page1.drawText(line, {
      x: margin,
      y,
      size: fontSize,
      font,
      color: rgb(0, 0, 0),
    });
    y -= lineHeight;
  }

  if (!page2Lines.length) return pdfDoc.save();

  let page2 = pdfDoc.addPage([595.28, pageHeight]);
  y = pageHeight - margin;

  for (const line of page2Lines) {
    page2.drawText(line, {
      x: margin,
      y,
      size: fontSize,
      font,
      color: rgb(0, 0, 0),
    });
    y -= lineHeight;
  }

  return await pdfDoc.save();
}

function wrapText(text, maxWidth, font, fontSize) {
  const words = text.split(" ");
  let lines = [];
  let currentLine = "";

  for (let word of words) {
    const testLine = currentLine ? currentLine + " " + word : word;
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
