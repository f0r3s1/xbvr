// rows of placeholder cards needed to fill the visible list area while it loads
export function skeletonRows (perRow, heightPerWidth, infoHeight) {
  const listWidth = Math.max(320, window.innerWidth - 380)
  const cardHeight = (listWidth / perRow) * heightPerWidth + infoHeight
  return Math.max(2, Math.ceil((window.innerHeight - 120) / cardHeight))
}
