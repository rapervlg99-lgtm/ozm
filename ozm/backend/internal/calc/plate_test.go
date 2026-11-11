package calc

import (
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
