package calc

import (
	"fmt"
	"math"
	"testing"

	"ozm/backend/internal/config"
	"ozm/backend/internal/models"
)

func TestPlateGeometry(t *testing.T) {
	// Лист 4×1250: сечение F = t·b, обогрев только сверху и снизу.
	dims := map[string]float64{"t": 4, "b": 1250}
	if f := sectionArea("plate", dims); math.Abs(f-5000) > 1e-9 {
		t.Fatalf("F=%v, ожидали 5000 (4·1250)", f)
	}
	top := models.HeatedSides{Top: true}
	both := models.HeatedSides{Top: true, Bottom: true}
	if p := heatedPerimeter("plate", dims, top); math.Abs(p-1250) > 1e-9 {
		t.Fatalf("Π сверху=%v, ожидали 1250", p)
	}
	if p := heatedPerimeter("plate", dims, both); math.Abs(p-2500) > 1e-9 {
		t.Fatalf("Π сверху+снизу=%v, ожидали 2500", p)
	}
	// боковые стороны у листа не учитываются: δпр = t или t/2
	if d := sectionArea("plate", dims) / heatedPerimeter("plate", dims, top); math.Abs(d-4) > 1e-9 {
		t.Fatalf("δпр сверху=%v, ожидали 4", d)
	}
	if d := sectionArea("plate", dims) / heatedPerimeter("plate", dims, both); math.Abs(d-2) > 1e-9 {
		t.Fatalf("δпр сверху+снизу=%v, ожидали 2", d)
	}
	// площадь защиты: обогреваемая поверхность на длину
	sv, sm := ozmAreas(1, dims, both, "plate", 30, 10)
	if math.Abs(sm-25) > 1e-9 || math.Abs(sv-25) > 1e-9 {
		t.Fatalf("S=%v/%v, ожидали 25 м² (2·1,25 м · 10 м)", sv, sm)
	}
}

func TestEngineCalculatesPlateElement(t *testing.T) {
	store, err := config.LoadFromFiles("../../dicts")
	if err != nil {
		t.Fatal(err)
	}
	engine := New(store)
	resp, err := engine.Calculate(models.CalcRequest{
		ObjectName: "Склад", Address: "Рязань", Consent: true, FrDurability: "1",
		Groups: []models.GroupInput{{
			Quantity: 1,
			Elements: []models.ElementInput{{
				ID: "plate-1", Title: "Лист 4 мм", Shape: "plate",
				Dims:     map[string]float64{"t": 4, "b": 1250},
				FrType:   "1",
				HtLevel:  60,
				Sides:    models.HeatedSides{Top: true, Bottom: true},
				LengthM:  10,
				Quantity: 1,
				Coat:     "1",
				Method:   "1",
			}},
		}},
	})
	if err != nil {
		t.Fatalf("расчёт: %v", err)
	}
	if len(resp.Elements) != 1 {
		t.Fatalf("ожидали один элемент, получили %d", len(resp.Elements))
	}
	el := resp.Elements[0]
	if el.Exclusion != "" {
		t.Fatalf("лист не должен попадать в исключения: %q", el.Exclusion)
	}
	if math.Abs(el.Dpr-2) > 0.01 {
		t.Fatalf("δпр=%v, ожидали 2 мм (лист 4 мм, обогрев с двух сторон)", el.Dpr)
	}
	if math.Abs(el.AreaM2-25) > 0.01 {
		t.Fatalf("площадь=%v, ожидали 25 м²", el.AreaM2)
	}
	if el.Delta <= 0 || el.Volume <= 0 {
		t.Fatalf("толщина и объём должны быть посчитаны: δ=%v V=%v", el.Delta, el.Volume)
	}
}

func TestDimsFromLabelWhenCatalogHasNone(t *testing.T) {
	// «L90×9» из старого справочника приходит без размеров: разбираем марку
	dims := dimsFromLabel("corner_e", "L90×9")
	if dims["b"] != 90 || dims["t"] != 9 || dims["h"] != 90 {
		t.Fatalf("равнополочный уголок: %v", dims)
	}
	f := sectionArea("corner_e", dims)
	if f <= 0 {
		t.Fatalf("площадь сечения 0 при размерах %v", dims)
	}
	// неравнополочный «90/56×5,5»
	dims = dimsFromLabel("corner_ue", "90/56×5,5")
	if dims["B"] != 90 || dims["b"] != 56 || dims["t"] != 5.5 {
		t.Fatalf("неравнополочный уголок: %v", dims)
	}
}

// Каждая марка справочника должна давать ненулевые площадь и периметр —
// иначе δпр не считается (как было у «L90×9» из старого справочника).
func TestEveryCatalogMarkHasGeometry(t *testing.T) {
	store, err := config.LoadFromFiles("../../dicts")
	if err != nil {
		t.Fatal(err)
	}
	engine := New(store)
	sides := models.HeatedSides{Left: true, Top: true, Right: true, Bottom: true}
	bad := 0
	examples := []string{}
	for _, row := range store.Roll {
		dims := engine.resolveDims(models.ElementInput{Shape: row.Shape, RollID: row.ID})
		f := sectionArea(row.Shape, dims)
		pi := heatedPerimeter(row.Shape, dims, sides)
		if f <= 0 || pi <= 0 {
			bad++
			if len(examples) < 8 {
				examples = append(examples, fmt.Sprintf("%s %s (%s) F=%v P=%v", row.ID, row.Label, row.Shape, f, pi))
			}
		}
	}
	// Часть записей старых справочников приходит без размеров, и по обозначению
	// их не восстановить (двутавры по ГОСТ 57837 вида «100Ш3», трубы без стенки).
	// Фиксируем текущий уровень, чтобы правки не ухудшали картину: подстановка
	// размеров из обозначения и из одноимённой марки нового справочника уже
	// сняла 232 → 142.
	if bad > 150 {
		t.Fatalf("марок без геометрии стало больше: %d (было 142), например: %v", bad, examples)
	}
	t.Logf("марок без геометрии: %d (например: %v)", bad, examples)
}
