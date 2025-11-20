export const fmtUSD = (v, digits = 2) => {
    const n = typeof v === 'string' ? parseFloat(v) : v
    if (!isFinite(n)) return '-'
    return n.toLocaleString(undefined, { style: 'currency', currency: 'USD', maximumFractionDigits: digits })
}

export const fmtAmount = (v, digits = 8) => {
    const n = typeof v === 'string' ? parseFloat(v) : v
    if (!isFinite(n)) return '-'
    return n.toLocaleString(undefined, { maximumFractionDigits: digits })
}

export const uniq = (arr) => Array.from(new Set(arr))
