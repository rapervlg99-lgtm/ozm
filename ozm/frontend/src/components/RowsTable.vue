<template>
  <div class="rows">
    <div v-if="rows.length === 0" class="rows__empty">
      {{ emptyText }}
    </div>

    <div v-else class="rows__scroll">
      <table class="rows__table">
        <thead>
          <tr>
            <th class="rows__idx">№</th>
            <th>Наименование</th>
            <th>{{ kind === "sheet" ? "Номер профиля" : "Профиль" }}</th>
            <th class="rows__num">{{ kind === "sheet" ? "Масса 1 м²" : "Масса 1 м" }}</th>
            <th class="rows__num">{{ kind === "sheet" ? "Площадь, м²" : "Длина, м" }}</th>
            <th
              title="Уверенность ИИ в распознанном содержимом и готовность строки"
            >
              Статус
            </th>
            <th>Обогрев</th>
            <th>Предел ОС</th>
            <th>Тип конструкции</th>
            <th>Покрытие</th>
            <th aria-label="Действия"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(row, i) in rows"
            :key="row.id"
            :class="{ 'rows__row_attention': needsAttention(row) }"
          >
            <td class="rows__idx">{{ i + 1 }}</td>
            <td>{{ row.name || "—" }}</td>
            <td>{{ row.profileMark || row.profileRaw || "—" }}</td>
            <td class="rows__num">{{ row.massPerMeter ? num(row.massPerMeter) : "—" }}</td>
            <td class="rows__num">
              {{ kind === "sheet" ? cell(row.areaM2) : cell(row.lengthM) }}
            </td>
            <td><StatusTag :status="row.status" /></td>
            <td>
              <HeatingSidesPicker
                compact
                :model-value="row.heatingSides.trim() || '4'"
                :aria-label="`Обогрев ${row.name || row.profileMark || 'строка'}`"
                @update:model-value="onHeatingChange(row, $event)"
              />
            </td>
            <td>
              <select
                class="rows__select"
                :value="row.fireLimit"
                :aria-label="`Предел ОС ${row.name || row.profileMark || 'строка'}`"
                @change="onFireLimitChange(row, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">—</option>
                <option
                  v-for="r in fireLimitChoices(row.fireLimit)"
                  :key="r"
                  :value="r"
                >
                  {{ r }}
                </option>
              </select>
            </td>
            <td>
              <select
                class="rows__select"
                :value="row.bearingType"
                :aria-label="`Тип конструкции ${row.name || row.profileMark || 'строка'}`"
                @change="onBearingTypeChange(row, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">—</option>
                <option
                  v-for="t in bearingTypeChoices(row.bearingType)"
                  :key="t"
                  :value="t"
                >
                  {{ t }}
                </option>
              </select>
            </td>
            <td>
              <select
                class="rows__select rows__coating-select"
                :value="row.coatingType"
                :aria-label="`Покрытие ${row.name || row.profileMark || 'строка'}`"
                @change="onCoatingTypeChange(row, ($event.target as HTMLSelectElement).value)"
              >
                <option value="">—</option>
                <option
                  v-for="t in coatingTypeChoices(row.coatingType, row.fireLimit)"
                  :key="t"
                  :value="t"
                >
                  {{ t }}
                </option>
              </select>
            </td>
            <td class="rows__acts">
              <button type="button" class="rows__act" title="Редактировать" aria-label="Редактировать строку" @click="$emit('edit', row)"><TnIcon name="edit" :size="20" /></button>
              <button type="button" class="rows__act" title="Копировать" aria-label="Копировать строку" @click="$emit('copy', row)"><TnIcon name="copy" :size="20" /></button>
              <button type="button" class="rows__act rows__del" title="Удалить" aria-label="Удалить строку" @click="$emit('remove', row)"><TnIcon name="delete" :size="20" /></button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import HeatingSidesPicker from "./HeatingSidesPicker.vue";
import StatusTag from "./StatusTag.vue";
import TnIcon from "./ui/TnIcon.vue";
import { bearingTypeChoices } from "@/utils/bearingTypes";
import { coatingTypeChoices } from "@/utils/coatingTypes";
import { fireLimitChoices } from "@/utils/fireLimits";
import { num } from "@/utils/format";
import type { OgzRow, OgzRowPatch } from "@/types/ogz";

defineProps<{ kind: "profile" | "sheet"; rows: OgzRow[]; emptyText: string }>();
const emit = defineEmits<{
  (e: "edit", row: OgzRow): void;
  (e: "copy", row: OgzRow): void;
  (e: "remove", row: OgzRow): void;
  (e: "patch", rowId: number, patch: OgzRowPatch): void;
}>();

function cell(v: number): string {
  return v ? num(v) : "—";
}

function onHeatingChange(row: OgzRow, value: string): void {
  if (value === row.heatingSides) return;
  emit("patch", row.id, { heatingSides: value });
}

function onFireLimitChange(row: OgzRow, value: string): void {
  if (value === row.fireLimit) return;
  emit("patch", row.id, { fireLimit: value });
}

function onBearingTypeChange(row: OgzRow, value: string): void {
  if (value === row.bearingType) return;
  emit("patch", row.id, { bearingType: value });
}

function onCoatingTypeChange(row: OgzRow, value: string): void {
  if (value === row.coatingType) return;
  emit("patch", row.id, { coatingType: value });
}

function needsAttention(row: OgzRow): boolean {
  return (
    row.status === "Требует проверки" ||
    row.status === "Нужен ввод массы" ||
    row.status === "Нет данных"
  );
}
</script>

<style scoped>
.rows__empty {
  padding: 24px;
  color: var(--content-secondary-enabled);
  text-align: center;
}

.rows__scroll {
  overflow-x: auto;
}

.rows__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-body-small);
}

.rows__table th,
.rows__table td {
  /* после добавления колонки действий таблица не должна уезжать за контейнер:
     отступы 8 px вместо 12 px дают запас по ширине */
  padding: 8px 8px;
  text-align: left;
  white-space: nowrap;
  border-bottom: 1px solid var(--border-secondary-enabled);
}

.rows__table th {
  font-weight: var(--font-weight-accent);
  color: var(--content-secondary-enabled);
}

.rows__num {
  text-align: right;
}

.rows__idx {
  text-align: right;
  color: var(--content-secondary-enabled);
}

.rows__select {
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--border-secondary-enabled);
  border-radius: 8px;
  background: var(--background-primary-a-enabled);
  font: inherit;
  cursor: pointer;
}

.rows__acts {
  display: flex;
  align-items: center;
  gap: 4px;
}

.rows__act {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--content-secondary-enabled);
  cursor: pointer;
}

.rows__act:hover {
  background: var(--background-secondary-a-enabled);
  color: var(--content-primary-a-enabled);
}

.rows__del:hover {
  color: var(--content-system-negative);
}

.rows__row_attention {
  background: var(--orange-10);
}

.rows__select {
  height: 32px;
  width: 140px;
  padding: 0 8px;
  border: 1px solid var(--border-secondary-enabled);
  border-radius: 8px;
  background: var(--background-primary-a-enabled);
  font: inherit;
  cursor: pointer;
}

.rows__coating-select {
  width: 220px;
  max-width: 28vw;
}
</style>
