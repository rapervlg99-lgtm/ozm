/**
 * Атрибуты группы в форме распознавания: предел ОС, тип конструкции и тип
 * защитного покрытия. Группа назначает их всем своим позициям — и они уходят
 * в предзаполнение калькулятора вместе со строками.
 *
 * Логика вынесена из компонента в чистые функции: её проверяют тестами без
 * монтирования таблицы.
 */
import { isCoatingAvailableForFireLimit } from "./coatingTypes";

export interface GroupAttrs {
  fireLimit: string;
  bearingType: string;
  /** канонический тип покрытия (ТЕХНО ОЗМ / TAIKOR … / АКЗ); '' — как у позиций */
  coatingType: string;
}

export interface RowAttrs {
  fireLimit?: string;
  bearingType?: string;
  coatingType?: string;
}

export interface GroupRowPatch {
  fireLimit?: string;
  bearingType?: string;
  coatingType?: string;
}

/**
 * Патч позиции по атрибутам группы. Пустые поля группы строку не трогают.
 *
 * Покрытие ставится только если оно доступно при пределе огнестойкости
 * (сначала смотрим предел группы, иначе предел самой строки) — иначе у позиции
 * оказалось бы покрытие, которого при таком R не бывает. Если предел группы
 * задан, а покрытие позиции ему не соответствует, покрытие снимается.
 */
export function groupRowPatch(group: GroupAttrs, row: RowAttrs): GroupRowPatch {
  const patch: GroupRowPatch = {};
  const fireLimit = (group.fireLimit || "").trim();
  const bearingType = (group.bearingType || "").trim();
  const coating = (group.coatingType || "").trim();

  if (fireLimit) patch.fireLimit = fireLimit;
  if (bearingType) patch.bearingType = bearingType;

  const effectiveLimit = fireLimit || row.fireLimit || "";
  if (coating && isCoatingAvailableForFireLimit(coating, effectiveLimit)) {
    patch.coatingType = coating;
  }

  const resultCoating = patch.coatingType ?? row.coatingType ?? "";
  if (fireLimit && resultCoating && !isCoatingAvailableForFireLimit(resultCoating, fireLimit)) {
    patch.coatingType = "";
  }
  return patch;
}

/** Нужен ли патч вообще: пустая группа ничего не назначает. */
export function groupPatchIsEmpty(patch: GroupRowPatch): boolean {
  return (
    patch.fireLimit === undefined &&
    patch.bearingType === undefined &&
    patch.coatingType === undefined
  );
}

/** Доступно ли покрытие группы при её пределе ОС (для подсказки в интерфейсе). */
export function groupCoatingAllowed(group: GroupAttrs): boolean {
  const coating = (group.coatingType || "").trim();
  if (!coating) return true;
  return isCoatingAvailableForFireLimit(coating, group.fireLimit);
}
