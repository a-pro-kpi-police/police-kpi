#!/usr/bin/osascript -l JavaScript

ObjC.import("Foundation")

function unwrap(value) {
  return ObjC.unwrap(value)
}

function csvCell(value) {
  const text = value === null || value === undefined ? "" : String(value)
  return `"${text.replace(/"/g, '""')}"`
}

function csvRow(values) {
  return `${values.map(csvCell).join(",")}\n`
}

function normalizedText(value) {
  return value === null || value === undefined ? "" : String(value).trim()
}

function numericValue(value, fallback = 0) {
  return typeof value === "number" && Number.isFinite(value) ? value : fallback
}

function writeUtf8(path, contents) {
  const text = $.NSString.alloc.initWithUTF8String(contents)
  if (!text.writeToFileAtomicallyEncodingError(path, true, $.NSUTF8StringEncoding, null)) {
    throw new Error(`Unable to write ${path}`)
  }
}

function temporaryWorkbookCopy(sourcePath) {
  const manager = $.NSFileManager.defaultManager
  const directory = unwrap($.NSTemporaryDirectory())
  const copyPath = `${directory}kpi-police-${unwrap($.NSUUID.UUID.UUIDString)}.numbers`
  if (!manager.copyItemAtPathToPathError(sourcePath, copyPath, null)) {
    throw new Error("Unable to create a temporary workbook copy")
  }
  return copyPath
}

function run(argv) {
  if (argv.length !== 3) {
    throw new Error(
      "Usage: export_monthly_numbers.js WORKBOOK MONTHLY_CSV CATALOG_CSV",
    )
  }

  const [sourcePath, monthlyPath, catalogPath] = argv
  const copyPath = temporaryWorkbookCopy(sourcePath)
  const numbers = Application("Numbers")
  const document = numbers.open(Path(copyPath))

  try {
    const monthlyTable = document
      .sheets.byName("Месячная_таблица-1")
      .tables[0]
    const catalogTable = document
      .sheets.byName("Справочник_KPI")
      .tables[0]

    const month = normalizedText(monthlyTable.rows[1].cells.value()[1])
    if (!/^\d{4}-\d{2}$/.test(month)) {
      throw new Error(`Invalid reporting month: ${month}`)
    }

    const codeRow = monthlyTable.rows[3].cells.value()
    const blockRow = monthlyTable.rows[4].cells.value()
    const pointsRow = monthlyTable.rows[5].cells.value()
    const massRow = monthlyTable.rows[6].cells.value()
    const criticalRow = monthlyTable.rows[7].cells.value()
    const limitRow = monthlyTable.rows[8].cells.value()
    const firstKpiColumn = codeRow.findIndex((value) => normalizedText(value))

    if (firstKpiColumn < 0) {
      throw new Error("No KPI columns found in Месячная_таблица-1")
    }

    const codes = codeRow
      .slice(firstKpiColumn)
      .map(normalizedText)
      .filter(Boolean)
    if (codes.length !== monthlyTable.columnCount() - firstKpiColumn) {
      throw new Error("KPI code row contains gaps")
    }
    if (new Set(codes).size !== codes.length) {
      throw new Error("KPI code row contains duplicates")
    }

    const catalogByCode = {}
    for (let rowIndex = 1; rowIndex < catalogTable.rowCount(); rowIndex += 1) {
      const values = catalogTable.rows[rowIndex].cells.value()
      const code = normalizedText(values[0])
      if (!code) continue
      catalogByCode[code] = {
        code,
        block: normalizedText(values[1]),
        subblock: normalizedText(values[2]),
        indicator: normalizedText(values[3]),
        npa: normalizedText(values[4]),
        unit: normalizedText(values[5]),
        sourceUrl: normalizedText(values[14]),
      }
    }

    let catalogCsv = csvRow([
      "code",
      "block",
      "subblock",
      "indicator",
      "npa",
      "points_per_unit",
      "source_url",
      "unit",
      "monthly_limit",
      "is_mass",
      "is_critical",
    ])
    codes.forEach((code, offset) => {
      const columnIndex = firstKpiColumn + offset
      const catalogItem = catalogByCode[code]
      if (!catalogItem) {
        throw new Error(`KPI ${code} is missing from Справочник_KPI`)
      }
      catalogCsv += csvRow([
        code,
        normalizedText(blockRow[columnIndex]).replace(/^\d+\.\s*/, ""),
        catalogItem.subblock,
        catalogItem.indicator,
        catalogItem.npa,
        numericValue(pointsRow[columnIndex]),
        catalogItem.sourceUrl,
        catalogItem.unit,
        numericValue(limitRow[columnIndex]),
        numericValue(massRow[columnIndex]) === 1 ? 1 : 0,
        numericValue(criticalRow[columnIndex]) === 1 ? 1 : 0,
      ])
    })

    let monthlyCsv = csvRow([
      "source_row",
      "month",
      "employee_number",
      "full_name",
      "position",
      "coefficient",
      ...codes,
    ])
    const seenNames = new Set()
    for (let rowIndex = 10; rowIndex < monthlyTable.rowCount(); rowIndex += 1) {
      const values = monthlyTable.rows[rowIndex].cells.value()
      const fullName = normalizedText(values[1])
      if (!fullName) continue
      if (seenNames.has(fullName)) {
        throw new Error(`Duplicate employee name: ${fullName}`)
      }
      seenNames.add(fullName)

      const employeeNumber = numericValue(values[0], rowIndex - 9)
      const coefficient = numericValue(values[3], 1) || 1
      const quantities = codes.map((code, offset) => {
        const value = numericValue(values[firstKpiColumn + offset], 0)
        if (value < 0) {
          throw new Error(
            `Negative quantity for ${fullName}, ${code}, row ${rowIndex + 1}`,
          )
        }
        return value
      })
      monthlyCsv += csvRow([
        rowIndex + 1,
        month,
        employeeNumber,
        fullName,
        normalizedText(values[2]),
        coefficient,
        ...quantities,
      ])
    }

    writeUtf8(monthlyPath, monthlyCsv)
    writeUtf8(catalogPath, catalogCsv)

    return JSON.stringify({
      month,
      employees: seenNames.size,
      kpis: codes.length,
      monthlyPath,
      catalogPath,
    })
  } finally {
    document.close({ saving: "no" })
    $.NSFileManager.defaultManager.removeItemAtPathError(copyPath, null)
  }
}
