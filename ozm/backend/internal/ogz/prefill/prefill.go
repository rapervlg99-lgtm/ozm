// Package prefill формирует контракт предзаполнения онлайн-калькулятора
// огнезащиты (nav.tn.ru/calculators/fireproof/) из подтверждённой формы ОГЗ
// (ТЗ §9). Контракт проектируем мы; приём реализует команда калькулятора.
// Выгрузки в файлы нет — это структура payload/deeplink.
package prefill

import (
	"math"
	"strconv"
	"strings"

	"ozm/backend/internal/ogz/classify"
)

// Version — версия контракта предзаполнения (для совместимости на стороне калькулятора).
const Version = "1.0"

// RowInput — строка подтверждённой формы (из storage.ogz_rows), нужная для payload.
type RowInput struct {
	ProfileMark  string
	Construction string
	IsSheet      bool
	LengthM      float64
	AreaM2       float64
	Status       string
	HeatingSides string
	FireLimit    string
	BearingType  string
	CoatingType  string
	// Масса 1 м² для листовой стали (кг/м²) — по ней определяется толщина,
	// если её нет в обозначении.
	MassPerM2 float64
}

// Item — позиция в калькуляторе: профильный погонаж или листовая сталь.
// Для листа задаём вид сечения и размеры прямо в payload: марки в сортаменте
// у листа нет, и калькулятор не смог бы подобрать сечение сам.
type Item struct {
	ProfileMark  string  `json:"profileMark"`
	Shape        string  `json:"shape,omitempty"`
	Dims         map[string]float64 `json:"dims,omitempty"`
	AreaM2       float64 `json:"areaM2,omitempty"`
	Construction string  `json:"construction,omitempty"`
	LengthM      float64 `json:"lengthM"`
	HeatingSides string  `json:"heatingSides,omitempty"`
	FireLimit    string  `json:"fireLimit,omitempty"`
	BearingType  string  `json:"bearingType,omitempty"`
	CoatingType  string  `json:"coatingType,omitempty"`
}

// Sheet — суммарная листовая сталь.
type Sheet struct {
	AreaM2 float64 `json:"areaM2"`
}

// Payload — контракт предзаполнения калькулятора огнезащиты.
type Payload struct {
	Version string `json:"version"`
	JobID   string `json:"jobId"`
	Items   []Item `json:"items"`
	Sheet   Sheet  `json:"sheet"`
}

// Build собирает payload из подтверждённых строк. В калькулятор попадают только
// рассчитанные строки (статус «Посчитано»/«Требует проверки») с назначенным
// пределом ОС (R); «Не считается» и «Нужен ввод массы» пропускаются.
// Листовая сталь суммируется по площади.
func Build(jobID string, rows []RowInput) Payload {
	p := Payload{Version: Version, JobID: jobID}
	for _, r := range rows {
		if !included(r.Status) {
			continue
		}
		if r.IsSheet {
			// Площадь листа оставляем в сводке справочно, а сам лист уходит в
			// калькулятор конструкцией: δпр = t (обогрев с одной стороны) или
			// t/2 (сверху и снизу), площадь — обогреваемая поверхность.
			p.Sheet.AreaM2 += r.AreaM2
			if r.AreaM2 <= 0 || strings.TrimSpace(r.FireLimit) == "" {
				continue
			}
			thickness, width := sheetSize(r.ProfileMark, r.MassPerM2)
			p.Items = append(p.Items, Item{
				ProfileMark:  r.ProfileMark,
				Shape:        "plate",
				Dims:         map[string]float64{"t": thickness, "b": width},
				AreaM2:       r.AreaM2,
				Construction: r.Construction,
				LengthM:      r.AreaM2 / (width / 1000),
				HeatingSides: r.HeatingSides,
				FireLimit:    r.FireLimit,
				BearingType:  r.BearingType,
				CoatingType:  r.CoatingType,
			})
			continue
		}
		if r.LengthM <= 0 {
			continue
		}
		// Без R элемент не уходит из таблицы в калькулятор.
		if strings.TrimSpace(r.FireLimit) == "" {
			continue
		}
		p.Items = append(p.Items, Item{
			ProfileMark:  r.ProfileMark,
			Construction: r.Construction,
			LengthM:      r.LengthM,
			HeatingSides: r.HeatingSides,
			FireLimit:    r.FireLimit,
			BearingType:  r.BearingType,
			CoatingType:  r.CoatingType,
		})
	}
	return p
}

func included(status string) bool {
	return status == classify.StatusCalculated || status == classify.StatusNeedsReview
}

// sheetSize разбирает обозначение листа: «4×1250×2500», «Лист 4 мм», «лист 6».
// Толщина — первое число (мм), ширина — второе, если оно похоже на ширину
// листа (400…3000 мм). Если числа нет, толщину берём из массы 1 м² (7,85 кг/м²
// на 1 мм), ширину — 1000 мм.
func sheetSize(mark string, massPerM2 float64) (thicknessMM, widthMM float64) {
	var nums []float64
	cur := ""
	flush := func() {
		if cur == "" {
			return
		}
		if v, err := strconv.ParseFloat(strings.ReplaceAll(cur, ",", "."), 64); err == nil {
			nums = append(nums, v)
		}
		cur = ""
	}
	for _, r := range mark {
		if (r >= '0' && r <= '9') || r == ',' || r == '.' {
			cur += string(r)
			continue
		}
		flush()
	}
	flush()
	if len(nums) > 0 {
		thicknessMM = nums[0]
	}
	if len(nums) > 1 && nums[1] >= 400 && nums[1] <= 3000 {
		widthMM = nums[1]
	}
	if thicknessMM <= 0 && massPerM2 > 0 {
		thicknessMM = massPerM2 / 7.85
	}
	if widthMM <= 0 {
		widthMM = 1000
	}
	return round1(thicknessMM), widthMM
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}
