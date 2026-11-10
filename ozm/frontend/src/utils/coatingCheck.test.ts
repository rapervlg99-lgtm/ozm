import { describe, expect, it } from 'vitest'
import { bearingTypeCode, buildCheckRequest, describeIssue, elementForCheck, issuesFromResponse, sidesFromText } from './coatingCheck'
import type { OgzRow } from '../types/ogz'
import type { RollMark } from '../types/api'

const roll: RollMark[] = [
  { id: 'r79', label: '20Ш1', shape: 'I-beam_sm', standard: 'ГОСТ Р 57837-2017', massPerMeter: 30.6, dims: { h: 100, b: 55, s: 4.1, t: 5.7, R: 7 } },
  { id: 'r1', label: '40Б1', shape: 'I-beam', standard: 'ГОСТ 26020-83', massPerMeter: 56.6, dims: { h: 396, b: 199, s: 7, t: 11.2, R: 13 } }
] as RollMark[]

function row(over: Partial<OgzRow> = {}): OgzRow {
  return {
    id: 1, name: 'Двутавры', profileMark: '40Б1', profileRaw: '40Б1', gostProfile: 'ГОСТ 26020-83',
    steelGrade: 'C245', ppNumber: '1', construction: 'балка', mass: 1, massUnit: 't', massPerMeter: 56.6,
    massSource: 'cache', sourceUrl: '', isSheet: false, lengthM: 6, areaM2: 0, classification: '',
    status: 'Посчитано', heatingSides: '4 стороны', fireLimit: 'R90', bearingType: 'Несущая',
    coatingType: 'TAIKOR FP Graphite', ...over
  } as OgzRow
}

describe('проверка покрытия движком на этапе распознавания', () => {
  it('тип конструкции переводится в код движка', () => {
    expect(bearingTypeCode('Несущая')).toBe('1')
    expect(bearingTypeCode('самонесущая')).toBe('0')
    expect(bearingTypeCode('')).toBe('1')
  })

  it('стороны обогрева разбираются из текста строки', () => {
    expect(sidesFromText('4 стороны')).toEqual({ bottom: true, left: true, top: true, right: true })
    expect(sidesFromText('снизу, слева')).toEqual({ bottom: true, left: true, top: false, right: false })
    expect(sidesFromText('')).toEqual({ bottom: true, left: true, top: true, right: true })
  })

  it('элемент для движка собирается из строки: марка, покрытие, R, тип', () => {
    const el = elementForCheck(row(), roll)
    expect(el).not.toBeNull()
    expect(el!.rollId).toBe('r1')
    expect(el!.shape).toBe('I-beam')
    expect(el!.coat).toBe('3')
    expect(el!.htLevel).toBe(90)
    expect(el!.frType).toBe('1')
    expect(el!.lengthM).toBe(6)
  })

  it('в запрос попадают только строки с покрытием и пределом ОС', () => {
    const rows = [
      row({ id: 1 }),
      row({ id: 2, coatingType: '' }),
      row({ id: 3, fireLimit: '' }),
      row({ id: 4, profileMark: 'несуществующая марка' })
    ]
    const { req, ids } = buildCheckRequest(rows, roll)
    expect(ids).toEqual([1])
    expect(req.groups).toHaveLength(1)
    expect(req.consent).toBe(true)
  })

  it('сообщения движка раскладываются по строкам', () => {
    const issues = issuesFromResponse([1, 2], [
      { id: '1', exclusion: 'недостижимый предел огнестойкости R180 при δпр=1,9' },
      { id: '2', exclusion: '' }
    ])
    expect(issues).toEqual([{ rowId: 1, message: 'недостижимый предел огнестойкости R180 при δпр=1,9' }])
  })

  it('внутренняя формулировка движка превращается в понятную', () => {
    const text = describeIssue(
      { coatingType: 'TAIKOR FP Graphite', fireLimit: 'R180' },
      'нет данных x500 для coat=3 R180 δпр=4.7'
    )
    expect(text).toBe(
      'TAIKOR FP Graphite: при R180 и δпр=4,7 мм решения нет — выберите другое покрытие или предел ОС'
    )
    // читаемые сообщения движка не трогаем
    const plain = 'недостижимый предел огнестойкости R180 при δпр=1,9 (Rфакт≈150)'
    expect(describeIssue({ coatingType: 'ТЕХНО ОЗМ', fireLimit: 'R180' }, plain)).toBe(plain)
  })
})
