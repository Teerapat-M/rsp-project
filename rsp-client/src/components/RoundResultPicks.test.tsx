import { createElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { describe, expect, it } from 'vitest'
import { RoundResultPicks, roundSideBadge, type RoundPicks } from './RoundResultPicks'

function sideHtml(html: string, side: 'you' | 'opponent'): string {
  const start = html.indexOf(`data-side="${side}"`)
  const next = html.indexOf('data-side="', start + 1)
  return html.slice(start, next === -1 ? undefined : next)
}

function badges(html: string, side: 'you' | 'opponent'): string[] {
  const chunk = sideHtml(html, side)
  const out: string[] = []
  if (chunk.includes('>Winner<')) out.push('Winner')
  if (chunk.includes('>Lost<')) out.push('Lost')
  return out
}

function render(picks: RoundPicks): string {
  return renderToStaticMarkup(createElement(RoundResultPicks, { roundResult: picks }))
}

describe('round result badges', () => {
  it('shows Winner and Lost on the correct cards from both players after a decided round', () => {
    const winnerView: RoundPicks = {
      youWon: true,
      draw: false,
      yourMove: 'paper',
      opponentMove: 'rock',
    }
    const loserView: RoundPicks = {
      youWon: false,
      draw: false,
      yourMove: 'rock',
      opponentMove: 'paper',
    }

    const winnerHtml = render(winnerView)
    expect(badges(winnerHtml, 'you')).toEqual(['Winner'])
    expect(badges(winnerHtml, 'opponent')).toEqual(['Lost'])

    const loserHtml = render(loserView)
    expect(badges(loserHtml, 'you')).toEqual(['Lost'])
    expect(badges(loserHtml, 'opponent')).toEqual(['Winner'])
  })

  it('shows no badge on either card from both players after a draw', () => {
    const alice: RoundPicks = {
      youWon: false,
      draw: true,
      yourMove: 'scissors',
      opponentMove: 'scissors',
    }
    const bob: RoundPicks = {
      youWon: false,
      draw: true,
      yourMove: 'scissors',
      opponentMove: 'scissors',
    }

    for (const html of [render(alice), render(bob)]) {
      expect(badges(html, 'you')).toEqual([])
      expect(badges(html, 'opponent')).toEqual([])
    }
  })

  it('roundSideBadge is symmetric across You/Opponent for win, lose, and draw', () => {
    expect(roundSideBadge(true, false, 'you')).toBe('Winner')
    expect(roundSideBadge(true, false, 'opponent')).toBe('Lost')
    expect(roundSideBadge(false, false, 'you')).toBe('Lost')
    expect(roundSideBadge(false, false, 'opponent')).toBe('Winner')
    expect(roundSideBadge(false, true, 'you')).toBeNull()
    expect(roundSideBadge(false, true, 'opponent')).toBeNull()
    expect(roundSideBadge(true, true, 'you')).toBeNull()
    expect(roundSideBadge(true, true, 'opponent')).toBeNull()
  })
})
