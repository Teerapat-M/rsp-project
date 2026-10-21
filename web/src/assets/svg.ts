const RAW = import.meta.glob<string>('./icons/*.svg', {
  query: '?raw',
  eager: true,
  import: 'default',
})

export function svgAsset(name: string): string {
  const svg = RAW[`./icons/${name}.svg`]
  if (!svg) throw new Error(`Missing SVG ${name}`)
  return svg
}
