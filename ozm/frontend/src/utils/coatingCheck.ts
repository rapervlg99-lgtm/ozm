/**
 * Проверка сочетания «профиль + покрытие + предел ОС + тип конструкции» на
 * этапе распознавания — тем же движком, что считает калькулятор.
 *
 * Правила пригодности покрытия живут в движке (таблицы T500 / X500): для
 * профиля с приведённой толщиной δпр и требуемого R движок знает, даёт ли
 * покрытие такую стойкость. Здесь мы собираем для каждой строки такой же
 * элемент, как калькулятор, и спрашиваем бэкенд — поэтому ответ совпадает с
 * тем, что пользователь увидит в результатах расчёта.
 */
import type { CalcRequest, ElementInput, RollMark } from "../types/api";
import type { OgzRow } from "../types/ogz";
import { mapCoatingType, markAliases } from "./prefillMapper";
import { parseFireLimitMinutes } from "./coatingTypes";
import { profileCategory, rollMarkCandidates } from "./ocrImport";

export interface CoatingIssue {
  rowId: number;
  message: string;
}

/** Код типа конструкции в движке: несущая — '1', самонесущая — '0'. */
export function bearingTypeCode(bearingType: string): string {
  const b = (bearingType || "").toLowerCase();
  if (b.startsWith("самонес")) return "0";
  if (b.startsWith("нес")) return "1";
  return "1";
}

/** Стороны обогрева из текста строки («4 стороны», «снизу, слева»). */
export function sidesFromText(text: string): {
  bottom: boolean;
  left: boolean;
  top: boolean;
  right: boolean;
} {
  const t = (text || "").toLowerCase();
  if (!t.trim()) return { bottom: true, left: true, top: true, right: true };
  if (/4\s*сторон|все\s*сторон|четыре/.test(t)) {
    return { bottom: true, left: true, top: true, right: true };
  }
  return {
    bottom: /снизу|низ/.test(t),
    left: /слева|лев/.test(t),
    top: /сверху|верх/.test(t),
    right: /справа|прав/.test(t)
  };
}

/** Марка проката для строки: та же, что подбирает калькулятор по обозначению. */
export function rollForRow(row: OgzRow, roll: RollMark[]): RollMark | null {
  const mark = (row.profileMark || "").trim();
  if (!mark) return null;
  const category = profileCategory(row.name || "", row.gostProfile || "", row.profileRaw || mark);
  const aliases = new Set(markAliases(mark));
  for (const candidate of rollMarkCandidates(category, mark)) {
    const hit = roll.find((r) => markAliases(r.label).includes(candidate));
    if (hit) return hit;
  }
  for (const r of roll) {
    if (markAliases(r.label).some((a) => aliases.has(a))) return r;
  }
  return null;
}

/**
 * Элемент калькулятора для строки распознанной формы — ровно те поля, которые
 * влияют на пригодность покрытия. null, если профиль не нашёлся в сортаменте
 * (тогда проверить нечем).
 */
export function elementForCheck(row: OgzRow, roll: RollMark[]): ElementInput | null {
  const matched = rollForRow(row, roll);
  if (!matched) return null;
  const ht = parseFireLimitMinutes(row.fireLimit);
  return {
    id: String(row.id),
    title: row.profileMark || row.name || "",
    shape: matched.shape,
    rollId: matched.id,
    dims: matched.dims ? { ...matched.dims } : {},
    frType: bearingTypeCode(row.bearingType),
    htLevel: ht ?? 0,
    sides: sidesFromText(row.heatingSides),
    lengthM: row.lengthM > 0 ? row.lengthM : 1,
    quantity: 1,
    coat: mapCoatingType(row.coatingType),
    method: "1"
  } as ElementInput;
}

/** Запрос проверки: по одному элементу в группе, порядок сохраняется. */
export function buildCheckRequest(rows: OgzRow[], roll: RollMark[]): { req: CalcRequest; ids: number[] } {
  const ids: number[] = [];
  const elements: ElementInput[] = [];
  for (const row of rows) {
    if (!row.coatingType.trim() || !row.fireLimit.trim()) continue;
    const el = elementForCheck(row, roll);
    if (!el) continue;
    ids.push(row.id);
    elements.push(el);
  }
  const req: CalcRequest = {
    objectName: "проверка покрытия",
    address: "—",
    consent: true,
    frDurability: "1",
    groups: elements.map((el) => ({ quantity: 1, elements: [el] }))
  };
  return { req, ids };
}


/**
 * Человеческий текст вердикта движка. Внутренние формулировки («нет данных
 * x500 для coat=3 R180 δпр=4.7») заменяем понятной фразой, остальные сообщения
 * движка уже читаемые — их оставляем как есть.
 */
export function describeIssue(row: { coatingType?: string; fireLimit?: string }, message: string): string {
  const m = /нет данных x500.*?R(\d+).*?δпр=([\d.,]+)/.exec(message);
  const coating = (row.coatingType || "").trim() || "покрытие";
  const r = (row.fireLimit || "").trim();
  if (m) {
    const dpr = m[2].replace(".", ",");
    return `${coating}: при ${r || "R" + m[1]} и δпр=${dpr} мм решения нет — выберите другое покрытие или предел ОС`;
  }
  return message;
}

/** Собирает проблемы из ответа движка. */
export function issuesFromResponse(ids: number[], elements: { id: string; exclusion?: string }[]): CoatingIssue[] {
  const out: CoatingIssue[] = [];
  elements.forEach((el, i) => {
    const rowId = ids[i];
    if (rowId == null) return;
    const message = (el.exclusion || "").trim();
    if (message) out.push({ rowId, message });
  });
  return out;
}
