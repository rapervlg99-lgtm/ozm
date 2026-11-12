import { describe, expect, it } from 'vitest'
import { groupCoatingAllowed, groupPatchIsEmpty, groupRowPatch } from './groupAttrs'

const group = (over: Partial<{ fireLimit: string; bearingType: string; coatingType: string }> = {}) => ({
  fireLimit: '',
  bearingType: '',
  coatingType: '',
  ...over
})

describe('атрибуты группы в форме распознавания', () => {
  it('пустая группа не трогает позиции', () => {
    const patch = groupRowPatch(group(), { fireLimit: 'R90', bearingType: 'несущая', coatingType: 'ТЕХНО ОЗМ' })
    expect(groupPatchIsEmpty(patch)).toBe(true)
  })

  it('предел ОС и тип конструкции назначаются всем позициям', () => {
    const patch = groupRowPatch(group({ fireLimit: 'R90', bearingType: 'несущая' }), {})
    expect(patch).toEqual({ fireLimit: 'R90', bearingType: 'несущая' })
  })

  it('тип покрытия группы назначается позициям', () => {
    const patch = groupRowPatch(group({ coatingType: 'TAIKOR FP Epoxy' }), { fireLimit: 'R60' })
    expect(patch.coatingType).toBe('TAIKOR FP Epoxy')
  })

  it('покрытие, недоступное при пределе ОС группы, не назначается', () => {
    // TAIKOR FP Extra бывает только при R90/R120
    const patch = groupRowPatch(group({ fireLimit: 'R45', coatingType: 'TAIKOR FP Extra + TAIKOR FP Graphite' }), {})
    expect(patch.coatingType).toBeUndefined()
    expect(patch.fireLimit).toBe('R45')
  })

  it('покрытие, недоступное при пределе ОС позиции, не назначается', () => {
    const patch = groupRowPatch(group({ coatingType: 'TAIKOR FP Extra + TAIKOR FP Graphite' }), { fireLimit: 'R30' })
    expect(patch.coatingType).toBeUndefined()
  })

  it('покрытие позиции снимается, если предел группы ему не соответствует', () => {
    const patch = groupRowPatch(group({ fireLimit: 'R15' }), { fireLimit: 'R90', coatingType: 'TAIKOR FP Extra + TAIKOR FP Graphite' })
    expect(patch.fireLimit).toBe('R15')
    expect(patch.coatingType).toBe('')
  })

  it('при совместимом покрытии позиция сохраняет своё', () => {
    const patch = groupRowPatch(group({ fireLimit: 'R90' }), { fireLimit: 'R45', coatingType: 'TAIKOR FP Epoxy' })
    expect(patch.coatingType).toBeUndefined()
    expect(patch.fireLimit).toBe('R90')
  })

  it('доступность покрытия группы проверяется по её пределу', () => {
    expect(groupCoatingAllowed(group({ fireLimit: 'R90', coatingType: 'TAIKOR FP Extra + TAIKOR FP Graphite' }))).toBe(true)
    expect(groupCoatingAllowed(group({ fireLimit: 'R30', coatingType: 'TAIKOR FP Extra + TAIKOR FP Graphite' }))).toBe(false)
    expect(groupCoatingAllowed(group({ coatingType: '' }))).toBe(true)
  })
})
