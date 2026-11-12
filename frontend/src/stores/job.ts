import { defineStore } from 'pinia'
import { coatingFitsRow } from '../utils/coatingTypes'
import { buildCheckRequest, describeIssue, issuesFromResponse } from '../utils/coatingCheck'
import { fetchDicts, postCalcCheck } from '../utils/api'
import { computed, ref } from 'vue'
import {
  confirmExtJob,
  copyExtRow,
  createExtRow,
  deleteExtRow,
  DEMO_JOB_ID,
  getExtJob,
  getExtPrefill,
  isLocalJob,
  patchExtRow,
  resetDemoJob
} from '../utils/extApi'
import type { Job, OgzRow, OgzRowPatch, PrefillPayload } from '../types/ogz'

function isEmptyJob(job: Job): boolean {
  return job.profileRows.length === 0 && job.sheetRows.length === 0
}

export const useJobStore = defineStore('job', () => {
  const job = ref<Job | null>(null)
  const loading = ref(false)
  const error = ref('')
  const lastPrefill = ref<PrefillPayload | null>(null)
  /** Замечания импорта из локального OCR — показываются один раз в форме. */
  const importNotes = ref<string[]>([])

  const profileRows = computed(() => job.value?.profileRows ?? [])
  const sheetRows = computed(() => job.value?.sheetRows ?? [])
  const isConfirmed = computed(() => job.value?.confirmed ?? false)

  const needsAttention = (r: OgzRow) =>
    r.status === 'Требует проверки' || r.status === 'Нужен ввод массы' || r.status === 'Нет данных'

  const needsAttentionCount = computed(
    () => [...profileRows.value, ...sheetRows.value].filter(needsAttention).length
  )

  /**
   * Номера строк (как в колонке «№» таблиц — порядковый номер в перечне
   * профильного погонажа и листовой стали), которые требуют проверки или ввода
   * массы. Отдельно по вкладкам: предупреждение называет вкладку и номера.
   */
  const attentionRows = computed(() => ({
    profile: profileRows.value.map((r, i) => (needsAttention(r) ? i + 1 : 0)).filter(Boolean),
    sheet: sheetRows.value.map((r, i) => (needsAttention(r) ? i + 1 : 0)).filter(Boolean)
  }))

  /**
   * Номера строк, где тип покрытия не подходит к пределу огнестойкости
   * (например, TAIKOR FP Graphite при R180): такие строки уйдут в калькулятор
   * с покрытием, которого при этом пределе не бывает.
   */
  const coatingMismatchRows = computed(() =>
    profileRows.value.map((r, i) => (coatingFitsRow(r) ? 0 : i + 1)).filter(Boolean)
  )

  const coatingMismatchCount = computed(() => coatingMismatchRows.value.length)

  /** Номера строк, по которым движок не подтвердил сочетание покрытия и R. */
  const engineIssueNumbers = computed(() =>
    profileRows.value.map((r, i) => (engineIssues.value[r.id] ? i + 1 : 0)).filter(Boolean)
  )
  const engineIssueCount = computed(() => engineIssueNumbers.value.length)
  /** Первое сообщение движка — для подсказки в баннере. */
  const engineIssueSample = computed(() => {
    for (const r of profileRows.value) {
      const m = engineIssues.value[r.id]
      if (m) return m
    }
    return ''
  })

  /**
   * Вердикты движка по сочетаниям «покрытие + предел ОС + тип конструкции»:
   * те же правила, что в калькуляторе. Ключ — id строки, значение — сообщение
   * («недостижимый предел огнестойкости R180 …»).
   */
  const engineIssues = ref<Record<number, string>>({})
  const engineChecking = ref(false)
  let checkTimer: ReturnType<typeof setTimeout> | null = null

  /** Пересчитать вердикты по текущим строкам (с задержкой, чтобы не дёргать на каждый символ). */
  function scheduleCoatingCheck(roll?: unknown[]): void {
    if (checkTimer) clearTimeout(checkTimer)
    checkTimer = setTimeout(async () => {
      const rows = profileRows.value.filter((r) => r.coatingType.trim() && r.fireLimit.trim())
      if (!rows.length) {
        engineIssues.value = {}
        return
      }
      let dict = roll as never[] | undefined
      if (!dict || !dict.length) {
        try {
          dict = (await fetchDicts()).roll as never[]
        } catch {
          engineIssues.value = {}
          return
        }
      }
      const { req, ids } = buildCheckRequest(rows, dict as never)
      if (!ids.length) {
        engineIssues.value = {}
        return
      }
      engineChecking.value = true
      try {
        const elements = await postCalcCheck(req)
        const issues = issuesFromResponse(ids, elements)
        const next: Record<number, string> = {}
        for (const i of issues) {
          const row = profileRows.value.find((r) => r.id === i.rowId)
          next[i.rowId] = describeIssue(row || {}, i.message)
        }
        engineIssues.value = next
      } catch {
        // движок недоступен — не показываем ложных предупреждений
        engineIssues.value = {}
      } finally {
        engineChecking.value = false
      }
    }, 700)
  }

  const missingFireLimitCount = computed(() =>
    profileRows.value.filter(
      (r) =>
        (r.status === 'Посчитано' || r.status === 'Требует проверки') &&
        !r.fireLimit.trim()
    ).length
  )

  const prefillableCount = computed(() =>
    profileRows.value.filter(
      (r) =>
        (r.status === 'Посчитано' || r.status === 'Требует проверки') &&
        r.lengthM > 0 &&
        r.fireLimit.trim() !== ''
    ).length
  )

  async function load(id: string) {
    loading.value = true
    error.value = ''
    try {
      job.value = await getExtJob(id)
      // Пустое серверное задание подменяем демо; задание из локального OCR
      // оставляем как есть — его пустота значима для пользователя.
      if (job.value && isEmptyJob(job.value) && !isLocalJob(id)) {
        await fillWithDemo()
      }
    } catch (e: any) {
      error.value = e?.message || 'Не удалось загрузить задание'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fillWithDemo() {
    resetDemoJob()
    job.value = await getExtJob(DEMO_JOB_ID)
    error.value = ''
  }

  function replaceRow(row: OgzRow) {
    if (!job.value) return
    const list = row.isSheet ? job.value.sheetRows : job.value.profileRows
    const idx = list.findIndex((r) => r.id === row.id)
    if (idx !== -1) list.splice(idx, 1, row)
  }

  function insertRowAfter(sourceId: number, row: OgzRow) {
    if (!job.value) return
    const list = row.isSheet ? job.value.sheetRows : job.value.profileRows
    // Avoid double-insert when demo API already mutated the shared job and we reloaded.
    if (list.some((r) => r.id === row.id)) return
    const idx = list.findIndex((r) => r.id === sourceId)
    if (idx === -1) list.push(row)
    else list.splice(idx + 1, 0, row)
  }

  function dropRow(rowId: number) {
    if (!job.value) return
    const drop = (rows: OgzRow[]) => {
      const i = rows.findIndex((r) => r.id === rowId)
      if (i !== -1) rows.splice(i, 1)
    }
    drop(job.value.profileRows)
    drop(job.value.sheetRows)
  }

  async function patch(rowId: number, body: OgzRowPatch) {
    if (!job.value) return
    const updated = await patchExtRow(job.value.id, rowId, body)
    replaceRow(updated)
    job.value.updatedAt = new Date().toISOString()
  }

  /** Alias used by ported OGZ form. */
  async function patchRow(rowId: number, body: OgzRowPatch) {
    await patch(rowId, body)
  }

  async function addRow(): Promise<OgzRow | null> {
    if (!job.value) return null
    const created = await createExtRow(job.value.id)
    if (!job.value.profileRows.some((r) => r.id === created.id)) {
      job.value.profileRows.push(created)
    }
    job.value.updatedAt = new Date().toISOString()
    return created
  }

  async function copyRow(rowId: number) {
    if (!job.value) return
    const created = await copyExtRow(job.value.id, rowId)
    insertRowAfter(rowId, created)
    job.value.updatedAt = new Date().toISOString()
  }

  async function removeRow(rowId: number) {
    if (!job.value) return
    await deleteExtRow(job.value.id, rowId)
    dropRow(rowId)
    job.value.updatedAt = new Date().toISOString()
  }

  async function confirm() {
    if (!job.value) return
    await confirmExtJob(job.value.id)
    job.value.confirmed = true
    job.value.confirmedAt = new Date().toISOString()
  }

  async function fetchPrefill(): Promise<PrefillPayload> {
    if (!job.value) throw new Error('no job')
    const payload = await getExtPrefill(job.value.id)
    lastPrefill.value = payload
    return payload
  }

  function openDemo() {
    resetDemoJob()
  }

  function setImportNotes(notes: string[]) {
    importNotes.value = [...notes]
  }

  function takeImportNotes(): string[] {
    const n = importNotes.value
    importNotes.value = []
    return n
  }

  return {
    job,
    loading,
    error,
    lastPrefill,
    importNotes,
    profileRows,
    sheetRows,
    isConfirmed,
    needsAttentionCount,
    attentionRows,
    coatingMismatchRows,
    coatingMismatchCount,
    engineIssues,
    engineChecking,
    engineIssueNumbers,
    engineIssueCount,
    engineIssueSample,
    scheduleCoatingCheck,
    missingFireLimitCount,
    prefillableCount,
    load,
    fillWithDemo,
    patch,
    patchRow,
    addRow,
    copyRow,
    removeRow,
    confirm,
    fetchPrefill,
    openDemo,
    setImportNotes,
    takeImportNotes
  }
})
