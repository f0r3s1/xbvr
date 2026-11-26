// sizes the label column of a filter panel to its longest label, so translated labels are not cut short
const MIN = 84
const MAX_SHARE = 0.45
const PADDING = 24

function fit (el) {
  const labels = el.querySelectorAll('.field.is-floating-label .label')
  if (!labels.length) return
  const widest = Math.max(...[...labels].map(l => l.scrollWidth))
  const col = Math.min(Math.max(widest + PADDING, MIN), el.clientWidth * MAX_SHARE)
  el.style.setProperty('--panel-label-col', `${Math.round(col)}px`)
}

export default {
  mounted: el => requestAnimationFrame(() => fit(el)),
  updated: el => requestAnimationFrame(() => fit(el))
}
