export const fmtUSD = (v) => {
    const n = Number(v)
    if (!isFinite(n) || n <= 0) return '—'
    const abs = Math.abs(n)
    const fmt = (x, unit = '') => '$' + (Number.isInteger(x) ? x.toFixed(0) : x.toFixed(2)) + unit
    if (abs >= 1e12) return fmt(n / 1e12, 'T')
    if (abs >= 1e9)  return fmt(n / 1e9,  'B')
    if (abs >= 1e6)  return fmt(n / 1e6,  'M')
    if (abs >= 1e3)  return fmt(n / 1e3,  'K')
    return fmt(n)
}

export const fmtAmount = (v, digits = 8) => {
    const n = typeof v === 'string' ? parseFloat(v) : v
    if (!isFinite(n)) return '-'
    return n.toLocaleString(undefined, { maximumFractionDigits: digits })
}

export const uniq = (arr) => Array.from(new Set(arr))
