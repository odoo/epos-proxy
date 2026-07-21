export async function copyPrinterFieldValue(printer, field, copiedIds) {
    await navigator.clipboard.writeText(printer[field]);
    (copiedIds[printer.id] ||= {})[field] = true;
    setTimeout(() => copiedIds[printer.id][field] = false, 2000);
}

async function sendEposPrint(printer, openCashDrawer = false) {
  const content = openCashDrawer
    ? '<pulse />'
    : `
        <text font="font_e" em="true"/>
        <text align="center">This is a test receipt ${printer.name}</text>
        <feed line="3" />
        <cut type="feed" />
      `

  return await fetch(`http://${printer.ip}/cgi-bin/epos/service.cgi`, {
    method: 'POST',
    body: `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
          <s:Body>
            <epos-print xmlns="http://www.epson-pos.com/schemas/2011/03/epos-print">
            ${content}
            </epos-print>
          </s:Body>
        </s:Envelope>`
  })
}

async function sendLabelPrint(printer) {
  const zpl = `
^XA
^PW800
^LL250
^CF0,35
^FO0,40^FB800,1,0,C^FDTEST PRINT^FS
^CF0,25
^FO0,100^FB800,1,0,C^FD${printer.name}^FS
^XZ`;

  return await fetch(`http://${printer.ip}/pstprnt`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/octet-stream' },
    body: new TextEncoder().encode(zpl),
  })
}

export async function executePrint(printer, openCashDrawer = false) {
  if (printer.type === 'label') {
    const response = await sendLabelPrint(printer)
    if (!response.ok) {
      if (response.status === 429) {
        throw new Error('Printer queue is full, please try again')
      }
      throw new Error(`Label print failed (HTTP ${response.status})`)
    }
    return
  }

  const response = await sendEposPrint(printer, openCashDrawer)
  const xml = await response.text()
  const parser = new DOMParser()
  const doc = parser.parseFromString(xml, 'text/xml')
  const responseEl = doc.querySelector('response')

  if (responseEl?.getAttribute('success') !== 'true') {
    const code = responseEl?.getAttribute('code') || 'Unknown error'
    if (code === 'EX_BADPORT') {
      throw new Error('The device is not connected, please check the printer power / connection')
    }
    throw new Error(code)
  }
}
