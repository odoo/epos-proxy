export async function copyPrinterFieldValue(printer, field = 'ip', { copiedIds, showToast }) {
  try {
    await navigator.clipboard.writeText(printer[field]);
    (copiedIds.value[printer.id] ||= {})[field] = true;
    setTimeout(() => copiedIds.value[printer.id][field] = false, 2000);
  } catch (err) {
    showToast('Copy failed:' + err)
  }
}

async function sendEposPrint(printerIp, name) {
  return await fetch(`http://${printerIp}/cgi-bin/epos/service.cgi`, {
    method: 'POST',
    body: `<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
          <s:Body>
            <epos-print xmlns="http://www.epson-pos.com/schemas/2011/03/epos-print">
              <feed line="1" />
              <text font="font_e" em="true"/>
              <text align="center">This is a test receipt ${name}</text>
              <feed line="10" />
              <cut type="feed" />
            </epos-print>
          </s:Body>
        </s:Envelope>`
  })
}

export async function handleTestPrint(printer, { testPrintIds, showToast }) {
  testPrintIds.value[printer.id] = true
  try {
    return await executePrint(printer, showToast)
  } finally {
    testPrintIds.value[printer.id] = false
  }
}

async function executePrint(printer, showToast) {
  try {
      const response = await sendEposPrint(printer.ip, printer.name)
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

      showToast(`Test print sent`, 'success')
  } catch (err) {
    showToast(`Test failed: ${err.message}`, 'error')
  }
}
