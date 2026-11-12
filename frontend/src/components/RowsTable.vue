<template>
  <div class="rows">
    <div v-if="rows.length === 0" class="rows__empty">
      {{ emptyText }}
    </div>

    <template v-else>
      <!-- Панель групп: у листовой стали те же группы, что у профильного
           погонажа, — название, предел ОС, тип конструкции и покрытие
           назначаются всем позициям группы. -->
      <div class="rows__groups">
        <div class="rows__groups-head">
          <span class="rows__groups-title">Группа элементов</span>
          <button type="button" class="btn-ghost btn-sm" @click="addGroup">Создать группу</button>
        </div>
        <p class="rows__groups-hint">
          Перетащи строку в окно группы — позиция уйдёт из общего перечня; предел
          ОС, тип конструкции и покрытие назначатся всем её позициям.
        </p>
      </div>

      <div
        v-if="selectedIds.size > 0"
        class="rows__bulk"
      >
        <span>Выбрано: {{ selectedIds.size }}</span>
        <select v-model="bulkGroupId" class="rows__select" aria-label="Группа для переноса">
          <option value="">— группа —</option>
          <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }}</option>
        </select>
        <button type="button" class="btn-primary btn-sm" :disabled="!bulkGroupId" @click="moveSelectedToGroup">Затянуть</button>
        <button type="button" class="btn-ghost btn-sm" @click="copySelected">Копировать</button>
        <button type="button" class="btn-ghost btn-sm" @click="removeSelected">Удалить</button>
      </div>

      <div
        v-for="sec in sections"
        :key="sec.key"
        class="rows__section"
        :class="{ 'rows__section_over': dragOverId === sec.group?.id }"
        @dragover.prevent="sec.group && onGroupDragOver(sec.group.id)"
        @dragleave="sec.group && onGroupDragLeave(sec.group.id)"
        @drop.prevent="sec.group && onDropToGroup(sec.group.id)"
      >
        <div v-if="sec.group" class="rows__group-head">
          <input v-model="sec.group.name" class="rows__group-name" aria-label="Название группы" />
          <select
            v-model="sec.group.fireLimit"
            class="rows__select"
            aria-label="Предел ОС группы"
            @change="onGroupFireLimit(sec.group.id)"
          >
            <option value="">R…</option>
            <option v-for="r in fireLimitChoices(sec.group.fireLimit)" :key="r" :value="r">{{ r }}</option>
          </select>
          <select
            v-model="sec.group.bearingType"
            class="rows__select"
            aria-label="Тип конструкции группы"
            @change="applyGroupAttrs(sec.group.id)"
          >
            <option value="">Тип…</option>
            <option v-for="t in bearingTypeChoices(sec.group.bearingType)" :key="t" :value="t">{{ t }}</option>
          </select>
          <select
            v-model="sec.group.coatingType"
            class="rows__select rows__coating-select"
            aria-label="Тип покрытия группы"
            @change="applyGroupAttrs(sec.group.id)"
          >
            <option value="">Покрытие…</option>
            <option v-for="c in coatingTypeChoices(sec.group.coatingType, sec.group.fireLimit)" :key="c" :value="c">{{ c }}</option>
          </select>
          <span class="rows__group-count">{{ sec.rows.length }} эл.</span>
          <button type="button" class="btn-ghost btn-sm" aria-label="Удалить группу" @click="removeGroup(sec.group.id)">Удал.</button>
        </div>

        <div v-if="sec.group && sec.rows.length === 0" class="rows__group-empty">
          Перетащи сюда позиции из общего перечня
        </div>
        <div v-else class="rows__scroll">
          <table class="rows__table">
            <thead>
              <tr>
                <th class="rows__check" aria-label="Выбор"></th>
                <th class="rows__idx">№</th>
                <th>Наименование</th>
                <th>{{ kind === "sheet" ? "Номер листа" : "Номер профиля" }}</th>
                <th class="rows__num">{{ kind === "sheet" ? "Масса 1 м²" : "Масса 1 м" }}</th>
                <th class="rows__num">{{ kind === "sheet" ? "Площадь, м²" : "Длина, м" }}</th>
                <th title="Уверенность ИИ в распознанном содержимом и готовность строки">Статус</th>
                <th>Обогрев</th>
                <th>Предел ОС</th>
                <th>Тип конструкции</th>
                <th>Покрытие</th>
                <th aria-label="Действия"></th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in sec.rows"
                :key="row.id"
                :class="{ 'rows__row_attention': needsAttention(row) }"
                draggable="true"
                @dragstart="onRowDragStart(row.id, $event)"
              >
                <td class="rows__check">
                  <input
                    type="checkbox"
                    :checked="selectedIds.has(row.id)"
                    :aria-label="`Выбрать строку ${rowNo(row)}`"
                    @change="toggleSelect(row.id)"
                  />
                </td>
                <td class="rows__idx">{{ rowNo(row) }}</td>
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
                    :only="sideOptions"
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
                    <option v-for="r in fireLimitChoices(row.fireLimit)" :key="r" :value="r">{{ r }}</option>
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
                    <option v-for="t in bearingTypeChoices(row.bearingType)" :key="t" :value="t">{{ t }}</option>
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
                    <option v-for="t in coatingTypeChoices(row.coatingType, row.fireLimit)" :key="t" :value="t">{{ t }}</option>
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
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import HeatingSidesPicker from "./HeatingSidesPicker.vue";
import StatusTag from "./StatusTag.vue";
import TnIcon from "./ui/TnIcon.vue";
import { bearingTypeChoices } from "@/utils/bearingTypes";
import { coatingTypeChoices } from "@/utils/coatingTypes";
import { groupRowPatch } from "@/utils/groupAttrs";
import { fireLimitChoices } from "@/utils/fireLimits";
import { num } from "@/utils/format";
import type { OgzRow, OgzRowPatch } from "@/types/ogz";

const props = withDefaults(
  defineProps<{
    kind: "profile" | "sheet";
    rows: OgzRow[];
    emptyText: string;
  }>(),
  { kind: "sheet" }
);

const emit = defineEmits<{
  (e: "edit", row: OgzRow): void;
  (e: "copy", row: OgzRow): void;
  (e: "remove", row: OgzRow): void;
  (e: "copy-many", rowIds: number[]): void;
  (e: "remove-many", rowIds: number[]): void;
  (e: "patch", rowId: number, patch: OgzRowPatch): void;
  (e: "patch-many", rowIds: number[], patch: OgzRowPatch): void;
}>();

/** Лист обогревается только сверху и снизу; профиль — со всех сторон. */
const sideOptions = computed<("top" | "bottom" | "left" | "right")[]>(() =>
  props.kind === "sheet" ? ["top", "bottom"] : ["bottom", "top", "left", "right"]
);

interface RowsGroup {
  id: string;
  name: string;
  fireLimit: string;
  coatingType: string;
  bearingType: string;
  rowIds: number[];
}

const groups = ref<RowsGroup[]>([]);
const selectedIds = ref<Set<number>>(new Set());
const bulkGroupId = ref("");
const dragRowId = ref<number | null>(null);
const dragOverId = ref<string | null>(null);
let groupSeq = 1;

const rowNumbers = computed(() => new Map(props.rows.map((r, i) => [r.id, i + 1])));
function rowNo(row: OgzRow): number {
  return rowNumbers.value.get(row.id) ?? 0;
}

const groupedIds = computed(() => {
  const ids = new Set<number>();
  for (const g of groups.value) for (const id of g.rowIds) ids.add(id);
  return ids;
});

/** Секции таблицы: окна групп и общий перечень нераспределённых позиций. */
const sections = computed(() => {
  const byId = new Map(props.rows.map((r) => [r.id, r]));
  const out: { key: string; group?: RowsGroup; rows: OgzRow[] }[] = groups.value.map((g) => ({
    key: g.id,
    group: g,
    rows: g.rowIds.map((id) => byId.get(id)).filter((r): r is OgzRow => !!r)
  }));
  const ungrouped = props.rows.filter((r) => !groupedIds.value.has(r.id));
  if (ungrouped.length || groups.value.length === 0) out.push({ key: "ungrouped", rows: ungrouped });
  return out;
});

function cell(v: number): string {
  return v ? num(v) : "—";
}

function addGroup(): void {
  groups.value.push({
    id: `g-${groupSeq++}`,
    name: `Группа ${groups.value.length + 1}`,
    fireLimit: "",
    coatingType: "",
    bearingType: "",
    rowIds: []
  });
}

function removeGroup(id: string): void {
  groups.value = groups.value.filter((g) => g.id !== id);
  if (bulkGroupId.value === id) bulkGroupId.value = "";
}

function toggleSelect(id: number): void {
  const next = new Set(selectedIds.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  selectedIds.value = next;
}

function onRowDragStart(rowId: number, ev: DragEvent): void {
  dragRowId.value = rowId;
  ev.dataTransfer?.setData("text/plain", String(rowId));
  if (ev.dataTransfer) ev.dataTransfer.effectAllowed = "move";
}

function onGroupDragOver(groupId: string): void {
  dragOverId.value = groupId;
}

function onGroupDragLeave(groupId: string): void {
  if (dragOverId.value === groupId) dragOverId.value = null;
}

function onDropToGroup(groupId: string): void {
  dragOverId.value = null;
  const rowId = dragRowId.value;
  dragRowId.value = null;
  if (rowId == null) return;
  assignRowsToGroup(groupId, [rowId]);
}

function moveSelectedToGroup(): void {
  if (!bulkGroupId.value) return;
  assignRowsToGroup(bulkGroupId.value, [...selectedIds.value]);
  selectedIds.value = new Set();
  bulkGroupId.value = "";
}

function copySelected(): void {
  const ids = [...selectedIds.value];
  if (!ids.length) return;
  emit("copy-many", ids);
  selectedIds.value = new Set();
}

function removeSelected(): void {
  const ids = [...selectedIds.value];
  if (!ids.length) return;
  emit("remove-many", ids);
  selectedIds.value = new Set();
}

function assignRowsToGroup(groupId: string, rowIds: number[]): void {
  const target = groups.value.find((g) => g.id === groupId);
  if (!target) return;
  for (const g of groups.value) {
    if (g.id === groupId) continue;
    g.rowIds = g.rowIds.filter((id) => !rowIds.includes(id));
  }
  const set = new Set(target.rowIds);
  for (const id of rowIds) set.add(id);
  target.rowIds = [...set];
  applyGroupAttrs(groupId);
}

/** Общие значения группы (R, покрытие, тип конструкции) — на все позиции. */
function applyGroupAttrs(groupId: string): void {
  const g = groups.value.find((x) => x.id === groupId);
  if (!g || g.rowIds.length === 0) return;
  if (!g.fireLimit.trim() && !g.bearingType.trim() && !g.coatingType.trim()) return;
  const rowsById = new Map(props.rows.map((r) => [r.id, r]));
  for (const id of g.rowIds) {
    const row = rowsById.get(id);
    emit("patch", id, groupRowPatch(g, row ?? {}) as OgzRowPatch);
  }
}

function onGroupFireLimit(groupId: string): void {
  applyGroupAttrs(groupId);
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

/** Группы для предзаполнения калькулятора (как у профильного погонажа). */
function getGroupsForPrefill(): { title: string; rowIds: number[] }[] {
  return groups.value
    .map((g) => ({ title: g.name.trim() || `Группа ${g.id}`, rowIds: [...g.rowIds] }))
    .filter((g) => g.rowIds.length > 0);
}

defineExpose({ getGroupsForPrefill });
</script>

<style scoped>
.rows__empty {
  padding: 24px;
  color: var(--content-secondary-enabled);
  text-align: center;
}

.rows__groups {
  margin: 0 16px 8px;
  padding: 12px;
  border: 1px dashed var(--border-secondary-enabled);
  border-radius: 12px;
}

.rows__groups-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.rows__groups-title {
  font-weight: var(--font-weight-accent);
}

.rows__groups-hint {
  margin: 6px 0 0;
  color: var(--content-secondary-enabled);
  font-size: var(--font-size-body-small);
  line-height: var(--font-lineHeight-body-small);
}

.rows__bulk {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 16px 8px;
  font-size: var(--font-size-body-small);
}

.rows__section {
  margin-bottom: 8px;
  border-radius: 12px;
}

.rows__section_over {
  outline: 2px dashed var(--content-accent-enabled);
  outline-offset: 2px;
}

.rows__group-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin: 0 16px;
  padding: 10px 12px;
  background: var(--background-secondary-a-enabled);
  border-radius: 12px 12px 0 0;
}

.rows__group-name {
  flex: 1 1 160px;
  min-width: 120px;
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--border-secondary-enabled);
  border-radius: 8px;
  background: var(--background-primary-a-enabled);
  font: inherit;
  font-weight: var(--font-weight-accent);
}

.rows__group-count {
  color: var(--content-secondary-enabled);
  font-size: var(--font-size-body-small);
}

.rows__group-empty {
  margin: 0 16px;
  padding: 12px;
  border: 1px dashed var(--border-secondary-hover);
  border-radius: 12px;
  color: var(--content-secondary-enabled);
  font-size: var(--font-size-body-small);
}

.rows__scroll {
  overflow-x: auto;
  margin: 0 16px;
}

.rows__table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--font-size-body-small);
}

.rows__table th,
.rows__table td {
  /* после добавления колонки действий таблица не должна уезжать за контейнер */
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

.rows__check {
  width: 32px;
}

.rows__row_attention {
  background: var(--background-system-warning, rgba(255, 196, 0, 0.08));
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

.rows__coating-select {
  width: 208px;
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
</style>
